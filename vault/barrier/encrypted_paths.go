package barrier

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/openbao/openbao/sdk/v2/helper/keysutil"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	dataPrefix   = "data"
	policyPrefix = "policy/"
	policySuffix = "keys"
	policyPath   = policyPrefix + policySuffix
)

type EncryptedPathsBarrier interface {
	Derive(storage SecurityBarrier) SecurityBarrier
}

type encryptedPathsBarrier struct {
	SecurityBarrier

	l           sync.RWMutex
	policy      *keysutil.Policy
	wrapper     *keysutil.EncryptedKeyStorageWrapper
	passthrough logical.Storage
}

var (
	_ SecurityBarrier       = &encryptedPathsBarrier{}
	_ EncryptedPathsBarrier = &encryptedPathsBarrier{}
)

type transactionalEncryptedPathsBarrier struct {
	*encryptedPathsBarrier
}

var (
	_ SecurityBarrier              = &transactionalEncryptedPathsBarrier{}
	_ EncryptedPathsBarrier        = &transactionalEncryptedPathsBarrier{}
	_ logical.TransactionalStorage = &transactionalEncryptedPathsBarrier{}
)

func NewEncryptedPathsBarrier(storage SecurityBarrier) SecurityBarrier {
	barrier := &encryptedPathsBarrier{
		SecurityBarrier: storage,
	}

	if _, ok := storage.(logical.TransactionalStorage); !ok {
		return barrier
	}

	return &transactionalEncryptedPathsBarrier{
		barrier,
	}
}

func (e *encryptedPathsBarrier) initialize(ctx context.Context) error {
	if e.wrapper != nil {
		return nil
	}

	policy, err := keysutil.LoadPolicy(ctx, e.SecurityBarrier, policyPath)
	if err != nil {
		return fmt.Errorf("error loading encrypted key wrapper policy: %w", err)
	}

	if policy == nil {
		policy = keysutil.NewPolicy(keysutil.PolicyConfig{
			Name:                 policySuffix,
			Type:                 keysutil.KeyType_AES256_GCM96,
			Derived:              true,
			KDF:                  keysutil.Kdf_hkdf_sha256,
			ConvergentEncryption: true,
			StoragePrefix:        "",
			VersionTemplate:      keysutil.EncryptedKeyPolicyVersionTpl,
		})

		err = policy.Rotate(ctx, e.SecurityBarrier, rand.Reader)
		if err != nil {
			return fmt.Errorf("unable to rotate initial encrypted key wrapper policy: %w", err)
		}
	}

	wrapper, err := keysutil.NewEncryptedKeyStorageWrapper(keysutil.EncryptedKeyStorageConfig{
		Policy: policy,
		Prefix: dataPrefix,
	})
	if err != nil {
		return fmt.Errorf("unable to initialize encrypted key wrapper: %w", err)
	}

	e.policy = policy
	e.wrapper = wrapper
	e.passthrough = wrapper.Wrap(e.SecurityBarrier)

	return nil
}

func (e *encryptedPathsBarrier) Initialized(ctx context.Context) (bool, error) {
	ok, err := e.SecurityBarrier.Initialized(ctx)
	if err != nil {
		return false, err
	}

	e.l.RLock()
	defer e.l.RUnlock()

	return ok && e.wrapper != nil, nil
}

func (e *encryptedPathsBarrier) Unseal(ctx context.Context, key []byte) error {
	e.l.Lock()
	defer e.l.Unlock()

	if err := e.SecurityBarrier.Unseal(ctx, key); err != nil {
		return err
	}

	if !e.Sealed() {
		return e.initialize(ctx)
	}

	return nil
}

func (e *encryptedPathsBarrier) readLockAndMaybeInitialize(ctx context.Context) (func(), error) {
	e.l.RLock()
	initialized := e.wrapper != nil
	if initialized {
		return e.l.RUnlock, nil
	} else if e.Sealed() {
		e.l.RUnlock()
		return nil, ErrBarrierSealed
	}

	// Upgrade to a write lock.
	e.l.RUnlock()
	e.l.Lock()

	if err := e.initialize(ctx); err != nil {
		e.l.Unlock()
		return nil, err
	}

	return e.l.Unlock, nil
}

func (e *encryptedPathsBarrier) List(ctx context.Context, prefix string) ([]string, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.List(ctx, prefix)
}

func (e *encryptedPathsBarrier) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.ListPage(ctx, prefix, after, limit)
}

func (e *encryptedPathsBarrier) Get(ctx context.Context, key string) (*logical.StorageEntry, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.Get(ctx, key)
}

func (e *encryptedPathsBarrier) Put(ctx context.Context, entry *logical.StorageEntry) error {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	return e.passthrough.Put(ctx, entry)
}

func (e *encryptedPathsBarrier) Delete(ctx context.Context, key string) error {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	return e.passthrough.Delete(ctx, key)
}

func (t *transactionalEncryptedPathsBarrier) BeginTx(ctx context.Context) (logical.Transaction, error) {
	unlock, err := t.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return t.passthrough.(logical.TransactionalStorage).BeginTx(ctx)
}

func (t *transactionalEncryptedPathsBarrier) BeginReadOnlyTx(ctx context.Context) (logical.Transaction, error) {
	unlock, err := t.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return t.passthrough.(logical.TransactionalStorage).BeginReadOnlyTx(ctx)
}

type derivedEncryptedPathsBarrier struct {
	SecurityBarrier
	passthrough logical.Storage
}

type transactionalDerivedEncryptedPathsBarrier struct {
	*derivedEncryptedPathsBarrier
}

func (e *encryptedPathsBarrier) Derive(storage SecurityBarrier) SecurityBarrier {
	e.l.RLock()
	defer e.l.RUnlock()

	wrapped := e.wrapper.Wrap(storage)
	barrier := &derivedEncryptedPathsBarrier{
		SecurityBarrier: storage,
		passthrough:     wrapped,
	}

	if _, ok := storage.(logical.TransactionalStorage); !ok {
		return barrier
	}

	return &transactionalDerivedEncryptedPathsBarrier{
		barrier,
	}
}

func (e *derivedEncryptedPathsBarrier) List(ctx context.Context, prefix string) ([]string, error) {
	return e.passthrough.List(ctx, prefix)
}

func (e *derivedEncryptedPathsBarrier) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	return e.passthrough.ListPage(ctx, prefix, after, limit)
}

func (e *derivedEncryptedPathsBarrier) Get(ctx context.Context, key string) (*logical.StorageEntry, error) {
	return e.passthrough.Get(ctx, key)
}

func (e *derivedEncryptedPathsBarrier) Put(ctx context.Context, entry *logical.StorageEntry) error {
	return e.passthrough.Put(ctx, entry)
}

func (e *derivedEncryptedPathsBarrier) Delete(ctx context.Context, key string) error {
	return e.passthrough.Delete(ctx, key)
}

func (t *transactionalDerivedEncryptedPathsBarrier) BeginTx(ctx context.Context) (logical.Transaction, error) {
	return t.passthrough.(logical.TransactionalStorage).BeginTx(ctx)
}

func (t *transactionalDerivedEncryptedPathsBarrier) BeginReadOnlyTx(ctx context.Context) (logical.Transaction, error) {
	return t.passthrough.(logical.TransactionalStorage).BeginReadOnlyTx(ctx)
}
