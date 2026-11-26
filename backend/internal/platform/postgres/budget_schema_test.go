package postgres_test

import (
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBudgetSchemaIsolation(t *testing.T) {
	f := identityFixture(t)
	key := make([]byte, 32)
	key[0] = 42
	put := func(org uuid.UUID, k []byte, kind string, max int) error {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO identity_attempt_budgets(organization_id,budget_key,kind,max_attempts,window_us,cooldown_us)
   VALUES($1,$2,$3,$4,60000000,120000000)`, org, k, kind, max)
		return err
	}
	require.NoError(t, put(f.orgA, key, "password", 3))
	require.NoError(t, put(f.orgB, key, "password", 3))
	require.NoError(t, put(f.orgA, key, "pin", 3))
	assert.Error(t, put(f.orgA, key, "password", 3))
	assert.Error(t, put(uuid.New(), key, "password", 3))
	assert.Error(t, put(f.orgA, []byte{1}, "password", 3))
	assert.Error(t, put(f.orgA, key, "unknown", 3))
	key[0] = 43
	assert.Error(t, put(f.orgA, key, "password", 101))
	_, err := f.pool.Exec(f.ctx, "UPDATE identity_attempt_budgets SET used=1 WHERE organization_id=$1", f.orgA)
	assert.Error(t, err)
	assert.Error(t, postgres.RollbackTo(f.ctx, f.pool, 9))
	var n int
	require.NoError(t, f.pool.QueryRow(f.ctx, "SELECT count(*) FROM identity_attempt_budgets").Scan(&n))
	assert.Equal(t, 3, n)
}
func TestBudgetSchemaEmptyRollback(t *testing.T) {
	f := identityFixture(t)
	require.NoError(t, postgres.RollbackTo(f.ctx, f.pool, 9))
	require.NoError(t, postgres.Migrate(f.ctx, f.pool))
	var n int
	require.NoError(t, f.pool.QueryRow(f.ctx, "SELECT count(*) FROM identity_attempt_budgets").Scan(&n))
	assert.Zero(t, n)
}
