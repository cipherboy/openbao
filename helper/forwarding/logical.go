// Copyright (c) OpenBao, Inc.
// SPDX-License-Identifier: MPL-2.0

package forwarding

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-sockaddr"
	"github.com/openbao/openbao/sdk/helper/wrapping"
	logical "github.com/openbao/openbao/sdk/logical"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// MarshalLogicalRequest encodes a logical.Request into a LogicalRequest
// for GRPC calling.
func MarshalLogicalRequest(req *logical.Request) (*LogicalRequest, error) {
	var err error
	ret := &LogicalRequest{
		Id:        req.ID,
		Operation: string(req.Operation),
		Path:      req.Path,

		ClientToken:         req.ClientToken,
		ClientTokenAccessor: req.ClientTokenAccessor,
		DisplayName:         req.DisplayName,
		MountPoint:          req.MountPoint,
		MountType:           req.MountType,
		MountAccessor:       req.MountAccessor,

		ClientTokenRemainingUses: int32(req.ClientTokenRemainingUses),
		EntityId:                 req.EntityID,
		PolicyOverride:           req.PolicyOverride,
		Unauthenticated:          req.Unauthenticated,

		ClientTokenSource: int32(req.ClientTokenSource),

		ClientId:      req.ClientID,
		ForwardedFrom: req.ForwardedFrom,
	}

	data, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}
	ret.Data = string(data)

	ret.Secret, err = MarshalLogicalSecret(req.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request secret: %w", err)
	}

	ret.Auth, err = MarshalLogicalAuth(req.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request auth: %w", err)
	}

	ret.Headers = MarshalHeaders(req.Headers)

	if req.Connection != nil {
		ret.Connection = &LogicalConnection{
			RemoteAddr: req.Connection.RemoteAddr,
			RemotePort: int32(req.Connection.RemotePort),
		}
	}

	if req.WrapInfo != nil {
		ret.WrapInfo = &LogicalRequestWrapInfo{
			Ttl:      durationpb.New(req.WrapInfo.TTL),
			Format:   req.WrapInfo.Format,
			SealWrap: req.WrapInfo.SealWrap,
		}
	}

	mfaCreds, err := json.Marshal(req.MFACreds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request mfa credentials: %w", err)
	}
	ret.MfaCreds = string(mfaCreds)

	return ret, nil
}

func MarshalLogicalSecret(secret *logical.Secret) (*LogicalSecret, error) {
	var ret *LogicalSecret
	if secret != nil {
		secretLease := &LogicalLeaseOptions{
			Ttl:       durationpb.New(secret.TTL),
			MaxTtl:    durationpb.New(secret.MaxTTL),
			Renewable: secret.Renewable,
			Increment: durationpb.New(secret.Increment),
			IssueTime: timestamppb.New(secret.IssueTime),
		}

		internalData, err := json.Marshal(secret.InternalData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal internal data: %w", err)
		}

		ret = &LogicalSecret{
			Lease:        secretLease,
			InternalData: string(internalData),
			LeaseId:      secret.LeaseID,
		}
	}

	return ret, nil
}

func MarshalLogicalAuth(auth *logical.Auth) (*LogicalAuth, error) {
	var ret *LogicalAuth
	if auth != nil {
		authLease := &LogicalLeaseOptions{
			Ttl:       durationpb.New(auth.TTL),
			MaxTtl:    durationpb.New(auth.MaxTTL),
			Renewable: auth.Renewable,
			Increment: durationpb.New(auth.Increment),
			IssueTime: timestamppb.New(auth.IssueTime),
		}

		internalData, err := json.Marshal(auth.InternalData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal internal data: %w", err)
		}

		externalNamespacePolicies, err := json.Marshal(auth.ExternalNamespacePolicies)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal external namespace policies: %w", err)
		}

		metadata, err := json.Marshal(auth.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		var boundCidrs []string
		for index, cidr := range auth.BoundCIDRs {
			repr, err := cidr.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal bound CIDRs %d: %w", index, err)
			}

			boundCidrs = append(boundCidrs, string(repr))
		}

		var policyResults *LogicalPolicyResults
		if auth.PolicyResults != nil {
			policyResults = &LogicalPolicyResults{
				Allowed: auth.PolicyResults.Allowed,
			}

			for _, granting := range auth.PolicyResults.GrantingPolicies {
				policyResults.GrantingPolicies = append(policyResults.GrantingPolicies, &LogicalPolicyInfo{
					Name:          granting.Name,
					NamespaceId:   granting.NamespaceId,
					NamespacePath: granting.NamespacePath,
					Type:          granting.Type,
				})
			}
		}

		ret = &LogicalAuth{
			Lease:                     authLease,
			InternalData:              string(internalData),
			DisplayName:               auth.DisplayName,
			Policies:                  auth.Policies,
			TokenPolicies:             auth.TokenPolicies,
			IdentityPolicies:          auth.IdentityPolicies,
			ExternalNamespacePolicies: string(externalNamespacePolicies),
			NoDefaultPolicy:           auth.NoDefaultPolicy,
			Metadata:                  string(metadata),
			ClientToken:               auth.ClientToken,
			Accessor:                  auth.Accessor,
			Period:                    durationpb.New(auth.Period),
			ExplicitMaxTtl:            durationpb.New(auth.ExplicitMaxTTL),
			NumUses:                   int32(auth.NumUses),
			EntityId:                  auth.EntityID,
			Alias:                     auth.Alias,
			GroupAliases:              auth.GroupAliases,
			BoundCidrs:                boundCidrs,
			CreationPath:              auth.CreationPath,
			Orphan:                    auth.Orphan,
			PolicyResults:             policyResults,
			MfaRequirement:            auth.MFARequirement,
			EntityCreated:             auth.EntityCreated,
		}
	}

	return ret, nil
}

