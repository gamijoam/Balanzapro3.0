package pgstore_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	perm "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCompanyIsolationAndConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	orgs := organization.NewStore(pool)
	org, err := orgs.CreateOrganization(ctx, "A")
	require.NoError(t, err)
	other, err := orgs.CreateOrganization(ctx, "B")
	require.NoError(t, err)
	company, err := orgs.CreateCompany(ctx, org, "A1")
	require.NoError(t, err)
	foreign, err := orgs.CreateCompany(ctx, other, "B1")
	require.NoError(t, err)
	store := pgstore.New(pool)
	require.ErrorIs(t, store.InTx(ctx, func(tx application.Tx) error {
		_, err := tx.LockModules(ctx, org, foreign)
		return err
	}), application.ErrNotFound)
	require.Error(t, store.InTx(ctx, func(tx application.Tx) error {
		return tx.SaveModule(ctx, company, "inventory", true)
	}))
	const workers = 12
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			results <- store.InTx(ctx, func(tx application.Tx) error {
				state, err := tx.LockModules(ctx, org, company)
				if err != nil {
					return err
				}
				if state.IsActive("inventory") {
					return nil
				}
				return tx.SaveModule(ctx, company, "inventory", true)
			})
		})
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	var version int
	var status string
	require.NoError(t, pool.QueryRow(ctx, "SELECT version,status FROM company_modules WHERE company_id=$1 AND module_id='inventory'", company).Scan(&version, &status))
	assert.Equal(t, 1, version)
	assert.Equal(t, "active", status)
	sentinel := errors.New("synthetic rollback")
	require.ErrorIs(t, store.InTx(ctx, func(tx application.Tx) error {
		if _, err := tx.LockModules(ctx, org, company); err != nil {
			return err
		}
		if err := tx.SaveModule(ctx, company, "inventory", false); err != nil {
			return err
		}
		return sentinel
	}), sentinel)
	require.NoError(t, pool.QueryRow(ctx, "SELECT version,status FROM company_modules WHERE company_id=$1 AND module_id='inventory'", company).Scan(&version, &status))
	assert.Equal(t, 1, version)
	assert.Equal(t, "active", status)
	require.NoError(t, store.InTx(ctx, func(tx application.Tx) error {
		if _, err := tx.LockModules(ctx, org, company); err != nil {
			return err
		}
		return tx.SaveModule(ctx, company, "inventory", false)
	}))
	require.NoError(t, pool.QueryRow(ctx, "SELECT version,status FROM company_modules WHERE company_id=$1 AND module_id='inventory'", company).Scan(&version, &status))
	assert.Equal(t, 2, version)
	assert.Equal(t, "inactive", status)
	require.Error(t, store.InTx(ctx, func(tx application.Tx) error {
		if _, err := tx.LockModules(ctx, org, company); err != nil {
			return err
		}
		return tx.SaveModule(ctx, foreign, "inventory", true)
	}))
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM company_modules WHERE company_id=$1", foreign).Scan(&n))
	assert.Zero(t, n)
}

type authorization struct{ allowed bool }

func (a authorization) Authorize(context.Context, perm.Request) error {
	if !a.allowed {
		return perm.ErrForbidden
	}
	return nil
}

type licenses domain.Entitlements

func (l licenses) Entitlements(context.Context, uuid.UUID, uuid.UUID) (domain.Entitlements, error) {
	return domain.Entitlements(l), nil
}

func TestServiceActivationAndAudit(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	o := organization.NewStore(pool)
	org, err := o.CreateOrganization(ctx, "Org")
	require.NoError(t, err)
	company, err := o.CreateCompany(ctx, org, "Company")
	require.NoError(t, err)
	actor := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO identity_users(id,organization_id,login,display_name) VALUES($1,$2,'fixture','Fixture')", actor, org)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO identity_memberships(organization_id,user_id,company_id) VALUES($1,$2,$3)", org, actor, company)
	require.NoError(t, err)
	cat, err := domain.NewCatalog(append(domain.BaseManifests(), domain.Manifest{ID: "inventory", Version: "1"}, domain.Manifest{ID: "sales", Version: "1", Requires: []string{"inventory"}})...)
	require.NoError(t, err)
	store := pgstore.New(pool)
	ent := licenses{"inventory": true, "sales": true}
	svc := application.NewService(cat, store, authorization{true}, ent)
	c := application.ChangeCommand{OrganizationID: org, CompanyID: company, ActorUserID: actor, ModuleID: "inventory", Reason: "fixture", CorrelationID: uuid.New()}
	assert.ErrorIs(t, application.NewService(cat, store, authorization{false}, ent).Activate(ctx, c), perm.ErrForbidden)
	assert.ErrorIs(t, application.NewService(cat, store, authorization{true}, licenses{}).Activate(ctx, c), domain.ErrNotLicensed)
	sales := c
	sales.ModuleID = "sales"
	sales.CorrelationID = uuid.New()
	assert.ErrorIs(t, svc.Activate(ctx, sales), domain.ErrDependencyInactive)
	require.NoError(t, svc.Activate(ctx, c))
	require.NoError(t, svc.Activate(ctx, c))
	require.NoError(t, svc.Activate(ctx, sales))
	assert.ErrorIs(t, svc.Deactivate(ctx, c), domain.ErrDependencyInUse)
	require.NoError(t, svc.Deactivate(ctx, sales))
	require.NoError(t, svc.Deactivate(ctx, c))
	bad := c
	bad.ActorUserID = uuid.New()
	assert.ErrorIs(t, svc.Activate(ctx, bad), application.ErrNotFound)
	var version, n int
	var status string
	require.NoError(t, pool.QueryRow(ctx, "SELECT status,version FROM company_modules WHERE company_id=$1 AND module_id='inventory'", company).Scan(&status, &version))
	assert.Equal(t, "inactive", status)
	assert.Equal(t, 2, version)
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE company_id=$1", company).Scan(&n))
	assert.Equal(t, 4, n)
}
