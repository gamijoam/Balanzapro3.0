package pgstore_test

import (
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrganizationRoleRequiresLiveMembershipAndActiveUser(t *testing.T) {
	e := setup(t)
	u := e.user(t, e.orgA, "org_admin", e.compA)
	r := e.role(t, e.orgA, "org_admin", map[string]string{"configuration.manage": ""})
	e.exec(t, `INSERT INTO permission_assignments (organization_id,user_id,role_id,scope_level)
        VALUES ($1,$2,$3,'organization')`, e.orgA, u, r)
	authorize := func(p domain.ScopePath) error {
		return e.svc.Authorize(e.ctx, e.req(u, "configuration.manage", p, nil))
	}
	org := domain.ScopePath{OrganizationID: e.orgA}
	company := domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA, WarehouseID: e.whA}
	require.NoError(t, authorize(org))
	require.NoError(t, authorize(company))
	assert.ErrorIs(t, authorize(domain.ScopePath{}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{OrganizationID: e.orgB}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{CompanyID: e.compA2}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{CompanyID: e.compB}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA2, WarehouseID: e.whA}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{CompanyID: e.compA, BranchID: uuid.New()}), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(domain.ScopePath{CompanyID: e.compA, WarehouseID: uuid.New()}), domain.ErrForbidden)
	e.exec(t, "DELETE FROM identity_memberships WHERE user_id=$1 AND company_id=$2", u, e.compA)
	assert.ErrorIs(t, authorize(company), domain.ErrForbidden, "revoked membership")
	require.NoError(t, authorize(org), "organization grant remains valid without a company membership")
	e.exec(t, "INSERT INTO identity_memberships(organization_id,user_id,company_id) VALUES($1,$2,$3)", e.orgA, u, e.compA)
	require.NoError(t, authorize(company))
	e.exec(t, "UPDATE identity_users SET status='disabled',disabled_at=now() WHERE id=$1", u)
	assert.ErrorIs(t, authorize(org), domain.ErrForbidden)
	assert.ErrorIs(t, authorize(company), domain.ErrForbidden)
}

func TestCompanyRoleCannotTrustForgedScopeOrDisabledUser(t *testing.T) {
	e := setup(t)
	r := e.role(t, e.orgA, "branch_staff", map[string]string{"sales.sell": ""})
	e.assign(t, e.orgA, e.cashier, r, e.compA, "branch", &e.branchA, nil)
	request := func(p domain.ScopePath) error {
		return e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.sell", p, nil))
	}
	require.NoError(t, request(domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA, WarehouseID: e.whA}))
	other := uuid.New()
	e.exec(t, "INSERT INTO warehouses(id,company_id,branch_id,name) VALUES($1,$2,$3,'Otro')", other, e.compA, e.branchA2)
	assert.ErrorIs(t, request(domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA, WarehouseID: other}), domain.ErrForbidden)
	e.exec(t, "UPDATE identity_users SET status='disabled',disabled_at=now() WHERE id=$1", e.cashier)
	assert.ErrorIs(t, request(domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA}), domain.ErrForbidden)
}
