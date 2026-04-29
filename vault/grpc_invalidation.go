package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/openbao/openbao/sdk/v2/physical"
	"github.com/openbao/openbao/vault/forwarding"
	"google.golang.org/grpc"
	"zgo.at/zcache/v2"
)

type invalidationPeers struct {
	l     sync.Mutex
	peers *zcache.Cache[string, *invalidationPeerInfo]
	core  *Core
}

type invalidationPeerInfo struct {
	startIndex string
	stream     grpc.ServerStreamingServer[forwarding.CheckInvalidationResponse]
	stopCh     chan struct{}
	restart    bool
}

func (c *Core) NewInvalidationPeers() *invalidationPeers {
	ret := &invalidationPeers{
		peers: zcache.New[string, *invalidationPeerInfo](3600*c.clusterHeartbeatInterval, 3600*time.Second),
		core:  c,
	}

	ret.peers.OnEvicted(func(uuid string, i *invalidationPeerInfo) {
		c.logger.Trace("evicting invalidation client for peer", "uuid", uuid)
		if i.stopCh != nil {
			close(i.stopCh)
			i.stopCh = nil
		}
	})

	return ret
}

func (c *Core) CleanupInvalidationPeers() {
	if c.connectedInvalidationPeers == nil {
		return
	}

	c.connectedInvalidationPeers.l.Lock()
	defer c.connectedInvalidationPeers.l.Unlock()

	c.connectedInvalidationPeers.peers.DeleteAll()

	c.connectedInvalidationPeers = nil
}

func (c *Core) SendInvalidationNotice(keys ...string) {
	c.logger.Error("calling SendInvalidationNotice", "keys", keys)
	// Ensure we're called on the active node only.
	if c.standby.Load() {
		c.logger.Error("skipping SendInvalidationNotice on standby", "keys", keys)
		return
	}

	ctx := c.activeContext.Load()
	if ctx.Err() != nil {
		c.logger.Error("bad active context on standby", "keys", keys, "err", ctx.Err())
		return
	}

	// Get the index from the underlying storage backend.
	index, err := c.underlyingPhysical.(physical.ReplicationIndexBackend).AppliedReplicationIndex(ctx)
	if err != nil {
		c.logger.Error("failed to get latest applied replication index", "error", err)
		return
	}

	go func() {
		c.logger.Error("propagating write: waiting for state lock", "keys", keys, "index", index)
		c.stateLock.RLock()
		defer c.stateLock.RUnlock()

		if c.connectedInvalidationPeers == nil {
			c.logger.Error("skipping SendInvalidationNotice on standby: no connected invalidation peers", "keys", keys, "index", index)
			return
		}

		c.logger.Error("propagating write", "keys", keys, "index", index)
		c.connectedInvalidationPeers.propagateWrite(c.activeContext.Load(), index, keys...)
	}()
}

func (c *Core) AddInvalidationPeer(stream grpc.ServerStreamingServer[forwarding.CheckInvalidationResponse]) (string, chan struct{}, error) {
	return c.connectedInvalidationPeers.AddPeer(stream)
}

func (i *invalidationPeers) AddPeer(stream grpc.ServerStreamingServer[forwarding.CheckInvalidationResponse]) (string, chan struct{}, error) {
	i.core.logger.Trace("adding replication peer")
	defer i.core.logger.Trace("done adding replication peer")

	i.l.Lock()
	defer i.l.Unlock()

	info := &invalidationPeerInfo{
		startIndex: "",
		stream:     stream,
		stopCh:     make(chan struct{}, 1),
	}

	peerUUID, err := uuid.GenerateUUID()
	if err != nil {
		return "", nil, err
	}

	i.peers.Add(peerUUID, info)
	return peerUUID, info.stopCh, nil
}

func (i *invalidationPeers) propagateWrite(ctx context.Context, index string, keys ...string) error {
	i.core.logger.Trace("propagating write", "index", index, "keys", keys)
	defer i.core.logger.Trace("done propagating write")

	i.l.Lock()
	defer i.l.Unlock()

	var failed []string
	var retErr error
	for peerUUID, peerInfoItem := range i.peers.Items() {
		i.core.logger.Trace("propagating write to peer", "index", index, "keys", keys, "peer", peerUUID)

		peerInfo := peerInfoItem.Object
		if peerInfo.startIndex == "" {
			i.core.logger.Debug("skipping write to peer: no start index", "peer", peerUUID)
			i.peers.Touch(peerUUID)
			continue
		}

		if peerInfo.stream == nil {
			retErr = multierror.Append(retErr, fmt.Errorf("while sending invalidation to %v: no active stream", peerUUID))
			failed = append(failed, peerUUID)
			continue
		}

		err := peerInfo.stream.Send(&forwarding.CheckInvalidationResponse{
			Index:   index,
			Keys:    keys,
			Restart: peerInfo.restart,
		})
		if err != nil {
			retErr = multierror.Append(retErr, fmt.Errorf("while sending invalidation to %v: %w", peerUUID, err))
			failed = append(failed, peerUUID)
			peerInfo.restart = true
			continue
		}

		i.peers.Touch(peerUUID)
	}

	for _, failedPeer := range failed {
		i.peers.Delete(failedPeer)
	}

	return retErr
}

