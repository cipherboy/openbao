// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/openbao/openbao/helper/testhelpers/corehelpers"
	"github.com/openbao/openbao/sdk/logical"
	"github.com/openbao/openbao/sdk/physical"
	"github.com/openbao/openbao/sdk/physical/inmem"
	"github.com/stretchr/testify/require"
)

// mockAESBarrier returns a physical backend, security barrier, and master key
func mockXChaCha20Barrier(t testing.TB) (physical.Backend, SecurityBarrier, []byte) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)
	return inm, b, key
}

func TestXChaCha20Barrier_Basic(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	testBarrier(t, b)
}

func TestXChaCha20Barrier_Rotate(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	testBarrier_Rotate(t, b)
}

func TestXChaCha20Barrier_MissingRotateConfig(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)

	// Write a keyring which lacks rotation config settings
	oldKeyring := b.keyring.Clone()
	oldKeyring.rotationConfig = KeyRotationConfig{}
	b.persistKeyring(context.Background(), oldKeyring)

	b.ReloadKeyring(context.Background())

	// At this point, the rotation config should match the default
	if !defaultRotationConfig.Equals(b.keyring.rotationConfig) {
		t.Fatalf("expected empty rotation config to recover as default config")
	}
}

func TestXChaCha20Barrier_Upgrade(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b1, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b2, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	testBarrier_Upgrade(t, b1, b2)
}

func TestXChaCha20Barrier_Upgrade_Rekey(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b1, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b2, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	testBarrier_Upgrade_Rekey(t, b1, b2)
}

func TestXChaCha20Barrier_Rekey(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	testBarrier_Rekey(t, b)
}

// Verify data sent through is encrypted
func TestXChaCha20Barrier_Confidential(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)

	// Put a logical entry
	entry := &logical.StorageEntry{Key: "test", Value: []byte("test")}
	err = b.Put(context.Background(), entry)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the physical entry
	pe, err := inm.Get(context.Background(), "test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pe == nil {
		t.Fatalf("missing physical entry")
	}

	if pe.Key != "test" {
		t.Fatalf("bad: %#v", pe)
	}
	if bytes.Equal(pe.Value, entry.Value) {
		t.Fatalf("bad: %#v", pe)
	}
}

// Verify data sent through cannot be tampered with
func TestXChaCha20Barrier_Integrity(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)

	// Put a logical entry
	entry := &logical.StorageEntry{Key: "test", Value: []byte("test")}
	err = b.Put(context.Background(), entry)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Change a byte in the underlying physical entry
	pe, _ := inm.Get(context.Background(), "test")
	pe.Value[15]++
	err = inm.Put(context.Background(), pe)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Read from the barrier
	_, err = b.Get(context.Background(), "test")
	if err == nil {
		t.Fatalf("should fail!")
	}
}

func TestXChaCha20Barrier_MoveIntegrity(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b.currentXChaCha20VersionByte = XChaCha20Version1

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	err = b.Initialize(context.Background(), key, nil, rand.Reader)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = b.Unseal(context.Background(), key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Put a logical entry
	entry := &logical.StorageEntry{Key: "test", Value: []byte("test")}
	err = b.Put(context.Background(), entry)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Change the location of the underlying physical entry
	pe, _ := inm.Get(context.Background(), "test")
	pe.Key = "moved"
	err = inm.Put(context.Background(), pe)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Read from the barrier
	_, err = b.Get(context.Background(), "moved")
	if err == nil {
		t.Fatalf("should fail with version 2!")
	}
}

func TestXChaCha20Encrypt_Unique(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)

	if b.keyring == nil {
		t.Fatalf("barrier is sealed")
	}

	entry := &logical.StorageEntry{Key: "test", Value: []byte("test")}
	term := b.keyring.ActiveTerm()
	primary, _ := b.aeadForTerm(term)

	first, err := b.encrypt("test", term, primary, entry.Value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.encrypt("test", term, primary, entry.Value)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Fatalf("improper random seeding detected")
	}
}

func TestXChaCha20Initialize_KeyLength(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	long := []byte("ThisKeyDoesNotHaveTheRightLength!")
	middle := []byte("ThisIsASecretKeyAndMore")
	short := []byte("Key")

	err = b.Initialize(context.Background(), long, nil, rand.Reader)

	if err == nil {
		t.Fatalf("key length protection failed")
	}

	err = b.Initialize(context.Background(), middle, nil, rand.Reader)

	if err == nil {
		t.Fatalf("key length protection failed")
	}

	err = b.Initialize(context.Background(), short, nil, rand.Reader)

	if err == nil {
		t.Fatalf("key length protection failed")
	}
}

