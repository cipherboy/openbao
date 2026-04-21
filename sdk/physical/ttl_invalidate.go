// Copyright (c) 2024 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0
// This file was completely removed in a prior commit and entirely
// new contents added to it.

package physical

import (
	"context"
	"path"
	"slices"
	"sync"
	"time"

	log "github.com/hashicorp/go-hclog"
	metrics "github.com/hashicorp/go-metrics/compat"
	"zgo.at/zcache/v2"
)

const DefaultInvalidationTTL = 15 * time.Second

type ttlInvalidateListArgs struct {
	prefix string
	after  string
	limit  int
}

type ttlInvalidate struct {
	backend    Backend
	ttl        time.Duration
	logger     log.Logger
	cache      *zcache.Cache[string, struct{}]
	listCache  *zcache.Cache[ttlInvalidateListArgs, []string]
	hookLock   sync.RWMutex
	hook       InvalidateFunc
	metricSink metrics.MetricSink
}

type transactionalTtlInvalidate struct {
	*ttlInvalidate
}

type ttlInvalidateTransaction struct {
	parent      *transactionalTtlInvalidate
	txn         Transaction
	readLock    sync.Mutex
	readEntries map[string]struct{}
	listEntries map[string]map[ttlInvalidateListArgs][]string
}

func NewTTLInvalidation(b Backend, ttl time.Duration, logger log.Logger, metricSink metrics.MetricSink) Backend {
	if _, ok := b.(CacheInvalidationBackend); ok {
		return b
	}

	if ttl <= 0 {
		ttl = DefaultInvalidationTTL
	}

	cache := zcache.New[string, struct{}](ttl, 50*time.Millisecond)
	listCache := zcache.New[ttlInvalidateListArgs, []string](ttl*2, 100*time.Millisecond)

	t := &ttlInvalidate{
		backend:    b,
		ttl:        ttl,
		cache:      cache,
		listCache:  listCache,
		logger:     logger,
		metricSink: metricSink,
	}

	cache.OnEvicted(func(path string, value struct{}) {
		t.HandleEviction(path)
	})

	listCache.OnEvicted(func(call ttlInvalidateListArgs, results []string) {
		t.HandleListEviction(call, results)
	})

	if _, ok := b.(TransactionalBackend); ok {
		return &transactionalTtlInvalidate{
			t,
		}
	}

	return t
}

func (t *ttlInvalidate) HookInvalidate(hook InvalidateFunc) {
	t.hookLock.Lock()
	defer t.hookLock.Unlock()
	t.hook = hook
}

func (t *ttlInvalidate) HandleEviction(path string) {
	t.hookLock.RLock()
	defer t.hookLock.RUnlock()

	if t.hook != nil {
		t.hook(path)
	}
}

func (t *ttlInvalidate) HandleListEviction(call ttlInvalidateListArgs, callResults []string) {
	ctx, cancel := context.WithTimeout(context.Background(), t.ttl)
	defer cancel()

	results, err := t.backend.ListPage(ctx, call.prefix, call.after, call.limit)
	if err != nil {
		t.logger.Error("failed to handle invalidation of list", "prefix", call.prefix, "after", call.after, "limit", call.limit, "error", err)
		return
	}

	if t.hook != nil {
		for _, suffix := range results {
			if !slices.Contains(callResults, suffix) {
				t.logger.Trace("handling invalidation of list: got new path", "prefix", call.prefix, "after", call.after, "limit", call.limit, "new", suffix, "previously", len(callResults), "now", len(results))

				fullpath := path.Join(call.prefix, suffix)
				t.trackRead(fullpath)
			}
		}

		for _, suffix := range callResults {
			if !slices.Contains(results, suffix) {
				t.logger.Trace("handling invalidation of list: got deleted path", "prefix", call.prefix, "after", call.after, "limit", call.limit, "new", suffix, "previously", len(callResults), "now", len(results))

				fullpath := path.Join(call.prefix, suffix)
				t.trackRead(fullpath)
			}
		}
	}
}

func (t *ttlInvalidate) trackRead(path string) {
	t.cache.GetOrAdd(path, struct{}{})
}

func (t *ttlInvalidate) trackList(prefix string, after string, limit int, results []string) {
	t.logger.Trace("tracking list for future invalidation", "prefix", prefix, "after", after, "limit", limit, "results", len(results))
	t.listCache.GetOrAdd(ttlInvalidateListArgs{
		prefix: prefix,
		after:  after,
		limit:  limit,
	}, results)
}

