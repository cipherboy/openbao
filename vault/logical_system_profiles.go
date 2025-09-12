// Copyright (c) 2025 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package vault

import (
	"context"
	"net/http"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func (b *SystemBackend) profilePaths() []*framework.Path {
	profileListSchema := map[string]*framework.FieldSchema{
		"keys": {
			Type:        framework.TypeStringSlice,
			Description: "List of profile paths.",
		},
		"key_info": {
			Type:        framework.TypeMap,
			Description: "Map of profile details by path.",
		},
	}

	profileSchema := map[string]*framework.FieldSchema{
		"description": {
			Type:        framework.TypeString,
			Required:    true,
			Description: "Profile description.",
		},
		"profile": {
			Type:        framework.TypeString,
			Required:    true,
			Description: "Profile definition in HCL or JSON.",
		},
		"version": {
			Type:        framework.TypeInt,
			Required:    true,
			Description: "Version of the profile.",
		},
		"cas_required": {
			Type:        framework.TypeBool,
			Required:    true,
			Description: "Whether check and set support is required.",
		},
		"allow_unauthenticated": {
			Type:        framework.TypeBool,
			Required:    true,
			Description: "Whether this profile can be accessed unauthenticated.",
		},
	}

	paths := []*framework.Path{
		{
			Pattern: "profiles/manage/?$",

			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "profiles",
			},

			Fields: map[string]*framework.FieldSchema{
				"after": {
					Type:        framework.TypeString,
					Description: "Optional entry to begin listing after; not required to exist.",
				},
				"limit": {
					Type:        framework.TypeInt,
					Description: "Optional number of entries to return; defaults to all entries.",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleProfilesList(false),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileListSchema}},
					},
					Summary: "List profiles.",
				},
				logical.ScanOperation: &framework.PathOperation{
					Callback: b.handleProfilesList(true),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileListSchema}},
					},
					Summary: "Scan (recursively list) profiles.",
				},
			},

			HelpSynopsis:    strings.TrimSpace(sysHelp["list-profiles"][0]),
			HelpDescription: strings.TrimSpace(sysHelp["list-profiles"][1]),
		},

		{
			Pattern: "profiles/manage/(?P<path>.+)",

			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "profiles",
			},

			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Path of the profile.",
				},
				"description": {
					Type:        framework.TypeString,
					Description: "Profile description.",
				},
				"profile": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Profile definition in either HCL or JSON format.",
				},
				"cas": {
					Type:        framework.TypeInt,
					Description: "Check and set version of the profile.",
				},
				"cas_required": {
					Type:        framework.TypeBool,
					Description: "Whether to require check and set for modifying this profile.",
				},
				"allow_unauthenticated": {
					Type:        framework.TypeBool,
					Description: "Whether this profile can be executed unauthenticated. Use with care.",
				},
				"after": {
					Type:        framework.TypeString,
					Description: "Optional entry to begin listing after; not required to exist.",
				},
				"limit": {
					Type:        framework.TypeInt,
					Description: "Optional number of entries to return; defaults to all entries.",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleProfilesRead(),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileSchema}},
					},
					Summary: "Retrieve a profile.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleProfilesUpdate(),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileSchema}},
					},
					Summary: "Create or update a profile.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleProfilesDelete(),
					Responses: map[int][]framework.Response{
						http.StatusNoContent: {{Description: "No Content"}},
					},
					Summary: "Delete a profile.",
				},
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleProfilesList(false),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileListSchema}},
					},
					Summary: "List profiles.",
				},
				logical.ScanOperation: &framework.PathOperation{
					Callback: b.handleProfilesList(true),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK", Fields: profileListSchema}},
					},
					Summary: "Scan (recursively list) profiles.",
				},
			},

			HelpSynopsis:    strings.TrimSpace(sysHelp["profiles"][0]),
			HelpDescription: strings.TrimSpace(sysHelp["profiles"][1]),
		},

		{
			Pattern: "profiles/execute/(?P<path>.+)",

			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "profiles-execute",
			},

			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Path of the profile.",
				},
			},
			TakesArbitraryInput: true,

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleProfilesExecute(false /* we are authenticated */),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK"}},
					},
					Summary: "Execute the given profile.",
				},
			},

			HelpSynopsis:    "execute " + strings.TrimSpace(sysHelp["profiles"][0]),
			HelpDescription: "execute " + strings.TrimSpace(sysHelp["profiles"][1]),
		},
	}

	if b.Core.allowUnauthedProfiles {
		paths = append(paths, &framework.Path{
			Pattern: "profiles/unauthed-execute/(?P<path>.+)",

			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "profiles-unauthed-execute",
			},

			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Path of the profile.",
				},
			},
			TakesArbitraryInput: true,

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleProfilesExecute(true /* we are unauthenticated */),
					Responses: map[int][]framework.Response{
						http.StatusOK: {{Description: "OK"}},
					},
					Summary: "Execute the given profile without authentication.",
				},
			},

			HelpSynopsis:    strings.TrimSpace(sysHelp["profiles"][0]),
			HelpDescription: strings.TrimSpace(sysHelp["profiles"][1]),
		})
	}

	return paths
}

