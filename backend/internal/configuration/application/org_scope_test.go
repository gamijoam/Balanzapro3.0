package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
	permdomain "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
)

func TestOrganizationConfigurationRequiresExplicitOwnScope(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	// The fake store must explicitly know the organization scope.
	e.store.scopes[[2]uuid.UUID{uuid.Nil, e.org}] = true
	c := e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelOrganization, ID: e.org}, int64(40))
	c.CompanyID, c.BranchID = uuid.Nil, uuid.Nil
	require.NoError(t, e.svc.Set(ctx, c))
	c.Value = int64(50)
	require.NoError(t, e.svc.Set(ctx, c))
	require.Len(t, e.store.entries, 2)
	assert.Equal(t, 2, e.store.entries[1].Version)
	require.Len(t, e.store.audit, 2)
	assert.Equal(t, uuid.Nil, e.store.audit[1].CompanyID)
	assert.Equal(t, e.org, e.store.audit[1].OrganizationID)
	assert.Equal(t, c.CorrelationID, e.store.audit[1].CorrelationID)
	assert.Equal(t, permdomain.Request{OrganizationID: e.org, UserID: e.admin, Permission: domain.PermissionManage,
		Target: permdomain.ScopePath{OrganizationID: e.org}}, e.authz.got[0])
	got, err := e.svc.Resolve(ctx, c.Key, domain.Chain{OrganizationID: e.org, CompanyID: e.comp})
	require.NoError(t, err)
	assert.Equal(t, domain.Resolved{Value: int64(50), Source: domain.LevelOrganization, Version: 2}, got)
	bad := c
	bad.Scope.ID = uuid.New()
	assert.ErrorIs(t, e.svc.Set(ctx, bad), permdomain.ErrForbidden)
	bad = c
	bad.ActorUserID = e.us
	assert.ErrorIs(t, e.svc.Set(ctx, bad), permdomain.ErrForbidden)
	bad = c
	bad.CompanyID = e.comp
	assert.ErrorIs(t, e.svc.Set(ctx, bad), permdomain.ErrForbidden)
	bad = c
	bad.BranchID = e.branch
	assert.ErrorIs(t, e.svc.Set(ctx, bad), permdomain.ErrForbidden)
	bad = c
	bad.Scope = domain.Scope{Level: domain.LevelGlobal}
	assert.ErrorIs(t, e.svc.Set(ctx, bad), permdomain.ErrForbidden)
	assert.Len(t, e.store.entries, 2)
	assert.Len(t, e.store.audit, 2)
}