func (t *ttlInvalidate) Put(ctx context.Context, entry *Entry) error {
	return t.backend.Put(ctx, entry)
}

func (t *ttlInvalidate) Get(ctx context.Context, key string) (*Entry, error) {
	t.trackRead(key)
	return t.backend.Get(ctx, key)
}

func (t *ttlInvalidate) Delete(ctx context.Context, key string) error {
	return t.backend.Delete(ctx, key)
}

func (t *ttlInvalidate) List(ctx context.Context, prefix string) ([]string, error) {
	results, err := t.backend.List(ctx, prefix)
	if err == nil {
		t.trackList(prefix, "", -1, results)
	}

	return results, err
}

func (t *ttlInvalidate) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	results, err := t.backend.ListPage(ctx, prefix, after, limit)
	if err == nil {
		t.trackList(prefix, after, limit, results)
	}

	return results, err
}

func (t *transactionalTtlInvalidate) BeginReadOnlyTx(ctx context.Context) (Transaction, error) {
	txn, err := t.backend.(TransactionalBackend).BeginReadOnlyTx(ctx)
	if err != nil {
		return nil, err
	}

	return &ttlInvalidateTransaction{
		parent:      t,
		txn:         txn,
		readEntries: map[string]struct{}{},
		listEntries: map[string]map[ttlInvalidateListArgs][]string{},
	}, nil
}

func (t *transactionalTtlInvalidate) BeginTx(ctx context.Context) (Transaction, error) {
	txn, err := t.backend.(TransactionalBackend).BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	return &ttlInvalidateTransaction{
		parent:      t,
		txn:         txn,
		readEntries: map[string]struct{}{},
		listEntries: map[string]map[ttlInvalidateListArgs][]string{},
	}, nil
}

func (t *ttlInvalidateTransaction) Put(ctx context.Context, entry *Entry) error {
	return t.txn.Put(ctx, entry)
}

func (t *ttlInvalidateTransaction) Get(ctx context.Context, key string) (*Entry, error) {
	ret, err := t.txn.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	t.readLock.Lock()
	defer t.readLock.Unlock()
	t.readEntries[key] = struct{}{}

	return ret, nil
}

func (t *ttlInvalidateTransaction) Delete(ctx context.Context, key string) error {
	return t.txn.Delete(ctx, key)
}

func (t *ttlInvalidateTransaction) trackList(prefix string, after string, limit int, results []string) {
	t.readLock.Lock()
	defer t.readLock.Unlock()

	var found bool
	for entry, entryResults := range t.listEntries[prefix] {
		if entry.after == after && entry.limit == limit && slices.Equal(entryResults, results) {
			found = true
			break
		}
	}

	if !found {
		if t.listEntries[prefix] == nil {
			t.listEntries[prefix] = map[ttlInvalidateListArgs][]string{}
		}

		t.listEntries[prefix][ttlInvalidateListArgs{
			after: after,
			limit: limit,
		}] = results
	}
}

func (t *ttlInvalidateTransaction) List(ctx context.Context, prefix string) ([]string, error) {
	results, err := t.txn.List(ctx, prefix)
	if err == nil {
		t.trackList(prefix, "", -1, results)
	}

	return results, err
}

func (t *ttlInvalidateTransaction) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	results, err := t.txn.ListPage(ctx, prefix, after, limit)
	if err == nil {
		t.trackList(prefix, after, limit, results)
	}

	return results, err
}

func (t *ttlInvalidateTransaction) Commit(ctx context.Context) error {
	if err := t.txn.Commit(ctx); err != nil {
		return err
	}

	t.handleTracking()
	return nil
}

func (t *ttlInvalidateTransaction) Rollback(ctx context.Context) error {
	if err := t.txn.Rollback(ctx); err != nil {
		return err
	}

	t.handleTracking()
	return nil
}

func (t *ttlInvalidateTransaction) handleTracking() {
	t.readLock.Lock()
	defer t.readLock.Unlock()

	for key := range t.readEntries {
		t.parent.trackRead(key)
	}

	for prefix, args := range t.listEntries {
		for arg, results := range args {
			t.parent.trackList(prefix, arg.after, arg.limit, results)
		}
	}
}
