package kv

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/consts"
	"github.com/openbao/openbao/sdk/v2/helper/locksutil"
	"github.com/openbao/openbao/sdk/v2/helper/pluginutil"
	"github.com/openbao/openbao/sdk/v2/logical"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (b *versionedKVBackend) perfSecondaryCheck() bool {
	replState := b.System().ReplicationState()
	if (!b.System().LocalMount() && replState.HasState(consts.ReplicationPerformanceSecondary)) ||
		replState.HasState(consts.ReplicationPerformanceStandby) {
		return true
	}
	return false
}

func (b *versionedKVBackend) upgradeCheck(next framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		if atomic.LoadUint32(b.upgrading) == 1 {
			// Sleep for a very short time before returning. This helps clients
			// that are trying to access a mount immediately upon enabling be
			// more likely to behave correctly since the operation should take
			// almost no time.
			time.Sleep(15 * time.Millisecond)

			if atomic.LoadUint32(b.upgrading) == 1 {
				if b.perfSecondaryCheck() {
					return logical.ErrorResponse("Waiting for the primary to upgrade from non-versioned to versioned data. This backend will be unavailable for a brief period and will resume service when the primary is finished."), logical.ErrInvalidRequest
				} else {
					return logical.ErrorResponse("Upgrading from non-versioned to versioned data. This backend will be unavailable for a brief period and will resume service shortly."), logical.ErrInvalidRequest
				}
			}
		}

		return next(ctx, req, data)
	}
}

func (b *versionedKVBackend) upgradeDone(ctx context.Context, s logical.Storage) (bool, error) {
	upgradeEntry, err := s.Get(ctx, path.Join(b.storagePrefix, "upgrading"))
	if err != nil {
		return false, err
	}

	var upgradeInfo UpgradeInfo
	if upgradeEntry != nil {
		err := proto.Unmarshal(upgradeEntry.Value, &upgradeInfo)
		if err != nil {
			return false, err
		}
	}

	return upgradeInfo.Done, nil
}

