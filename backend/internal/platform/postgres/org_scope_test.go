package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// ADR-174: organization assignments are independent of company membership;
// the authorization adapter must still check membership for company targets.
func TestOrganizationScopeMigration(t *testing.T) {
	f := identityFixture(t)
	u, err := f.user(t, f.orgA, "orgadmin")
	require.NoError(t, err)
	other, err := f.user(t, f.orgB, "other")
	require.NoError(t, err)
	role, foreignRole := uuid.New(), uuid.New()
	for id, org := range map[uuid.UUID]uuid.UUID{role: f.orgA, foreignRole: f.orgB} {
		_, err = f.pool.Exec(f.ctx, "INSERT INTO permission_roles (id, organization_id, code) VALUES ($1,$2,'admin')", id, org)
		require.NoError(t, err)
	}
	insert := func(user, org, r uuid.UUID, company any, level string, branch any) error {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO permission_assignments
            (organization_id,user_id,role_id,company_id,scope_level,branch_id)
            VALUES ($1,$2,$3,$4,$5,$6)`, org, user, r, company, level, branch)
		return err
	}
	require.NoError(t, insert(u, f.orgA, role, nil, "organization", nil))
	assert.Error(t, insert(u, f.orgA, role, nil, "organization", nil), "duplicate")
	assert.Error(t, insert(other, f.orgA, role, nil, "organization", nil), "cross organization user")
	assert.Error(t, insert(u, f.orgA, foreignRole, nil, "organization", nil), "cross organization role")
	assert.Error(t, insert(u, f.orgA, role, f.compA1, "organization", nil), "organization must not carry company")
	assert.Error(t, insert(u, f.orgA, role, nil, "organization", uuid.New()), "organization must not carry branch")
	assert.Error(t, insert(u, f.orgA, role, nil, "company", nil), "company cannot be absent")
	assert.Error(t, insert(u, f.orgA, role, f.compA1, "company", nil), "company membership still mandatory")
	require.NoError(t, f.member(u, f.orgA, f.compA1))
	require.NoError(t, insert(u, f.orgA, role, f.compA1, "company", nil))
	assert.Error(t, postgres.RollbackTo(f.ctx, f.pool, 7), "rollback must not discard organization assignments")
	var n int
	require.NoError(t, f.pool.QueryRow(f.ctx, "SELECT count(*) FROM permission_assignments").Scan(&n))
	assert.Equal(t, 2, n)
	_, err = f.pool.Exec(f.ctx, "DELETE FROM permission_assignments WHERE scope_level='organization'")
	require.NoError(t, err)
	require.NoError(t, postgres.RollbackTo(f.ctx, f.pool, 7))
	require.NoError(t, postgres.Migrate(f.ctx, f.pool))
	require.NoError(t, f.pool.QueryRow(f.ctx, "SELECT count(*) FROM permission_assignments").Scan(&n))
	assert.Equal(t, 1, n, "existing company assignment survives upgrade and rollback")
	require.NoError(t, insert(u, f.orgA, role, nil, "organization", nil))
}
