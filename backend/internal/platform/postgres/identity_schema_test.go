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

// BP-008 subtask 1 (#78): identity schema. Users belong to an organization
// and act in companies only through memberships; the database itself keeps
// a user from being attached to another organization's company (SEC-01).

type idFixture struct {
	ctx            context.Context
	pool           *pgxpool.Pool
	orgA, orgB     uuid.UUID
	compA1, compA2 uuid.UUID
	compB1         uuid.UUID
}

func identityFixture(t *testing.T) idFixture {
	t.Helper()
	f := idFixture{ctx: context.Background(), pool: pgtest.New(t)}
	f.orgA, f.orgB = uuid.New(), uuid.New()
	f.compA1, f.compA2, f.compB1 = uuid.New(), uuid.New(), uuid.New()
	for _, o := range []uuid.UUID{f.orgA, f.orgB} {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'org')`, o)
		require.NoError(t, err)
	}
	for c, o := range map[uuid.UUID]uuid.UUID{f.compA1: f.orgA, f.compA2: f.orgA, f.compB1: f.orgB} {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO companies (id, organization_id, name) VALUES ($1, $2, 'co')`, c, o)
		require.NoError(t, err)
	}
	return f
}

func (f idFixture) user(t *testing.T, org uuid.UUID, login string) (uuid.UUID, error) {
	t.Helper()
	id := uuid.New()
	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO identity_users (id, organization_id, login, display_name) VALUES ($1, $2, $3, 'Usuario sintético')`,
		id, org, login)
	return id, err
}

func (f idFixture) member(user, org, company uuid.UUID) error {
	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO identity_memberships (organization_id, user_id, company_id) VALUES ($1, $2, $3)`,
		org, user, company)
	return err
}

func TestIdentityMembershipStaysInsideTheOrganization(t *testing.T) {
	f := identityFixture(t)
	u, err := f.user(t, f.orgA, "ana")
	require.NoError(t, err)
	require.NoError(t, f.member(u, f.orgA, f.compA1))
	require.NoError(t, f.member(u, f.orgA, f.compA2), "a user may belong to several companies of its organization")

	assert.Error(t, f.member(u, f.orgA, f.compB1), "company of another organization (SEC-01)")
	assert.Error(t, f.member(u, f.orgB, f.compB1), "the user does not belong to that organization")
	assert.Error(t, f.member(u, f.orgA, f.compA1), "duplicate membership")
}

func TestIdentityLoginIsNormalizedAndUniquePerOrganization(t *testing.T) {
	f := identityFixture(t)
	_, err := f.user(t, f.orgA, "ana")
	require.NoError(t, err)
	_, err = f.user(t, f.orgA, "ana")
	assert.Error(t, err, "same login twice in one organization")
	_, err = f.user(t, f.orgB, "ana")
	assert.NoError(t, err, "logins are scoped to the organization")
	for _, bad := range []string{"", "  ", "Ana", " ana2", "ana 3"} {
		_, err := f.user(t, f.orgA, bad)
		assert.Error(t, err, "%q must be stored lowercase, trimmed and without spaces", bad)
	}
}

func TestIdentityUserStatusIsConsistent(t *testing.T) {
	f := identityFixture(t)
	u, err := f.user(t, f.orgA, "beto")
	require.NoError(t, err)
	var status string
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT status FROM identity_users WHERE id = $1`, u).Scan(&status))
	assert.Equal(t, "active", status)

	_, err = f.pool.Exec(f.ctx, `UPDATE identity_users SET status = 'disabled' WHERE id = $1`, u)
	assert.Error(t, err, "disabling requires disabled_at")
	_, err = f.pool.Exec(f.ctx, `UPDATE identity_users SET status = 'disabled', disabled_at = now() WHERE id = $1`, u)
	assert.NoError(t, err)
	_, err = f.pool.Exec(f.ctx, `UPDATE identity_users SET status = 'deleted' WHERE id = $1`, u)
	assert.Error(t, err, "unknown status")
}

func TestIdentityRowsCannotBeRemovedUnderneathMemberships(t *testing.T) {
	f := identityFixture(t)
	u, err := f.user(t, f.orgA, "carla")
	require.NoError(t, err)
	require.NoError(t, f.member(u, f.orgA, f.compA1))
	_, err = f.pool.Exec(f.ctx, `DELETE FROM identity_users WHERE id = $1`, u)
	assert.Error(t, err, "users with memberships are disabled, never deleted")
	_, err = f.pool.Exec(f.ctx, `DELETE FROM companies WHERE id = $1`, f.compA1)
	assert.Error(t, err)
}