func (core *Core) MarkPeerStated(ctx context.Context, uuid string) (string, error) {
	core.logger.Trace("marking peer as started", "uuid", uuid)
	defer core.logger.Trace("done marking peer as started")

	indexable, ok := core.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		return "", errors.New("underlying physical backend does not expose indices but peer requested index info")
	}

	index, err := indexable.AppliedReplicationIndex(ctx)
	if err != nil {
		return "", errors.New("failed getting current physical replication index")
	}

	err = core.connectedInvalidationPeers.markPeerStarted(uuid, index)
	if err != nil {
		return "", fmt.Errorf("failed marking peer as started: %w", err)
	}

	return index, nil
}

func (i *invalidationPeers) markPeerStarted(uuid string, index string) error {
	i.core.logger.Trace("(invalidationPeers) marking peer as started", "index", index)
	defer i.core.logger.Trace("(invalidationPeers) done marking peer as started")

	i.l.Lock()
	defer i.l.Unlock()

	if _, existing := i.peers.Modify(uuid, func(peerInfo *invalidationPeerInfo) *invalidationPeerInfo {
		peerInfo.restart = false
		peerInfo.startIndex = index
		return peerInfo
	}); !existing {
		return fmt.Errorf("peer %q does not have active replication stream", uuid)
	}

	return nil
}

func (core *Core) TrackWrite(ctx context.Context, index string, keys ...string) {
	core.logger.Trace("start tracking write", "index", index)
	defer core.logger.Trace("done tracking write", "index", index)

	if err := core.connectedInvalidationPeers.propagateWrite(ctx, index, keys...); err != nil {
		core.logger.Error("failed to track write", "err", err)
	}
}

func (core *Core) AwaitInvalidation(ctx context.Context, index string, keys ...string) error {
	core.logger.Debug("awaiting invalidation", "index", index, "keys", keys)
	defer core.logger.Debug("done awaiting invalidation", "index", index, "keys", keys)

	if len(keys) == 0 || index == "" {
		return nil
	}

	replicated, ok := core.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		return fmt.Errorf("unknown type for underlying physical storage: %T expected physical.ReplicationIndexBackend", core.underlyingPhysical)
	}

	var b backoff.BackOff = backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(15*time.Millisecond),
		backoff.WithMaxInterval(1*time.Second),
		backoff.WithMaxElapsedTime(60*time.Second),
	)
	b.Reset()

	core.logger.Debug("beginning replication index check loop for invalidation", "invalidationIndex", index)
	if err := backoff.Retry(func() error {
		storageIndex, err := replicated.AppliedReplicationIndex(ctx)
		if err != nil {
			return fmt.Errorf("error checking replication index: %w", err)
		}

		passed, err := replicated.GreaterEqualReplicationIndex(ctx, storageIndex, index)
		if err != nil {
			return fmt.Errorf("failed comparing replication indices: %w", err)
		}
		if !passed {
			core.logger.Debug("current storage index was not new enough", "storageIndex", storageIndex, "invalidationIndex", index)
			return errors.New("have not reached invalidation replication index")
		}

		return nil
	}, b); err != nil {
		return fmt.Errorf("unable to catch active state during invalidation; additional errors: %w", err)
	}

	core.Invalidate(keys...)

	return nil
}

// AwaitReplication allows us to be sure that all state loaded after this
// point will have been loaded from an index, after which the active node
// will know to send us invalidations for updates. And because we already
// started tracking invalidations, any invalidations which we get will be
// queued locally.
func (core *Core) AwaitReplication(ctx context.Context) error {
	core.logger.Trace("awaiting replication")
	defer core.logger.Trace("done awaiting replication")

	core.requestForwardingConnectionLock.RLock()
	activeIndex, err := core.rpcForwardingClient.CheckReplicationIndex(ctx)
	core.requestForwardingConnectionLock.RUnlock()
	if err != nil {
		return err
	}

	replicated, ok := core.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		return fmt.Errorf("unknown type for underlying physical storage: %T expected physical.ReplicationIndexBackend", core.underlyingPhysical)
	}

	var b backoff.BackOff = backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(15*time.Millisecond),
		backoff.WithMaxInterval(1*time.Second),
		backoff.WithMaxElapsedTime(60*time.Second),
	)
	b.Reset()

	core.logger.Debug("beginning replication index check loop", "activeIndex", activeIndex)
	if err := backoff.Retry(func() error {
		index, err := replicated.AppliedReplicationIndex(ctx)
		if err != nil {
			return fmt.Errorf("error checking replication index: %w", err)
		}

		passed, err := replicated.GreaterEqualReplicationIndex(ctx, index, activeIndex)
		if err != nil {
			return fmt.Errorf("failed comparing replication indices: %w", err)
		}
		if !passed {
			core.logger.Debug("current index was not new enough", "index", index, "activeIndex", activeIndex)
			return errors.New("have not reached active node replication index")
		}

		return nil
	}, b); err != nil {
		return fmt.Errorf("unable to catch active state; additional errors: %w", err)
	}

	return nil
}
