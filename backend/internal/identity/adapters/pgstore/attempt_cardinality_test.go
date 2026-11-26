package pgstore_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptCardinality(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := pgtest.New(t)
	orgs := organization.NewStore(pool)
	org, err := orgs.CreateOrganization(ctx, "A")
	require.NoError(t, err)
	other, err := orgs.CreateOrganization(ctx, "B")
	require.NoError(t, err)
	p := domain.AttemptPolicy{MaxAttempts: 3, Window: time.Minute, Cooldown: time.Minute}
	key := func(i byte) application.AttemptKey {
		return application.AttemptKey{OrganizationID: org, Kind: "password", Digest: [32]byte{i}}
	}
	store, err := pgstore.NewAttemptStore(pool, 4)
	require.NoError(t, err)
	_, err = pgstore.NewAttemptStore(pool, 0)
	require.Error(t, err)
	ok, err := pgstore.New(pool).ReserveAttempt(ctx, key(1), p)
	require.Error(t, err)
	assert.False(t, ok)
	require.NoError(t, errOrGrant(store.ReserveAttempt(ctx, key(1), p)))
	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, 20)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Go(func() {
			replica, e := pgstore.NewAttemptStore(pool, 4)
			if e != nil {
				results <- result{false, e}
				return
			}
			ok, e := replica.ReserveAttempt(ctx, key(byte(i+2)), p)
			results <- result{ok, e}
		})
	}
	wg.Wait()
	close(results)
	grants, rejections := 0, 0
	for r := range results {
		if r.ok {
			require.NoError(t, r.err)
			grants++
		} else {
			require.ErrorIs(t, r.err, application.ErrAttemptCapacity)
			rejections++
		}
	}
	assert.Equal(t, 3, grants)
	assert.Equal(t, 17, rejections)
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM identity_attempt_budgets WHERE organization_id=$1", org).Scan(&n))
	assert.Equal(t, 4, n)
	require.NoError(t, errOrGrant(store.ReserveAttempt(ctx, key(1), p)))
	otherKey := key(1)
	otherKey.OrganizationID = other
	require.NoError(t, errOrGrant(store.ReserveAttempt(ctx, otherKey, p)))
	fresh := key(100)
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{{Key: key(1), Policy: p}, {Key: fresh, Policy: p}})
	assert.ErrorIs(t, err, application.ErrAttemptCapacity)
	assert.False(t, ok)
	first := key(1)
	require.NoError(t, pool.QueryRow(ctx, "SELECT used FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2", org, first.Digest[:]).Scan(&n))
	assert.Equal(t, 2, n)
}

func errOrGrant(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("expected grant")
	}
	return nil
}
