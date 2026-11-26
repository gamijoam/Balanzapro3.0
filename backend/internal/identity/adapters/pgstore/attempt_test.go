package pgstore_test

import (
	"context"
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

func TestAttemptReservationsAreDurable(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	orgs := organization.NewStore(pool)
	org, err := orgs.CreateOrganization(ctx, "A")
	require.NoError(t, err)
	other, err := orgs.CreateOrganization(ctx, "B")
	require.NoError(t, err)
	key := application.AttemptKey{OrganizationID: org, Kind: "password", Digest: [32]byte{7}}
	policy := domain.AttemptPolicy{MaxAttempts: 3, Window: time.Minute, Cooldown: 2 * time.Minute}
	type result struct {
		allowed bool
		err     error
	}
	results := make(chan result, 24)
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Go(func() {
			replica, err := pgstore.NewAttemptStore(pool, 100)
			if err != nil {
				results <- result{false, err}
				return
			}
			ok, err := replica.ReserveAttempt(ctx, key, policy)
			results <- result{ok, err}
		})
	}
	wg.Wait()
	close(results)
	granted := 0
	for r := range results {
		require.NoError(t, r.err)
		if r.allowed {
			granted++
		}
	}
	assert.Equal(t, 3, granted)
	var used int
	var blocked time.Time
	require.NoError(t, pool.QueryRow(ctx, "SELECT used,blocked_until FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2 AND kind=$3", org, key.Digest[:], key.Kind).Scan(&used, &blocked))
	assert.Equal(t, 3, used)
	store, err := pgstore.NewAttemptStore(pool, 100)
	require.NoError(t, err)
	ok, err := store.ReserveAttempt(ctx, key, policy)
	require.NoError(t, err)
	assert.False(t, ok)
	var after time.Time
	require.NoError(t, pool.QueryRow(ctx, "SELECT blocked_until FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2 AND kind=$3", org, key.Digest[:], key.Kind).Scan(&after))
	assert.True(t, after.Equal(blocked))
	changed := policy
	changed.MaxAttempts++
	ok, err = store.ReserveAttempt(ctx, key, changed)
	assert.ErrorIs(t, err, application.ErrAttemptPolicyChanged)
	assert.False(t, ok)
	foreign := key
	foreign.OrganizationID = other
	ok, err = store.ReserveAttempt(ctx, foreign, policy)
	require.NoError(t, err)
	assert.True(t, ok)
	alternate := key
	alternate.Kind = "pin"
	ok, err = store.ReserveAttempt(ctx, alternate, policy)
	require.NoError(t, err)
	assert.True(t, ok)
	// Move only the synthetic fixture's timestamps to a prior window.
	_, err = pool.Exec(ctx, "UPDATE identity_attempt_budgets SET window_start=window_start-interval '10 minutes',last_reserved=last_reserved-interval '10 minutes',blocked_until=blocked_until-interval '10 minutes' WHERE organization_id=$1 AND budget_key=$2 AND kind=$3", org, key.Digest[:], key.Kind)
	require.NoError(t, err)
	ok, err = store.ReserveAttempt(ctx, key, policy)
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, pool.QueryRow(ctx, "SELECT used FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2 AND kind=$3", org, key.Digest[:], key.Kind).Scan(&used))
	assert.Equal(t, 1, used)
}
