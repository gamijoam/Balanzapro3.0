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

func TestAttemptBatchAtomicity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := pgtest.New(t)
	org, err := organization.NewStore(pool).CreateOrganization(ctx, "A")
	require.NoError(t, err)
	store, err := pgstore.NewAttemptStore(pool, 100)
	require.NoError(t, err)
	policy := domain.AttemptPolicy{MaxAttempts: 3, Window: time.Minute, Cooldown: time.Minute}
	common := application.AttemptBudget{Key: application.AttemptKey{OrganizationID: org, Kind: "password", Digest: [32]byte{1}}, Policy: policy}
	budgets := func(i byte) []application.AttemptBudget {
		return []application.AttemptBudget{common, {Key: application.AttemptKey{OrganizationID: org, Kind: "password", Digest: [32]byte{i}}, Policy: policy}}
	}
	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, 20)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Go(func() {
			pair := budgets(byte(i + 2))
			if i%2 == 0 {
				pair[0], pair[1] = pair[1], pair[0]
			}
			ok, err := store.ReserveAttempts(ctx, pair)
			results <- result{ok, err}
		})
	}
	wg.Wait()
	close(results)
	n := 0
	for r := range results {
		require.NoError(t, r.err)
		if r.ok {
			n++
		}
	}
	assert.Equal(t, 3, n)
	var rows, total int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*),sum(used) FROM identity_attempt_budgets WHERE organization_id=$1", org).Scan(&rows, &total))
	assert.Equal(t, 4, rows)
	assert.Equal(t, 6, total)
	ok, err := store.ReserveAttempts(ctx, budgets(100))
	require.NoError(t, err)
	assert.False(t, ok)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM identity_attempt_budgets WHERE organization_id=$1", org).Scan(&rows))
	assert.Equal(t, 4, rows)
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{common, common})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	assert.False(t, ok)
	mismatched := budgets(100)
	mismatched[1].Key.Kind = "pin"
	ok, err = store.ReserveAttempts(ctx, mismatched)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	assert.False(t, ok)

	// Force the already exhausted budget to sort after a fresh one.
	later := common
	later.Key.Digest = [32]byte{250}
	later.Policy.MaxAttempts = 1
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{later})
	require.NoError(t, err)
	require.True(t, ok)
	earlier := common
	earlier.Key.Digest = [32]byte{200}
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{later, earlier})
	require.NoError(t, err)
	assert.False(t, ok)
	var found int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2", org, earlier.Key.Digest[:]).Scan(&found))
	assert.Zero(t, found, "earlier insert must roll back")
	// Existing budget changes must also roll back.
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{earlier})
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = store.ReserveAttempts(ctx, []application.AttemptBudget{earlier, later})
	require.NoError(t, err)
	assert.False(t, ok)
	require.NoError(t, pool.QueryRow(ctx, "SELECT used FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2", org, earlier.Key.Digest[:]).Scan(&found))
	assert.Equal(t, 1, found)
}
