package vault

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/openbao/openbao/sdk/v2/physical"
)

func (c *Core) GetIndex(ctx context.Context) string {
	if !c.HAEnabled() {
		c.logger.Info("getIndex failed: not HA")
		return ""
	}

	backend, ok := c.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		c.logger.Info("getIndex failed: not a replicated backend")
		return ""
	}

	index, err := backend.AppliedReplicationIndex(ctx)
	if err != nil {
		c.logger.Error("getIndex failed", "error", err)
		return ""
	}

	return index
}

// Whether the request should be forwarded instead.
func (c *Core) AwaitIndexAndMaybeForward(ctx context.Context, maxWait time.Duration, target string) bool {
	if !c.HAEnabled() {
		c.logger.Error("await index skipped: not ha enabled")
		// Never forward.
		return false
	}

	if !c.StandbyReadsEnabled() {
		c.logger.Error("await index skipped: reads not enabled")
		// Never forward; we'll be forwarded anyways. This may occasionally
		// return false on the active node, but that's fine as we won't be
		// forwarded later.
		return false
	}

	if !c.Standby() {
		c.logger.Error("await index skipped: active node")
		return false
	}

	backend, ok := c.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		c.logger.Error("await index skipped: not indexed")
		return false
	}

	minValue := maxWait / 100
	if minValue < 15*time.Millisecond {
		minValue = 15 * time.Millisecond
		if minValue > maxWait {
			minValue = maxWait / 10
		}
	}
	maxIncrement := 1 * time.Second

	var b backoff.BackOff = backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(minValue),
		backoff.WithMaxInterval(maxIncrement),
		backoff.WithMaxElapsedTime(maxWait),
	)
	b.Reset()

	if err := backoff.Retry(func() error {
		applied, err := backend.AppliedReplicationIndex(ctx)
		if err != nil {
			return err
		}

		ok, err := backend.GreaterEqualReplicationIndex(ctx, applied, target)
		if err != nil {
			return err
		}

		if !ok {
			return errors.New("not yet reached")
		}

		return nil
	}, b); err != nil {
		c.logger.Trace("not yet at replication index", "error", err)
		return true
	}

	return false
}

// Whether the request should be retried instead.
func (c *Core) CheckIndexAndMaybeRetry(ctx context.Context, target string) bool {
	if !c.HAEnabled() {
		c.logger.Error("check index skipped: not ha enabled")
		// Never forward.
		return false
	}

	if !c.StandbyReadsEnabled() {
		c.logger.Error("check index skipped: reads not enabled")
		// Never forward; we'll be forwarded anyways. This may occasionally
		// return false on the active node, but that's fine as we won't be
		// forwarded later.
		return false
	}

	if !c.Standby() {
		c.logger.Error("check index skipped: active node")
		return false
	}

	backend, ok := c.underlyingPhysical.(physical.ReplicationIndexBackend)
	if !ok {
		c.logger.Error("check index skipped: not indexed")
		return false
	}

	applied, err := backend.AppliedReplicationIndex(ctx)
	if err != nil {
		return false
	}

	ok, err := backend.GreaterEqualReplicationIndex(ctx, applied, target)
	if err != nil {
		return false
	}

	return !ok
}
