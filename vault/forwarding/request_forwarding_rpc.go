// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package forwarding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	metrics "github.com/hashicorp/go-metrics/compat"
	"github.com/openbao/openbao/helper/forwarding"
	"github.com/openbao/openbao/physical/raft"
	"github.com/openbao/openbao/sdk/v2/helper/consts"
	"google.golang.org/grpc"
)

type forwardedRequestRPCServer struct {
	UnimplementedRequestForwardingServer

	core                         core
	handler                      http.Handler
	raftFollowerStates           *raft.FollowerStates
	clusterPeerClusterAddrsCache clusterPeerClusterAddrsCache
}

func (s *forwardedRequestRPCServer) ForwardRequest(ctx context.Context, freq *forwarding.Request) (*forwarding.Response, error) {
	// Parse an http.Request out of it
	req, err := forwarding.ParseForwardedRequest(freq)
	if err != nil {
		return nil, err
	}

	// A very dummy response writer that doesn't follow normal semantics, just
	// lets you write a status code (last written wins) and a body. But it
	// meets the interface requirements.
	w := forwarding.NewRPCResponseWriter()

	resp := &forwarding.Response{}

	runRequest := func() {
		defer func() {
			if err := recover(); err != nil {
				s.core.Logger().Error("panic serving forwarded request", "path", req.URL.Path, "error", err, "stacktrace", string(debug.Stack()))
			}
		}()
		s.handler.ServeHTTP(w, req)
	}
	runRequest()
	resp.StatusCode = uint32(w.StatusCode())
	resp.Body = w.Body().Bytes()

	header := w.Header()
	if header != nil {
		resp.HeaderEntries = make(map[string]*forwarding.HeaderEntry, len(header))
		for k, v := range header {
			resp.HeaderEntries[k] = &forwarding.HeaderEntry{
				Values: v,
			}
		}
	}

	return resp, nil
}

type NodeHAConnectionInfo struct {
	NodeInfo       *NodeInformation
	LastHeartbeat  time.Time
	Version        string
	UpgradeVersion string
}

func (s *forwardedRequestRPCServer) Echo(ctx context.Context, in *EchoRequest) (*EchoReply, error) {
	incomingNodeConnectionInfo := NodeHAConnectionInfo{
		NodeInfo:       in.NodeInfo,
		LastHeartbeat:  time.Now(),
		Version:        in.SdkVersion,
		UpgradeVersion: in.RaftUpgradeVersion,
	}
	if in.ClusterAddr != "" {
		s.clusterPeerClusterAddrsCache.Set(in.ClusterAddr, incomingNodeConnectionInfo)
	}

	if in.RaftAppliedIndex > 0 && len(in.RaftNodeID) > 0 && s.raftFollowerStates != nil {
		s.raftFollowerStates.Update(&raft.EchoRequestUpdate{
			NodeID:          in.RaftNodeID,
			AppliedIndex:    in.RaftAppliedIndex,
			Term:            in.RaftTerm,
			DesiredSuffrage: in.RaftDesiredSuffrage,
			SDKVersion:      in.SdkVersion,
			UpgradeVersion:  in.RaftUpgradeVersion,
		})
	}

	reply := &EchoReply{
		Message:          "pong",
		ReplicationState: uint32(s.core.ReplicationState()),
	}

	if raftBackend := s.core.GetRaftBackend(); raftBackend != nil {
		reply.RaftAppliedIndex = raftBackend.AppliedIndex()
		reply.RaftNodeID = raftBackend.NodeID()
	}

	return reply, nil
}

func (s *forwardedRequestRPCServer) StartInvalidations(ctx context.Context, req *StartInvalidationRequest) (*StartInvalidationResponse, error) {
	index, err := s.core.MarkPeerStated(ctx, req.Uuid)

	var errMsg string
	if err != nil {
		s.core.Logger().Error("invalidation: failed to mark peer as started", "err", err)
		errMsg = "failed to start invalidations; check active logs for more info"
	}

	return &StartInvalidationResponse{
		Err:   errMsg,
		Index: index,
	}, nil
}

func (s *forwardedRequestRPCServer) CheckInvalidations(req *CheckInvalidationRequest, stream grpc.ServerStreamingServer[CheckInvalidationResponse]) error {
	uuid, stopCh, err := s.core.AddInvalidationPeer(stream)
	if err != nil {
		s.core.Logger().Error("invalidation: failed registering invalidation peer", "err", err)
		return fmt.Errorf("not registered; check active server's logs for information")
	}

	// Send an initial response with the uuid.
	stream.Send(&CheckInvalidationResponse{
		Uuid: uuid,
	})

	select {
	case <-stopCh:
	}

	s.core.Logger().Trace("invalidation: finishing invalidation handling for server", "uuid", uuid)

	return nil
}

type Client struct {
	RequestForwardingClient
	core        core
	echoTicker  *time.Ticker
	echoContext context.Context

	invalidationsContext       context.Context
	invalidationsContextCancel context.CancelFunc
	peerUUID                   atomic.Pointer[string]
}

func NewClient(core core, requestForwardingClient RequestForwardingClient, echoTicker *time.Ticker, echoContext context.Context) *Client {
	return &Client{
		RequestForwardingClient: requestForwardingClient,
		core:                    core,
		echoTicker:              echoTicker,
		echoContext:             echoContext,
	}
}

