// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package postgresql

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/openbao/openbao/sdk/v2/physical"
)

var _ physical.ReplicationIndexBackend = &PostgreSQLBackend{}

const indexRefreshWindow = 25 * time.Millisecond

func (p *PostgreSQLBackend) AppliedReplicationIndex(ctx context.Context) (string, error) {
	now := time.Now()
	last := p.lastIndexTime.Load()
	if last == nil || last.Before(now.Add(-1*indexRefreshWindow)) {
		if err := func() error {
			p.lastIndexLock.Lock()
			defer p.lastIndexLock.Unlock()

			last := p.lastIndexTime.Load()
			if last != nil && last.After(now.Add(-1*indexRefreshWindow)) {
				// Already updated
				return nil
			}

			now = time.Now()
			index, err := p.queryLSN(ctx)
			if err != nil {
				return err
			}

			p.lastIndex.Store(index)
			p.lastIndexTime.Store(&now)

			return nil
		}(); err != nil {
			return "", err
		}
	}

	return strconv.FormatUint(p.lastIndex.Load(), 10), nil
}

func (p *PostgreSQLBackend) queryLSN(ctx context.Context) (uint64, error) {
	var index int64

	indexQuery := "SELECT pg_current_wal_lsn()::bigint;"
	if err := p.client.QueryRowContext(ctx, indexQuery).Scan(&index); err != nil {
		return 0, fmt.Errorf("pg_current_wal_lsn() failed: %w", err)
	}

	return uint64(index), nil
}

func (p *PostgreSQLBackend) GreaterEqualReplicationIndex(ctx context.Context, left string, right string) (bool, error) {
	leftInt, err := strconv.ParseUint(left, 10, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing left: %w", err)
	}

	rightInt, err := strconv.ParseUint(right, 10, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing right: %w", err)
	}

	return leftInt >= rightInt, nil
}
