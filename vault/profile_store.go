package vault

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/openbao/openbao/helper/namespace"
	"github.com/openbao/openbao/helper/profiles"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	profileSubPath = "profiles/"
)

type ProfileEntry struct {
	Path                 string
	Profile              string
	Description          string
	Version              int
	CASRequired          bool
	AllowUnauthenticated bool
}

func (pe *ProfileEntry) Parse(ctx context.Context) (*profiles.InputConfig, []*profiles.OuterConfig, *profiles.OutputConfig, error) {
	var input *profiles.InputConfig
	var profile []*profiles.OuterConfig
	var output *profiles.OutputConfig

	obj, err := hcl.Parse(pe.Profile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed during HCL parsing: %w", err)
	}

	list, ok := obj.Node.(*ast.ObjectList)
	if !ok {
		return nil, nil, nil, errors.New("profile doesn't contain a root object")
	}

	if o := list.Filter("input"); len(o.Items) > 0 {
		input, err = profiles.ParseInputConfig(o)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse input configuration block: %w", err)
		}
	}

	if o := list.Filter("context"); len(o.Items) > 0 {
		profile, err = profiles.ParseOuterConfig("context", o)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse profile: %w", err)
		}
	}

	if o := list.Filter("output"); len(o.Items) > 0 {
		output, err = profiles.ParseOutputConfig(o)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse output configuration block: %w", err)
		}
	}

	if len(profile) == 0 {
		return nil, nil, nil, fmt.Errorf("profile must have at least one 'context' block")
	}

	return input, profile, output, nil
}

type ProfileStore struct {
	core   *Core
	lock   sync.RWMutex
	logger log.Logger
	engine profiles.ProfileEngine
}

func NewProfileStore(c *Core) *ProfileStore {
	logger := c.baseLogger.Named("profile")
	return &ProfileStore{
		core:   c,
		logger: logger,
	}
}

func (c *Core) setupProfileStore(ctx context.Context) {
	c.profileStore = NewProfileStore(c)
}

// getView returns the storage view for the given namespace
func (ps *ProfileStore) getView(ns *namespace.Namespace) BarrierView {
	if ns.ID == namespace.RootNamespaceID {
		return ps.core.systemBarrierView.SubView(profileSubPath)
	}

	return ps.core.namespaceMountEntryView(ns, systemBarrierPrefix+profileSubPath)
}

func (ps *ProfileStore) Get(ctx context.Context, path string) (*ProfileEntry, error) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.getLocked(ctx, path)
}

func (ps *ProfileStore) getLocked(ctx context.Context, path string) (*ProfileEntry, error) {
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	path = ps.sanitizePath(path)
	view := ps.getView(ns)

	entry, err := view.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	var profile ProfileEntry
	if err := entry.DecodeJSON(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

func (ps *ProfileStore) Set(ctx context.Context, profile *ProfileEntry, casVersion *int) error {
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return err
	}

	path := ps.sanitizePath(profile.Path)
	view := ps.getView(ns)

	ps.lock.Lock()
	defer ps.lock.Unlock()

	existing, err := ps.getLocked(ctx, profile.Path)
	if err != nil {
		return err
	}

	casRequired := (existing != nil && existing.CASRequired) || profile.CASRequired
	if casVersion == nil && casRequired {
		return fmt.Errorf("check-and-set parameter required for this call")
	}
	if casVersion != nil {
		if *casVersion == -1 && existing != nil {
			return fmt.Errorf("check-and-set parameter set to -1 on existing entry")
		}

		if *casVersion != -1 && *casVersion != existing.Version {
			return fmt.Errorf("check-and-set parameter did not match the current version")
		}
	}

	entry, err := logical.StorageEntryJSON(path, profile)
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	if err := view.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

func (ps *ProfileStore) Delete(ctx context.Context, path string) error {
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return err
	}

	path = ps.sanitizePath(path)
	view := ps.getView(ns)

	ps.lock.Lock()
	defer ps.lock.Unlock()

	return view.Delete(ctx, path)
}

func (ps *ProfileStore) List(ctx context.Context, prefix string, recursive bool, after string, limit int) ([]*ProfileEntry, error) {
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	prefix = ps.sanitizePath(prefix)
	view := ps.getView(ns)
	view = view.SubView(prefix)

	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var keys []string
	if !recursive {
		keys, err = view.ListPage(ctx, "", after, limit)
	} else {
		err = logical.ScanView(ctx, view, func(path string) {
			keys = append(keys, path)
		})
	}

	if err != nil {
		return nil, err
	}

	var results []*ProfileEntry
	for index, key := range keys {
		path := filepath.Join(prefix, key)
		entry, err := ps.getLocked(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch profile (%d/%v) in list: %w", index, path, err)
		}

		results = append(results, entry)
	}

	return results, nil
}

func (ps *ProfileStore) Execute(ctx context.Context, path string, unauthed bool, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	ps.logger.Trace("here?")
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	ps.lock.RLock()
	defer ps.lock.RUnlock()

	profile, err := ps.getLocked(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to execute profile: %w", err)
	}

	// Prefer permission denied for missing profiles when unauthenticated.
	if unauthed && (profile == nil || !profile.AllowUnauthenticated) {
		ps.logger.Trace("here?")
		return nil, logical.ErrPermissionDenied
	}

	if profile == nil {
		return nil, nil
	}

	input, contents, output, err := profile.Parse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	engine, err := profiles.NewEngine(
		// Do not allow sources which could bypass authorization.
		profiles.WithRequestSource(),
		profiles.WithResponseSource(),
		profiles.WithTemplateSource(),
		profiles.WithInputSource(input, req, data),
		profiles.WithOutput(output),

		// Name of our outer block; this is called context.
		profiles.WithOuterBlockName("context"),

		// The actual profile we're trying to execute.
		profiles.WithProfile(contents),

		// Default token to use.
		profiles.WithDefaultToken(req.ClientToken),

		// Create a named logger for this, to allow operators to debug
		// profile failures.
		profiles.WithLogger(ps.logger.Named(fmt.Sprintf("%v/%v", ns.Path, profile.Path))),

		// Execute our request handler here; here is where we validate that
		// this policy can only access requests under its own namespace and
		// forbid requests to parent namespaces.
		profiles.WithRequestHandler(func(ctx context.Context, req *logical.Request) (*logical.Response, error) {
			return ps.core.HandleRequest(ctx, req)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed building profile engine: %w", err)
	}

	// Remove context from the namespace.
	ctxNoNs := namespace.ContextWithNamespace(ctx, nil)

	if output != nil {
		return engine.EvaluateResponse(ctxNoNs)
	}

	if err := engine.Evaluate(ctxNoNs); err != nil {
		return nil, fmt.Errorf("failed to evaluate namespace: %w", err)
	}

	return nil, nil
}

func (ps *ProfileStore) sanitizePath(path string) string {
	return strings.ToLower(strings.TrimSpace(path))
}
