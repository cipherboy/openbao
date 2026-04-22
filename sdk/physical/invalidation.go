package physical

import (
	"context"
	"maps"
	"slices"

	log "github.com/hashicorp/go-hclog"
)

type GRPCInvalidator interface{}

type grpcInvalidator struct {
	backend     Backend
	logger      log.Logger
	hook        InvalidateFunc
	notifyWrite InvalidateFunc
}

type transactionalGRPCInvalidator struct {
	*grpcInvalidator
}

type grpcInvalidatorTransaction struct {
	*grpcInvalidator
	txn    Transaction
	writes map[string]struct{}
}

func NewGRPCInvalidator(b Backend, logger log.Logger, notifyWrite InvalidateFunc) Backend {
	g := &grpcInvalidator{
		backend:     b,
		logger:      logger,
		notifyWrite: notifyWrite,
	}

	if _, ok := b.(TransactionalBackend); ok {
		return &transactionalGRPCInvalidator{
			g,
		}
	}

	return g
}

func (g *grpcInvalidator) Put(ctx context.Context, entry *Entry) error {
	err := g.backend.Put(ctx, entry)
	if err == nil {
		g.notifyWrite(entry.Key)
	}

	return err
}

func (g *grpcInvalidator) Get(ctx context.Context, key string) (*Entry, error) {
	return g.backend.Get(ctx, key)
}

func (g *grpcInvalidator) Delete(ctx context.Context, key string) error {
	err := g.backend.Delete(ctx, key)
	if err == nil {
		g.notifyWrite(key)
	}

	return err
}

func (g *grpcInvalidator) List(ctx context.Context, prefix string) ([]string, error) {
	return g.backend.List(ctx, prefix)
}

func (g *grpcInvalidator) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	return g.backend.ListPage(ctx, prefix, after, limit)
}

func (g *transactionalGRPCInvalidator) BeginReadOnlyTx(ctx context.Context) (Transaction, error) {
	txn, err := g.grpcInvalidator.backend.(TransactionalBackend).BeginReadOnlyTx(ctx)
	if err != nil {
		return nil, err
	}

	return &grpcInvalidatorTransaction{
		grpcInvalidator: g.grpcInvalidator,
		txn:             txn,
		writes:          map[string]struct{}{},
	}, nil
}

func (g *transactionalGRPCInvalidator) BeginTx(ctx context.Context) (Transaction, error) {
	txn, err := g.grpcInvalidator.backend.(TransactionalBackend).BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	return &grpcInvalidatorTransaction{
		grpcInvalidator: g.grpcInvalidator,
		txn:             txn,
		writes:          map[string]struct{}{},
	}, nil
}

func (g *grpcInvalidatorTransaction) Put(ctx context.Context, entry *Entry) error {
	err := g.txn.Put(ctx, entry)
	if err == nil {
		g.writes[entry.Key] = struct{}{}
	}

	return err
}

func (g *grpcInvalidatorTransaction) Get(ctx context.Context, key string) (*Entry, error) {
	return g.txn.Get(ctx, key)
}

func (g *grpcInvalidatorTransaction) Delete(ctx context.Context, key string) error {
	err := g.txn.Delete(ctx, key)
	if err == nil {
		g.writes[key] = struct{}{}
	}

	return err
}

func (g *grpcInvalidatorTransaction) List(ctx context.Context, prefix string) ([]string, error) {
	return g.txn.List(ctx, prefix)
}

func (g *grpcInvalidatorTransaction) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	return g.txn.ListPage(ctx, prefix, after, limit)
}

func (g *grpcInvalidatorTransaction) Commit(ctx context.Context) error {
	err := g.txn.Commit(ctx)
	if err == nil {
		g.grpcInvalidator.notifyWrite(slices.Collect(maps.Keys(g.writes))...)
	}
	return err
}

func (g *grpcInvalidatorTransaction) Rollback(ctx context.Context) error {
	return g.txn.Rollback(ctx)
}
