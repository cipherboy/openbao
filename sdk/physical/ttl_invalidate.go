// Copyright (c) 2024 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0
// This file was completely removed in a prior commit and entirely
// new contents added to it.

package physical

import (
	"context"
	"sync"
	"time"

	log "github.com/hashicorp/go-hclog"
	metrics "github.com/hashicorp/go-metrics/compat"
	"zgo.at/zcache/v2"
)

const DefaultInvalidationTTL = 15 * time.Second

type ttlInvalidate struct {
	backend    Backend
	ttl        time.Duration
	logger     log.Logger
	cache      *zcache.Cache[string, struct{}]
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
}

func NewTTLInvalidation(b Backend, ttl time.Duration, logger log.Logger, metricSink metrics.MetricSink) Backend {
	if _, ok := b.(CacheInvalidationBackend); ok {
		return b
	}

	if ttl <= 0 {
		ttl = DefaultInvalidationTTL
	}

	cache := zcache.New[string, struct{}](ttl, 50*time.Millisecond)
	t := &ttlInvalidate{
		backend:    b,
		ttl:        ttl,
		cache:      cache,
		logger:     logger,
		metricSink: metricSink,
	}

	cache.OnEvicted(func(path string, value struct{}) {
		t.HandleEviction(path)
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
	t.logger.Trace("hooking invalidation")
	t.hook = hook
}

func (t *ttlInvalidate) HandleEviction(path string) {
	t.hookLock.RLock()
	defer t.hookLock.RUnlock()

	t.logger.Trace("handling triggering invalidation", "key", path, "hook", t.hook != nil)
	if t.hook != nil {
		t.logger.Trace("dispatching invalidation", "key", path)
		t.hook(path)
	}
}

func (t *ttlInvalidate) trackRead(path string) {
	t.logger.Trace("tracking read for future invalidation", "key", path)
	t.cache.GetOrAdd(path, struct{}{})
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
	return t.backend.List(ctx, prefix)
}

func (t *ttlInvalidate) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	return t.backend.ListPage(ctx, prefix, after, limit)
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

func (t *ttlInvalidateTransaction) List(ctx context.Context, prefix string) ([]string, error) {
	return t.txn.List(ctx, prefix)
}

func (t *ttlInvalidateTransaction) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	return t.txn.ListPage(ctx, prefix, after, limit)
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
}
