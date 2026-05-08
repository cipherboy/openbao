package barrier

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync"

	"github.com/openbao/openbao/sdk/v2/helper/keysutil"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const dataPrefix = "data"
const policyPrefix = "policy/"
const policySuffix = "keys"
const policyPath = policyPrefix + policySuffix

type EncryptedPathsBarrier struct {
	SecurityBarrier

	l           sync.RWMutex
	policy      *keysutil.Policy
	wrapper     *keysutil.EncryptedKeyStorageWrapper
	passthrough logical.Storage
}

var _ SecurityBarrier = &EncryptedPathsBarrier{}

type TransactionalEncryptedPathsBarrier struct {
	*EncryptedPathsBarrier
}

var _ SecurityBarrier = &TransactionalEncryptedPathsBarrier{}
var _ logical.TransactionalStorage = &TransactionalEncryptedPathsBarrier{}

func NewEncryptedPathsBarrier(storage SecurityBarrier) SecurityBarrier {
	barrier := &EncryptedPathsBarrier{
		SecurityBarrier: storage,
	}

	if _, ok := storage.(logical.TransactionalStorage); !ok {
		return barrier
	}

	return &TransactionalEncryptedPathsBarrier{
		barrier,
	}
}

func (e *EncryptedPathsBarrier) initialize(ctx context.Context) error {
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

func (e *EncryptedPathsBarrier) Initialized(ctx context.Context) (bool, error) {
	ok, err := e.SecurityBarrier.Initialized(ctx)
	if err != nil {
		return false, err
	}

	e.l.RLock()
	defer e.l.RUnlock()

	fmt.Fprintf(os.Stderr, "called initialized: ok = %v / wrapper = %v / result = %v\n", ok, e.wrapper != nil, ok && e.wrapper != nil)

	return ok && e.wrapper != nil, nil
}

func (e *EncryptedPathsBarrier) Unseal(ctx context.Context, key []byte) error {
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

func (e *EncryptedPathsBarrier) readLockAndMaybeInitialize(ctx context.Context) (func(), error) {
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

func (e *EncryptedPathsBarrier) List(ctx context.Context, prefix string) ([]string, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.List(ctx, prefix)
}

func (e *EncryptedPathsBarrier) ListPage(ctx context.Context, prefix string, after string, limit int) ([]string, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.ListPage(ctx, prefix, after, limit)
}

func (e *EncryptedPathsBarrier) Get(ctx context.Context, key string) (*logical.StorageEntry, error) {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return e.passthrough.Get(ctx, key)
}

func (e *EncryptedPathsBarrier) Put(ctx context.Context, entry *logical.StorageEntry) error {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	return e.passthrough.Put(ctx, entry)
}

func (e *EncryptedPathsBarrier) Delete(ctx context.Context, key string) error {
	unlock, err := e.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	return e.passthrough.Delete(ctx, key)
}

func (t *TransactionalEncryptedPathsBarrier) BeginTx(ctx context.Context) (logical.Transaction, error) {
	unlock, err := t.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return t.passthrough.(logical.TransactionalStorage).BeginTx(ctx)
}

func (t *TransactionalEncryptedPathsBarrier) BeginReadOnlyTx(ctx context.Context) (logical.Transaction, error) {
	unlock, err := t.readLockAndMaybeInitialize(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return t.passthrough.(logical.TransactionalStorage).BeginReadOnlyTx(ctx)
}