func TestXChaCha20Encrypt_BarrierEncryptor(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, err := b.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("err generating key: %v", err)
	}
	ctx := context.Background()
	b.Initialize(ctx, key, nil, rand.Reader)
	b.Unseal(ctx, key)

	cipher, err := b.Encrypt(ctx, "foo", []byte("quick brown fox"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	plain, err := b.Decrypt(ctx, "foo", cipher)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(plain) != "quick brown fox" {
		t.Fatalf("bad: %s", plain)
	}
}

// Ensure Decrypt returns an error (rather than panic) when given a ciphertext
// that is nil or too short
func TestXChaCha20Decrypt_InvalidCipherLength(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	key, err := b.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("err generating key: %v", err)
	}
	ctx := context.Background()
	b.Initialize(ctx, key, nil, rand.Reader)
	b.Unseal(ctx, key)

	var nilCipher []byte
	if _, err = b.Decrypt(ctx, "", nilCipher); err == nil {
		t.Fatal("expected error when given nil cipher")
	}
	emptyCipher := []byte{}
	if _, err = b.Decrypt(ctx, "", emptyCipher); err == nil {
		t.Fatal("expected error when given empty cipher")
	}

	badTermLengthCipher := make([]byte, 3, 3)
	if _, err = b.Decrypt(ctx, "", badTermLengthCipher); err == nil {
		t.Fatal("expected error when given cipher with too short term")
	}
}

func TestXChaCha20Barrier_ReloadKeyring(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Initialize and unseal
	key, _ := b.GenerateKey(rand.Reader)
	b.Initialize(context.Background(), key, nil, rand.Reader)
	b.Unseal(context.Background(), key)

	keyringRaw, err := inm.Get(context.Background(), keyringPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Encrypt something to test cache invalidation
	_, err = b.Encrypt(context.Background(), "foo", []byte("quick brown fox"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	{
		// Create a second barrier and rotate the keyring
		b2, err := NewXChaCha20Barrier(inm)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		b2.Unseal(context.Background(), key)
		_, err = b2.Rotate(context.Background(), rand.Reader)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Reload the keyring on the first
	err = b.ReloadKeyring(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if b.keyring.ActiveTerm() != 2 {
		t.Fatal("failed to reload keyring")
	}
	if len(b.cache) != 0 {
		t.Fatal("failed to clear cache")
	}

	// Encrypt something to test cache invalidation
	_, err = b.Encrypt(context.Background(), "foo", []byte("quick brown fox"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Restore old keyring to test rolling back
	err = inm.Put(context.Background(), keyringRaw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Reload the keyring on the first
	err = b.ReloadKeyring(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if b.keyring.ActiveTerm() != 1 {
		t.Fatal("failed to reload keyring")
	}
	if len(b.cache) != 0 {
		t.Fatal("failed to clear cache")
	}
}

func TestXChaCha20Barrier_LegacyRotate(t *testing.T) {
	inm, err := inmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b1, err := NewXChaCha20Barrier(inm)
	if err != nil {
		t.Fatalf("err: %v", err)
	} // Initialize the barrier
	key, _ := b1.GenerateKey(rand.Reader)
	b1.Initialize(context.Background(), key, nil, rand.Reader)
	err = b1.Unseal(context.Background(), key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	k1 := b1.keyring.TermKey(1)
	k1.Encryptions = 0
	k1.InstallTime = time.Now().Add(-24 * 366 * time.Hour)
	b1.persistKeyring(context.Background(), b1.keyring)
	b1.Seal()

	err = b1.Unseal(context.Background(), key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	reason, err := b1.CheckBarrierAutoRotate(context.Background())
	if err != nil || reason != legacyRotateReason {
		t.Fail()
	}
}

// TestXChaCha20Barrier_persistKeyring_Context checks that we get the right errors if
// the context is cancelled or times-out before the first part of persistKeyring
// is able to persist the keyring itself (i.e. we don't go on to try and persist
// the root key).
func TestXChaCha20Barrier_persistKeyring_Context(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		shouldCancel         bool
		isErrorExpected      bool
		expectedErrorMessage string
		contextTimeout       time.Duration
		testTimeout          time.Duration
	}{
		"cancelled": {
			shouldCancel:         true,
			isErrorExpected:      true,
			expectedErrorMessage: "failed to persist keyring: context canceled",
			contextTimeout:       8 * time.Second,
			testTimeout:          10 * time.Second,
		},
		"timeout-before-keyring": {
			isErrorExpected:      true,
			expectedErrorMessage: "failed to persist keyring: context deadline exceeded",
			contextTimeout:       1 * time.Nanosecond,
			testTimeout:          5 * time.Second,
		},
	}

	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Set up barrier
			backend, err := inmem.NewInmem(nil, corehelpers.NewTestLogger(t))
			require.NoError(t, err)
			barrier, err := NewXChaCha20Barrier(backend)
			require.NoError(t, err)
			key, err := barrier.GenerateKey(rand.Reader)
			require.NoError(t, err)
			err = barrier.Initialize(context.Background(), key, nil, rand.Reader)
			require.NoError(t, err)
			err = barrier.Unseal(context.Background(), key)
			require.NoError(t, err)
			k := barrier.keyring.TermKey(1)
			k.Encryptions = 0
			k.InstallTime = time.Now().Add(-24 * 366 * time.Hour)

			// Persist the keyring
			ctx, cancel := context.WithTimeout(context.Background(), tc.contextTimeout)
			persistChan := make(chan error)
			go func() {
				if tc.shouldCancel {
					cancel()
				}
				persistChan <- barrier.persistKeyring(ctx, barrier.keyring)
			}()

			select {
			case err := <-persistChan:
				switch {
				case tc.isErrorExpected:
					require.Error(t, err)
					require.EqualError(t, err, tc.expectedErrorMessage)
				default:
					require.NoError(t, err)
				}
			case <-time.After(tc.testTimeout):
				t.Fatal("timeout reached")
			}
		})
	}
}
