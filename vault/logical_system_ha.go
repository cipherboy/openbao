// Copyright (c) 2024 OpenBao a Series of LF Projects, LLC
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vault

import (
	"context"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// haStoragePaths returns paths for use when the storage mechanism supports
// HA mode.
func (b *SystemBackend) haStoragePaths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "storage/ha/voter",

			Fields: map[string]*framework.FieldSchema{
				"voter": {
					Type: framework.TypeBool,
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleHAReadVoter(),
					Summary:  "Gets the HA status of this node.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleHAWriteVoter(),
					Summary:  "Sets the HA status of this node.",
				},
			},

			HelpSynopsis:    strings.TrimSpace(sysHAHelp["ha-voter"][0]),
			HelpDescription: strings.TrimSpace(sysHAHelp["ha-voter"][1]),
		},
	}
}

func (b *SystemBackend) handleHAReadVoter() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		return nil, nil
	}
}

func (b *SystemBackend) handleHAWriteVoter() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		return nil, nil
	}
}

var sysHAHelp = map[string][2]string{
	"ha-voter": {
		"Returns or modifies information about voting status of this node.",
		"",
	},
}
