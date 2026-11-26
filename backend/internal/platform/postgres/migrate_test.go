package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// Rollback documented in every migration (-- +goose Down) must work: down to
// zero leaves no table, and migrating up again succeeds.
func TestMigrationsRoundTrip(t *testing.T) {
	pool := pgtest.New(t) // already migrated up
	ctx := context.Background()
	count := func() int {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name <> 'goose_db_version'`).Scan(&n))
		return n
	}
	require.Positive(t, count())
	require.NoError(t, postgres.RollbackTo(ctx, pool, 0))
	assert.Zero(t, count(), "every Down must remove what its Up created")
	require.NoError(t, postgres.Migrate(ctx, pool))
	assert.Positive(t, count())
}