func MarshalHeaders(reqHeaders map[string][]string) map[string]*HeaderEntry {
	if len(reqHeaders) == 0 {
		return nil
	}

	var headers map[string]*HeaderEntry = make(map[string]*HeaderEntry, len(reqHeaders))
	for key, values := range reqHeaders {
		headers[key] = &HeaderEntry{
			Values: values,
		}
	}

	return headers
}

// UnmarshalLogicalRequest decodes a logical.Request from a LogicalRequest
// for GRPC calling.
func UnmarshalLogicalRequest(req *LogicalRequest) (*logical.Request, error) {
	var err error
	ret := &logical.Request{
		ID:        req.Id,
		Operation: logical.Operation(req.Operation),
		Path:      req.Path,

		ClientToken:         req.ClientToken,
		ClientTokenAccessor: req.ClientTokenAccessor,
		DisplayName:         req.DisplayName,
		MountPoint:          req.MountPoint,
		MountType:           req.MountType,
		MountAccessor:       req.MountAccessor,

		ClientTokenRemainingUses: int(req.ClientTokenRemainingUses),
		EntityID:                 req.EntityId,
		PolicyOverride:           req.PolicyOverride,
		Unauthenticated:          req.Unauthenticated,

		ClientTokenSource: logical.ClientTokenSource(req.ClientTokenSource),

		ClientID:      req.ClientId,
		ForwardedFrom: req.ForwardedFrom,
	}

	if err := json.Unmarshal([]byte(req.Data), &ret.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
	}

	ret.Secret, err = UnmarshalLogicalSecret(req.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request secret: %w", err)
	}

	ret.Auth, err = UnmarshalLogicalAuth(req.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request auth: %w", err)
	}

	ret.Headers = UnmarshalHeaders(req.Headers)

	if req.Connection != nil {
		ret.Connection = &logical.Connection{
			RemoteAddr: req.Connection.RemoteAddr,
			RemotePort: int(req.Connection.RemotePort),
		}
	}

	if req.WrapInfo != nil {
		ret.WrapInfo = &logical.RequestWrapInfo{
			TTL:      req.WrapInfo.Ttl.AsDuration(),
			Format:   req.WrapInfo.Format,
			SealWrap: req.WrapInfo.SealWrap,
		}
	}

	if err := json.Unmarshal([]byte(req.MfaCreds), &ret.MFACreds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mfa creds: %w", err)
	}

	return ret, nil
}

func UnmarshalLogicalSecret(secret *LogicalSecret) (*logical.Secret, error) {
	var ret *logical.Secret
	if secret != nil {
		ret = &logical.Secret{
			LeaseOptions: logical.LeaseOptions{
				TTL:       secret.Lease.Ttl.AsDuration(),
				MaxTTL:    secret.Lease.MaxTtl.AsDuration(),
				Renewable: secret.Lease.Renewable,
				Increment: secret.Lease.Increment.AsDuration(),
				IssueTime: secret.Lease.IssueTime.AsTime(),
			},
			LeaseID: secret.LeaseId,
		}

		if err := json.Unmarshal([]byte(secret.InternalData), &ret.InternalData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal internal data: %w", err)
		}
	}

	return ret, nil
}

