package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
)

// An organization assignment is not a substitute for verified membership.
func TestOrganizationPermissionScope(t *testing.T) {
	r := role(t, "org_admin", domain.Grant{Permission: "configuration.manage"})
	roles := map[uuid.UUID]domain.Role{r.ID: r}
	a := domain.Assignment{OrganizationID: org, UserID: user, RoleID: r.ID,
		Scope: domain.ScopeRef{Level: domain.LevelOrganization, ID: org}}
	target := domain.ScopePath{OrganizationID: org}
	decide := func(p domain.ScopePath, a domain.Assignment) error {
		return domain.Decide(req("configuration.manage", p, nil), []domain.Assignment{a}, roles)
	}
	assert.NoError(t, decide(target, a))
	assert.ErrorIs(t, decide(domain.ScopePath{}, a), domain.ErrForbidden, "global is not organization")
	assert.ErrorIs(t, decide(domain.ScopePath{OrganizationID: otherOrg}, a), domain.ErrForbidden)
	assert.ErrorIs(t, decide(domain.ScopePath{OrganizationID: org, CompanyID: company}, a), domain.ErrForbidden, "ambiguous target")
	assert.ErrorIs(t, decide(atWarehouse, a), domain.ErrForbidden, "no membership evidence")
	a.MemberCompanyID = company
	assert.NoError(t, decide(atWarehouse, a))
	assert.ErrorIs(t, decide(domain.ScopePath{CompanyID: uuid.New()}, a), domain.ErrForbidden, "evidence scoped to one company")
	a.Scope.ID = otherOrg
	assert.ErrorIs(t, decide(atWarehouse, a), domain.ErrForbidden, "foreign organization")
	a.Scope.ID = org
	a.CompanyID = company
	assert.ErrorIs(t, decide(atWarehouse, a), domain.ErrForbidden, "stored organization assignment cannot carry company")
	assert.ErrorIs(t, decide(target, assign(r, domain.LevelCompany, company)), domain.ErrForbidden, "no upward inheritance")
}

func TestOrganizationPermissionRequiresValidActorAndAmount(t *testing.T) {
	r := role(t, "org_admin", domain.Grant{Permission: "configuration.manage", Limit: q("10")})
	a := domain.Assignment{OrganizationID: org, UserID: user, RoleID: r.ID, Scope: domain.ScopeRef{Level: domain.LevelOrganization, ID: org}}
	roles := map[uuid.UUID]domain.Role{r.ID: r}
	request := req("configuration.manage", domain.ScopePath{OrganizationID: org}, q("10"))
	assert.NoError(t, domain.Decide(request, []domain.Assignment{a}, roles))
	request.Amount = q("10.000001")
	assert.ErrorIs(t, domain.Decide(request, []domain.Assignment{a}, roles), domain.ErrLimitExceeded)
	request.Amount = q("-1")
	assert.ErrorIs(t, domain.Decide(request, []domain.Assignment{a}, roles), domain.ErrForbidden)
	request.Amount = q("1")
	request.UserID = uuid.Nil
	a.UserID = uuid.Nil
	assert.ErrorIs(t, domain.Decide(request, []domain.Assignment{a}, roles), domain.ErrForbidden)
}
