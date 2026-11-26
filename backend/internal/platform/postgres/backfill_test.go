package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// Second independent review: migrations must work on databases that
// already hold data written by earlier versions.
func TestMigrationsUpgradeExistingData(t *testing.T) {
	pool := pgtest.New(t)
	ctx := context.Background()
	require.NoError(t, postgres.RollbackTo(ctx, pool, 2)) // schema as of the first PR commit

	org, co, br, wh := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	skuUnit, skuKg := uuid.New(), uuid.New()
	active, released := uuid.New(), uuid.New()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organizations (id, name) VALUES ($1, 'o')`, []any{org}},
		{`INSERT INTO companies (id, organization_id, name) VALUES ($1, $2, 'c')`, []any{co, org}},
		{`INSERT INTO branches (id, company_id, name) VALUES ($1, $2, 'b')`, []any{br, co}},
		{`INSERT INTO warehouses (id, company_id, branch_id, name) VALUES ($1, $2, $3, 'w')`, []any{wh, co, br}},
		// v2 accepted free-text, mixed-case units.
		{`INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision) VALUES ($1, $2, 'U', 'u', 'unit', ' Und ', 0)`, []any{skuUnit, co}},
		{`INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision) VALUES ($1, $2, 'K', 'k', 'weight', 'KG', 3)`, []any{skuKg, co}},
		{`INSERT INTO inventory_balances (company_id, warehouse_id, sku_id, physical, reserved) VALUES ($1, $2, $3, 10, 4)`, []any{co, wh, skuUnit}},
		{`INSERT INTO inventory_reservations (id, company_id, warehouse_id, sku_id, quantity, purpose, status, expires_at) VALUES ($1, $2, $3, $4, 4, 'p', 'active', $5)`, []any{active, co, wh, skuUnit, time.Now().Add(time.Hour)}},
		{`INSERT INTO inventory_reservations (id, company_id, warehouse_id, sku_id, quantity, purpose, status, expires_at, closed_at) VALUES ($1, $2, $3, $4, 2, 'p', 'released', $5, now())`, []any{released, co, wh, skuUnit, time.Now()}},
	} {
		_, err := pool.Exec(ctx, q.sql, q.args...)
		require.NoError(t, err, q.sql)
	}

	require.NoError(t, postgres.Migrate(ctx, pool), "upgrade must succeed with legacy rows")

	var u, k string
	require.NoError(t, pool.QueryRow(ctx, `SELECT base_unit FROM inventory_skus WHERE id = $1`, skuUnit).Scan(&u))
	require.NoError(t, pool.QueryRow(ctx, `SELECT base_unit FROM inventory_skus WHERE id = $1`, skuKg).Scan(&k))
	assert.Equal(t, "und", u)
	assert.Equal(t, "kg", k)

	var created, terminal int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE event_type = 'created'), count(*) FILTER (WHERE event_type <> 'created')
		FROM inventory_reservation_events`).Scan(&created, &terminal))
	assert.Equal(t, 2, created, "every legacy reservation gets a created event")
	assert.Equal(t, 1, terminal, "closed legacy reservations get their terminal event")
	var sum string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN event_type = 'created' THEN quantity ELSE -quantity END), 0)::text
		FROM inventory_reservation_events`).Scan(&sum))
	assert.Equal(t, "4.000000", sum, "events add up to the reserved balance")
}

// Legacy rows that cannot be mapped do not block the upgrade; the
// constraint still applies to new rows.
func TestUnitConstraintToleratesUnmappableLegacyRows(t *testing.T) {
	pool := pgtest.New(t)
	ctx := context.Background()
	require.NoError(t, postgres.RollbackTo(ctx, pool, 2))
	org, co := uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'o')`, org)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO companies (id, organization_id, name) VALUES ($1, $2, 'c')`, co, org)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision) VALUES ($1, $2, 'C', 'c', 'unit', 'caja', 0)`, uuid.New(), co)
	require.NoError(t, err)

	require.NoError(t, postgres.Migrate(ctx, pool))
	_, err = pool.Exec(ctx, `INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision) VALUES ($1, $2, 'N', 'n', 'unit', 'caja', 0)`, uuid.New(), co)
	assert.Error(t, err, "new rows must respect the dimension constraint")
}