// NOTE: we also take advantage of gRPC's keepalive bits, but as we send data
// with these requests it's useful to keep this as well
func (c *Client) StartHeartbeat() {
	go func() {
		clusterAddr := c.core.ClusterAddr()
		hostname, _ := os.Hostname()
		ni := NodeInformation{
			ApiAddr:  c.core.RedirectAddr(),
			Hostname: hostname,
			Mode:     "standby",
		}
		tick := func() {
			labels := make([]metrics.Label, 0, 1)
			now := time.Now()

			req := &EchoRequest{
				Message:     "ping",
				ClusterAddr: clusterAddr,
				NodeInfo:    &ni,
				SdkVersion:  c.core.EffectiveSDKVersion(),
			}

			if raftBackend := c.core.GetRaftBackend(); raftBackend != nil {
				req.RaftAppliedIndex = raftBackend.AppliedIndex()
				req.RaftNodeID = raftBackend.NodeID()
				req.RaftTerm = raftBackend.Term()
				req.RaftDesiredSuffrage = raftBackend.DesiredSuffrage()
				req.RaftUpgradeVersion = raftBackend.EffectiveVersion()
				labels = append(labels, metrics.Label{Name: "peer_id", Value: raftBackend.NodeID()})
			}
			defer metrics.MeasureSinceWithLabels([]string{"ha", "rpc", "client", "echo"}, now, labels)

			ctx, cancel := context.WithTimeout(c.echoContext, 2*time.Second)
			resp, err := c.Echo(ctx, req)
			cancel()
			if err != nil {
				metrics.IncrCounter([]string{"ha", "rpc", "client", "echo", "errors"}, 1)
				c.core.Logger().Debug("forwarding: error sending echo request to active node", "error", err)
				return
			}
			if resp == nil {
				c.core.Logger().Debug("forwarding: empty echo response from active node")
				return
			}
			if resp.Message != "pong" {
				c.core.Logger().Debug("forwarding: unexpected echo response from active node", "message", resp.Message)
				return
			}
			// Store the active node's replication state to display in
			// sys/health calls
			c.core.SetActiveNodeReplicationState(consts.ReplicationState(resp.ReplicationState))
		}

		tick()

		for {
			select {
			case <-c.echoContext.Done():
				c.echoTicker.Stop()
				c.core.Logger().Debug("forwarding: stopping heartbeating")
				c.core.SetActiveNodeReplicationState(consts.ReplicationUnknown)
				return
			case <-c.echoTicker.C:
				tick()
			}
		}
	}()
}

func (c *Client) CheckReplicationIndex(ctx context.Context) (string, error) {
	var uuid string

	var b backoff.BackOff = backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(15*time.Millisecond),
		backoff.WithMaxInterval(1*time.Second),
		backoff.WithMaxElapsedTime(5*time.Second),
	)
	b.Reset()

	if err := backoff.Retry(func() error {
		if value := c.peerUUID.Load(); value != nil {
			uuid = *value
			return nil
		}

		return fmt.Errorf("active node has not returned a uuid")
	}, b); err != nil {
		return "", err
	}

	resp, err := c.StartInvalidations(ctx, &StartInvalidationRequest{
		Uuid: uuid,
	})
	if err != nil {
		return "", fmt.Errorf("error checking active node's replication index: %w", err)
	}

	if resp == nil {
		return "", errors.New("no replication index returned by active node")
	}

	if resp.Err != "" {
		return "", fmt.Errorf("error checking replication index: %v", resp.Err)
	}

	return resp.Index, nil
}

func (c *Client) StreamInvalidations(ctx context.Context) {
	go func() {
		c.invalidationsContext, c.invalidationsContextCancel = context.WithCancel(c.echoContext)
		for {
			select {
			case <-c.invalidationsContext.Done():
				c.core.Logger().Debug("forwarding: stopping invalidation streaming")
				return
			default:
			}

			c.core.Logger().Trace("forwarding: starting invalidation stream")

			stream, err := c.CheckInvalidations(ctx, &CheckInvalidationRequest{})
			if err != nil {
				c.core.Logger().Debug("forwarding: failed getting invalidation; restarting standby", "err", err)
				c.core.Restart()
				break
			}

			for {
				select {
				case <-c.invalidationsContext.Done():
					break
				default:
				}

				invalidation, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						c.core.Logger().Debug("forwarding: failed processing invalidation; restarting standby", "err", err)
						break
					}

					c.core.Logger().Debug("forwarding: failed processing invalidation; restarting standby", "err", err)
					c.core.Restart()
					break
				}

				if c.peerUUID.Load() == nil {
					c.peerUUID.Store(&invalidation.Uuid)
				}

				c.core.Logger().Trace("forwarding: invalidating keys", "index", invalidation.Index, "keys", invalidation.Keys, "restart", invalidation.Restart, "uuid", invalidation.Uuid)
				if invalidation.Restart {
					c.core.Logger().Debug("forwarding: server indicated restart")
					c.core.Restart()
					break
				}

				err = c.core.AwaitInvalidation(ctx, invalidation.Index, invalidation.Keys...)
				if err != nil {
					c.core.Logger().Debug("forwarding: failed awaiting invalidation; restarting standby", "err", err)
					c.core.Restart()
					break
				}
			}
		}
	}()
}

func (c *Client) StopInvalidations() {
	if c == nil || c.invalidationsContextCancel == nil {
		return
	}

	c.invalidationsContextCancel()
}