func createProfileListResponse(pe *ProfileEntry) map[string]any {
	return map[string]any{
		"path":                  pe.Path,
		"version":               pe.Version,
		"cas_required":          pe.CASRequired,
		"allow_unauthenticated": pe.AllowUnauthenticated,
		"description":           pe.Description,
	}
}

func createProfileDataResponse(pe *ProfileEntry) map[string]any {
	base := createProfileListResponse(pe)
	base["profile"] = pe.Profile
	return base
}

// handleProfilesList handles "/sys/profiles/manage/*" endpoints to list the
// profiles.
func (b *SystemBackend) handleProfilesList(scan bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		parent := ""
		if _, present := data.Schema["path"]; present {
			parent = data.Get("parent").(string)
		}

		after := data.Get("after").(string)
		limit := data.Get("limit").(int)

		profiles, err := b.Core.profileStore.List(ctx, parent, scan, after, limit)
		if err != nil {
			return nil, err
		}

		var keys []string
		keyInfo := make(map[string]interface{})
		for _, entry := range profiles {
			keys = append(keys, entry.Path)
			keyInfo[entry.Path] = createProfileDataResponse(entry)
		}

		return logical.ListResponseWithInfo(keys, keyInfo), nil
	}
}

// handleProfilesRead handles the "/sys/profiles/manage/<path>" endpoints to read a
// profile.
func (b *SystemBackend) handleProfilesRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		path := data.Get("path").(string)

		pe, err := b.Core.profileStore.Get(ctx, path)
		if err != nil {
			return handleError(err)
		}

		return &logical.Response{Data: createProfileDataResponse(pe)}, nil
	}
}

// handleProfileSet handles the "/sys/profiles/manage/<path>" endpoint to
// update a profile.
func (b *SystemBackend) handleProfilesUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		path := data.Get("path").(string)

		var cas *int
		casRaw, ok := data.GetOk("cas")
		if ok {
			cas := new(int)
			*cas = casRaw.(int)
		}

		profile := data.Get("profile").(string)
		description := data.Get("description").(string)
		allowUnauthenticated := data.Get("allow_unauthenticated").(bool)
		casRequired := data.Get("cas_required").(bool)

		pe := &ProfileEntry{
			Path:                 path,
			Profile:              profile,
			Description:          description,
			CASRequired:          casRequired,
			AllowUnauthenticated: allowUnauthenticated,
		}

		err := b.Core.profileStore.Set(ctx, pe, cas)
		if err != nil {
			return handleError(err)
		}

		return &logical.Response{Data: createProfileDataResponse(pe)}, nil
	}
}

// handleProfilesDelete handles the "/sys/profile/<path>" endpoint to delete a profile.
func (b *SystemBackend) handleProfilesDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		path := data.Get("path").(string)

		err := b.Core.profileStore.Delete(ctx, path)
		return nil, err
	}
}

// handleProfilesExecute handles the "/sys/profile/execute/<path>" and
// "sys/profile/unauthed-execute/<path>" endpoint to execute profiles.
func (b *SystemBackend) handleProfilesExecute(unauthed bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		path := data.Get("path").(string)

		return b.Core.profileStore.Execute(ctx, path, unauthed, req, data)
	}
}
