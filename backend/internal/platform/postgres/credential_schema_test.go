package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

func TestCredentialSchemaScopeAndRollback(t *testing.T) {
	f := identityFixture(t)
	user, err := f.user(t, f.orgA, "credential_test")
	require.NoError(t, err)
	// Syntactically shaped fixtures only, not real or usable verifiers.
	password := "$argon2id$" + "v=19$m=65536,t=3,p=1$" + "c2FsdA$" + "dGVzdA"
	pin := "$argon2id$" + "v=19$m=65536,t=3,p=1$" + "c2FsdDI$" + "dGVzdDI"
	insert := func(org, u uuid.UUID, pass string, p any) error {
		_, err := f.pool.Exec(f.ctx, `INSERT INTO identity_credentials
          (organization_id,user_id,password_verifier,pin_verifier) VALUES ($1,$2,$3,$4)`, org, u, pass, p)
		return err
	}
	require.NoError(t, insert(f.orgA, user, password, pin))
	assert.Error(t, insert(f.orgA, user, password, nil), "one row per user")
	other, err := f.user(t, f.orgB, "foreign_test")
	require.NoError(t, err)
	assert.Error(t, insert(f.orgA, other, password, nil), "foreign user")
	assert.Error(t, insert(f.orgB, other, "not-a-verifier", nil))
	assert.Error(t, insert(f.orgB, other, password, "1234"))
	assert.Error(t, insert(f.orgB, other, password, password), "separate verifiers")
	require.NoError(t, insert(f.orgB, other, password, nil))
	assert.Error(t, postgres.RollbackTo(f.ctx, f.pool, 8), "do not silently discard credentials")
	var count int
	require.NoError(t, f.pool.QueryRow(f.ctx, "SELECT count(*) FROM identity_credentials").Scan(&count))
	assert.Equal(t, 2, count)
	_, err = f.pool.Exec(f.ctx, "DELETE FROM identity_credentials")
	require.NoError(t, err)
	require.NoError(t, postgres.RollbackTo(f.ctx, f.pool, 8))
	require.NoError(t, postgres.Migrate(f.ctx, f.pool))
	require.NoError(t, insert(f.orgA, user, password, nil))
}
