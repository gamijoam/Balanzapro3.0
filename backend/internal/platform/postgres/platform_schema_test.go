package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
)

// BP-009 subtask 1 (#82): schema for module activation (ADR-023), versioned
// configuration (ADR-027) and audit events. Synthetic data only.

type platFixture struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	orgA, orgB   uuid.UUID
	compA, compB uuid.UUID
}

func platformFixture(t *testing.T) platFixture {
	t.Helper()
	f := platFixture{ctx: context.Background(), pool: pgtest.New(t),
		orgA: uuid.New(), orgB: uuid.New(), compA: uuid.New(), compB: uuid.New()}
	for _, o := range []uuid.UUID{f.orgA, f.orgB} {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'o')`, o)
		require.NoError(t, err)
	}
	for c, o := range map[uuid.UUID]uuid.UUID{f.compA: f.orgA, f.compB: f.orgB} {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO companies (id, organization_id, name) VALUES ($1, $2, 'c')`, c, o)
		require.NoError(t, err)
	}
	return f
}

func (f platFixture) exec(sql string, args ...any) error {
	_, err := f.pool.Exec(f.ctx, sql, args...)
	return err
}

func TestCompanyModulesStateIsConsistent(t *testing.T) {
	f := platformFixture(t)
	ins := `INSERT INTO company_modules (company_id, module_id, status) VALUES ($1, $2, $3)`
	require.NoError(t, f.exec(ins, f.compA, "inventory", "active"))
	assert.Error(t, f.exec(ins, f.compA, "inventory", "active"), "one row per company and module")
	require.NoError(t, f.exec(ins, f.compB, "inventory", "active"), "activation is per company")
	for _, bad := range []string{"", "Inventory", "inv entory", "1inv", "inventory!"} {
		assert.Error(t, f.exec(ins, f.compA, bad, "active"), "module id %q", bad)
	}
	assert.Error(t, f.exec(ins, f.compA, "sales", "deleted"), "unknown status")
	assert.Error(t, f.exec(ins, uuid.New(), "sales", "active"), "unknown company")
	assert.Error(t, f.exec(`DELETE FROM company_modules WHERE company_id = $1`, f.compA),
		"deactivation never deletes: rows are kept")
	require.NoError(t, f.exec(`UPDATE company_modules SET status = 'inactive', version = version + 1 WHERE company_id = $1`, f.compA))
}

func TestConfigEntriesAreVersionedAndAppendOnly(t *testing.T) {
	f := platformFixture(t)
	ins := `INSERT INTO config_entries (key, scope_level, organization_id, scope_id, version, value, reason)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, 'prueba sintética')`
	require.NoError(t, f.exec(ins, "pos.receipt_footer", "global", nil, nil, 1, `"Gracias"`))
	require.NoError(t, f.exec(ins, "pos.receipt_footer", "company", f.orgA, f.compA, 1, `"Hola A"`))
	require.NoError(t, f.exec(ins, "pos.receipt_footer", "company", f.orgA, f.compA, 2, `"Hola A2"`))
	assert.Error(t, f.exec(ins, "pos.receipt_footer", "company", f.orgA, f.compA, 2, `"dup"`), "versions are unique")
	assert.Error(t, f.exec(ins, "pos.receipt_footer", "company", f.orgA, f.compA, 0, `"v0"`), "versions start at 1")

	assert.Error(t, f.exec(ins, "x.y", "global", f.orgA, nil, 1, `1`), "global has no organization")
	assert.Error(t, f.exec(ins, "x.y", "company", nil, f.compA, 1, `1`), "non-global needs organization")
	assert.Error(t, f.exec(ins, "x.y", "company", f.orgA, nil, 1, `1`), "non-global needs a scope id")
	assert.Error(t, f.exec(ins, "x.y", "planet", f.orgA, f.compA, 1, `1`), "unknown level")
	assert.Error(t, f.exec(ins, "Bad Key", "global", nil, nil, 1, `1`), "key format")
	assert.Error(t, f.exec(ins, "x.y", "organization", uuid.New(), uuid.New(), 1, `1`), "unknown organization")
	assert.Error(t, f.exec(`INSERT INTO config_entries (key, scope_level, version, value, reason)
		VALUES ('x.y', 'global', 1, '1'::jsonb, '  ')`), "a reason is required")

	assert.Error(t, f.exec(`UPDATE config_entries SET value = '"edit"'::jsonb`), "history is never edited")
	assert.Error(t, f.exec(`DELETE FROM config_entries`))
}

func TestAuditEventsAreAppendOnlyAndScoped(t *testing.T) {
	f := platformFixture(t)
	ins := `INSERT INTO audit_events (organization_id, company_id, actor_user_id, action, entity_type, entity_id,
		before, after, reason, correlation_id) VALUES ($1, $2, $3, $4, 'module', 'inventory', NULL, '{"status":"active"}'::jsonb, 'alta', $5)`
	require.NoError(t, f.exec(ins, f.orgA, f.compA, nil, "module.activate", uuid.New()))
	assert.Error(t, f.exec(ins, f.orgA, f.compB, nil, "module.activate", uuid.New()),
		"a company of another organization cannot be recorded under this one (SEC-01)")
	assert.Error(t, f.exec(ins, f.orgA, f.compA, nil, "", uuid.New()), "action is required")
	assert.Error(t, f.exec(ins, f.orgA, f.compA, nil, "module.activate", nil), "correlation id is required")
	assert.Error(t, f.exec(`UPDATE audit_events SET reason = 'x'`))
	assert.Error(t, f.exec(`DELETE FROM audit_events`))
	assert.Error(t, f.exec(`TRUNCATE audit_events`))
}
