package pgstore_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
)

func TestPasswordStoreIsolationRotationAndSanitizedErrors(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	orgs := organization.NewStore(pool)
	org, err := orgs.CreateOrganization(ctx, "A")
	require.NoError(t, err)
	foreign, err := orgs.CreateOrganization(ctx, "B")
	require.NoError(t, err)
	user := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO identity_users(id,organization_id,login,display_name) VALUES ($1,$2,'synthetic','Synthetic')", user, org)
	require.NoError(t, err)
	var store application.PasswordStore = pgstore.New(pool)
	// Syntactic fixtures only; not usable password hashes.
	a := "$argon2id$" + "v=19$m=65536,t=3,p=1$" + "c2FsdA$" + "dGVzdA"
	b := "$argon2id$" + "v=19$m=65536,t=3,p=1$" + "c2FsdDI$" + "dGVzdDI"
	pin := "$argon2id$" + "v=19$m=65536,t=3,p=1$" + "c2FsdDM$" + "dGVzdDM"
	_, err = store.LoadPassword(ctx, org, user)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.ErrorIs(t, store.InsertPassword(ctx, foreign, user, a), domain.ErrNotFound)
	err = store.InsertPassword(ctx, org, user, "invalid synthetic value")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "invalid synthetic value")
	require.NoError(t, store.InsertPassword(ctx, org, user, a))
	assert.ErrorIs(t, store.InsertPassword(ctx, org, user, b), domain.ErrDuplicate)
	got, err := store.LoadPassword(ctx, org, user)
	require.NoError(t, err)
	assert.Equal(t, a, got)
	_, err = store.LoadPassword(ctx, foreign, user)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = pool.Exec(ctx, "UPDATE identity_credentials SET pin_verifier=$2 WHERE user_id=$1", user, pin)
	require.NoError(t, err)
	assert.ErrorIs(t, store.ReplacePassword(ctx, org, user, b, a), application.ErrCredentialConflict)
	require.NoError(t, store.ReplacePassword(ctx, org, user, a, b))
	assert.ErrorIs(t, store.ReplacePassword(ctx, org, user, a, b), application.ErrCredentialConflict)
	var savedPIN string
	require.NoError(t, pool.QueryRow(ctx, "SELECT pin_verifier FROM identity_credentials WHERE user_id=$1", user).Scan(&savedPIN))
	assert.Equal(t, pin, savedPIN)
	got, err = store.LoadPassword(ctx, org, user)
	require.NoError(t, err)
	assert.Equal(t, b, got)
	_, err = pool.Exec(ctx, "UPDATE identity_users SET status='disabled',disabled_at=now() WHERE id=$1", user)
	require.NoError(t, err)
	_, err = store.LoadPassword(ctx, org, user)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.ErrorIs(t, store.ReplacePassword(ctx, org, user, b, a), domain.ErrNotFound)
}