func UnmarshalLogicalAuth(auth *LogicalAuth) (*logical.Auth, error) {
	var ret *logical.Auth

	if auth != nil {
		ret = &logical.Auth{
			LeaseOptions: logical.LeaseOptions{
				TTL:       auth.Lease.Ttl.AsDuration(),
				MaxTTL:    auth.Lease.MaxTtl.AsDuration(),
				Renewable: auth.Lease.Renewable,
				Increment: auth.Lease.Increment.AsDuration(),
				IssueTime: auth.Lease.IssueTime.AsTime(),
			},
			DisplayName:      auth.DisplayName,
			Policies:         auth.Policies,
			TokenPolicies:    auth.TokenPolicies,
			IdentityPolicies: auth.IdentityPolicies,
			NoDefaultPolicy:  auth.NoDefaultPolicy,
			ClientToken:      auth.ClientToken,
			Accessor:         auth.Accessor,
			Period:           auth.Period.AsDuration(),
			ExplicitMaxTTL:   auth.ExplicitMaxTtl.AsDuration(),
			NumUses:          int(auth.NumUses),
			EntityID:         auth.EntityId,
			Alias:            auth.Alias,
			GroupAliases:     auth.GroupAliases,
			CreationPath:     auth.CreationPath,
			Orphan:           auth.Orphan,
			MFARequirement:   auth.MfaRequirement,
			EntityCreated:    auth.EntityCreated,
		}

		if err := json.Unmarshal([]byte(auth.InternalData), &ret.InternalData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal internal data: %w", err)
		}

		if err := json.Unmarshal([]byte(auth.ExternalNamespacePolicies), &ret.ExternalNamespacePolicies); err != nil {
			return nil, fmt.Errorf("failed to unmarshal external namespace policies: %w", err)
		}

		if err := json.Unmarshal([]byte(auth.Metadata), &ret.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		ret.BoundCIDRs = make([]*sockaddr.SockAddrMarshaler, len(auth.BoundCidrs))
		for index := range ret.BoundCIDRs {
			ret.BoundCIDRs[index] = &sockaddr.SockAddrMarshaler{}
			if err := ret.BoundCIDRs[index].UnmarshalJSON([]byte(auth.BoundCidrs[index])); err != nil {
				return nil, fmt.Errorf("failed to unmarshal bound CIDRs %d: %w", index, err)
			}
		}

		if auth.PolicyResults != nil {
			ret.PolicyResults = &logical.PolicyResults{
				Allowed: auth.PolicyResults.Allowed,
			}

			for _, granting := range auth.PolicyResults.GrantingPolicies {
				ret.PolicyResults.GrantingPolicies = append(ret.PolicyResults.GrantingPolicies, logical.PolicyInfo{
					Name:          granting.Name,
					NamespaceId:   granting.NamespaceId,
					NamespacePath: granting.NamespacePath,
					Type:          granting.Type,
				})
			}
		}
	}

	return ret, nil
}

func UnmarshalHeaders(reqHeaders map[string]*HeaderEntry) map[string][]string {
	if len(reqHeaders) == 0 {
		return nil
	}

	var headers map[string][]string = make(map[string][]string, len(reqHeaders))
	for key, entry := range reqHeaders {
		headers[key] = entry.Values
	}

	return headers
}

// MarshalLogicalResponse encodes a logical.Response into a LogicalResponse
// for GRPC calling.
func MarshalLogicalResponse(resp *logical.Response) (*LogicalResponse, error) {
	var err error
	ret := &LogicalResponse{
		Redirect: resp.Redirect,
		Warnings: resp.Warnings,
	}

	ret.Secret, err = MarshalLogicalSecret(resp.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response secret: %w", err)
	}

	ret.Auth, err = MarshalLogicalAuth(resp.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response auth: %w", err)
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}
	ret.Data = string(data)

	if resp.WrapInfo != nil {
		ret.WrapInfo = &LogicalResponseWrapInfo{
			Ttl:             durationpb.New(resp.WrapInfo.TTL),
			Token:           resp.WrapInfo.Token,
			Accessor:        resp.WrapInfo.Accessor,
			CreationTime:    timestamppb.New(resp.WrapInfo.CreationTime),
			WrappedAccessor: resp.WrapInfo.WrappedAccessor,
			WrappedEntityId: resp.WrapInfo.WrappedEntityID,
			Format:          resp.WrapInfo.Format,
			CreationPath:    resp.WrapInfo.CreationPath,
			SealWrap:        resp.WrapInfo.SealWrap,
		}
	}

	ret.Headers = MarshalHeaders(resp.Headers)

	return ret, nil
}

// UnmarshalLogicalResponse decodes a logical.Response from a LogicalResponse
// for GRPC calling.
func UnmarshalLogicalResponse(resp *LogicalResponse) (*logical.Response, error) {
	var err error
	ret := &logical.Response{
		Redirect: resp.Redirect,
		Warnings: resp.Warnings,
	}

	ret.Secret, err = UnmarshalLogicalSecret(resp.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response secret: %w", err)
	}

	ret.Auth, err = UnmarshalLogicalAuth(resp.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response auth: %w", err)
	}

	if err := json.Unmarshal([]byte(resp.Data), &ret.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	if resp.WrapInfo != nil {
		ret.WrapInfo = &wrapping.ResponseWrapInfo{
			TTL:             resp.WrapInfo.Ttl.AsDuration(),
			Token:           resp.WrapInfo.Token,
			Accessor:        resp.WrapInfo.Accessor,
			CreationTime:    resp.WrapInfo.CreationTime.AsTime(),
			WrappedAccessor: resp.WrapInfo.WrappedAccessor,
			WrappedEntityID: resp.WrapInfo.WrappedEntityId,
			Format:          resp.WrapInfo.Format,
			CreationPath:    resp.WrapInfo.CreationPath,
			SealWrap:        resp.WrapInfo.SealWrap,
		}
	}

	ret.Headers = UnmarshalHeaders(resp.Headers)

	return ret, nil
}