func (b *versionedKVBackend) Upgrade(ctx context.Context, s logical.Storage) error {
	replState := b.System().ReplicationState()

	// Don't run if the plugin is in metadata mode.
	if pluginutil.InMetadataMode() {
		b.Logger().Info("upgrade not running while plugin is in metadata mode")
		return nil
	}

	// Don't run while on a DR secondary.
	if replState.HasState(consts.ReplicationDRSecondary) {
		b.Logger().Info("upgrade not running on disaster recovery replication secondary")
		return nil
	}

	if !atomic.CompareAndSwapUint32(b.upgrading, 0, 1) {
		return errors.New("upgrade already in process")
	}

	// If we are a replication secondary or performance standby, wait until the primary has finished
	// upgrading.
	if b.perfSecondaryCheck() {
		b.Logger().Info("upgrade not running on performance replication secondary or performance standby")

		go func() {
			for {
				time.Sleep(time.Second)

				// If we failed because the context is closed we are
				// shutting down. Close this go routine and set the upgrade
				// flag back to 0 for good measure.
				if ctx.Err() != nil {
					atomic.StoreUint32(b.upgrading, 0)
					return
				}

				done, err := b.upgradeDone(ctx, s)
				if err != nil {
					b.Logger().Error("upgrading resulted in error", "error", err)
				}

				if done {
					break
				}
			}

			atomic.StoreUint32(b.upgrading, 0)
		}()

		return nil
	}

	upgradeInfo := &UpgradeInfo{
		StartedTime: timestamppb.Now(),
	}

	// Encode the canary
	info, err := proto.Marshal(upgradeInfo)
	if err != nil {
		return err
	}

	// Because this is a long running process we need a new context.
	ctx = context.Background()

	upgradeKey := func(key string) error {
		if strings.HasPrefix(key, b.storagePrefix) {
			return nil
		}

		// Create a transaction if we can.
		originalStorage := s
		storage := s
		if txnStorage, ok := storage.(logical.TransactionalStorage); ok {
			txn, err := txnStorage.BeginTx(ctx)
			if err != nil {
				return err
			}

			defer txn.Rollback(ctx)
			storage = txn
		}

		// Read the old data
		data, err := storage.Get(ctx, key)
		if err != nil {
			return err
		}

		locksutil.LockForKey(b.locks, key).Lock()
		defer locksutil.LockForKey(b.locks, key).Unlock()

		meta := &KeyMetadata{
			Key:      key,
			Versions: map[uint64]*VersionMetadata{},
		}

		versionKey, err := b.getVersionKey(ctx, key, 1, s)
		if err != nil {
			return err
		}

		version := &Version{
			Data:        data.Value,
			CreatedTime: timestamppb.Now(),
		}

		buf, err := proto.Marshal(version)
		if err != nil {
			return err
		}

		// Store the version data
		if err := storage.Put(ctx, &logical.StorageEntry{
			Key:   versionKey,
			Value: buf,
		}); err != nil {
			return err
		}

		// Store the metadata
		meta.AddVersion(version.CreatedTime, nil, 1)
		err = b.writeKeyMetadata(ctx, s, meta)
		if err != nil {
			return err
		}

		// delete the old key
		err = storage.Delete(ctx, key)
		if err != nil {
			return err
		}

		// Commit our transaction if we created one! We're done making
		// modifications to storage.
		if txn, ok := storage.(logical.Transaction); ok && storage != originalStorage {
			if err := txn.Commit(ctx); err != nil {
				return err
			}
			storage = originalStorage
		}

		return nil
	}

	// Run the actual upgrade in a go routine so we don't block the client on a
	// potentially long process.
	go func() {
		// Write the canary value and if we are read only wait until the setup
		// process has finished.
	READONLY_LOOP:
		for {
			err := s.Put(ctx, &logical.StorageEntry{
				Key:   path.Join(b.storagePrefix, "upgrading"),
				Value: info,
			})
			switch {
			case err == nil:
				break READONLY_LOOP
			case err.Error() == logical.ErrSetupReadOnly.Error():
				time.Sleep(10 * time.Millisecond)
			default:
				b.Logger().Error("writing upgrade info resulted in an error", "error", err)
				return
			}
		}

		b.Logger().Info("collecting keys to upgrade")
		keys, err := logical.CollectKeys(ctx, s)
		if err != nil {
			b.Logger().Error("upgrading resulted in error", "error", err)
			return
		}

		b.Logger().Info("done collecting keys", "num_keys", len(keys))
		for i, key := range keys {
			if b.Logger().IsDebug() && i%500 == 0 {
				b.Logger().Debug("upgrading keys", "progress", fmt.Sprintf("%d/%d", i, len(keys)))
			}
			err := upgradeKey(key)
			if err != nil {
				b.Logger().Error("upgrading resulted in error", "error", err, "progress", fmt.Sprintf("%d/%d", i+1, len(keys)))
				return
			}
		}

		b.Logger().Info("upgrading keys finished")

		// We do this now so that we ensure it's written by the primary before
		// secondaries unblock
		b.l.Lock()
		if _, err = b.policy(ctx, s); err != nil {
			b.Logger().Error("error checking/creating policy after upgrade", "error", err)
		}
		b.l.Unlock()

		// Write upgrade done value
		upgradeInfo.Done = true
		info, err := proto.Marshal(upgradeInfo)
		if err != nil {
			b.Logger().Error("encoding upgrade info resulted in an error", "error", err)
		}

		err = s.Put(ctx, &logical.StorageEntry{
			Key:   path.Join(b.storagePrefix, "upgrading"),
			Value: info,
		})
		if err != nil {
			b.Logger().Error("writing upgrade done resulted in an error", "error", err)
		}

		atomic.StoreUint32(b.upgrading, 0)
	}()

	return nil
}
