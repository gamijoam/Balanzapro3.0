package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

var (
	org, otherOrg = uuid.New(), uuid.New()
	company       = uuid.New()
	branch1       = uuid.New()
	branch2       = uuid.New()
	wh1           = uuid.New() // in branch1
	user          = uuid.New()
)

func q(s string) *qty.Q { v := qty.MustParse(s); return &v }

func role(t *testing.T, code string, grants ...domain.Grant) domain.Role {
	t.Helper()
	r, err := domain.NewRole(uuid.New(), org, code, grants)
	require.NoError(t, err)
	return r
}

func assign(r domain.Role, level domain.ScopeLevel, scope uuid.UUID) domain.Assignment {
	return domain.Assignment{OrganizationID: org, UserID: user, RoleID: r.ID, CompanyID: company,
		Scope: domain.ScopeRef{Level: level, ID: scope}}
}

func req(perm string, target domain.ScopePath, amount *qty.Q) domain.Request {
	return domain.Request{OrganizationID: org, UserID: user, Permission: domain.Permission(perm), Target: target, Amount: amount}
}

var atWarehouse = domain.ScopePath{CompanyID: company, BranchID: branch1, WarehouseID: wh1}

func TestNewRoleValidates(t *testing.T) {
	_, err := domain.NewRole(uuid.New(), org, "cajero", []domain.Grant{{Permission: "sales.checkout"}})
	require.NoError(t, err)
	_, err = domain.NewRole(uuid.New(), org, "Cajero!", nil)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewRole(uuid.New(), org, "x", []domain.Grant{{Permission: "Bad Code"}})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewRole(uuid.New(), org, "x", []domain.Grant{{Permission: "a.b"}, {Permission: "a.b"}})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "duplicate permission")
	_, err = domain.NewRole(uuid.New(), org, "x", []domain.Grant{{Permission: "a.b", Limit: q("-1")}})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "negative limit")
	_, err = domain.NewRole(uuid.Nil, org, "x", nil)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestDenyByDefault(t *testing.T) {
	assert.ErrorIs(t, domain.Decide(req("sales.checkout", atWarehouse, nil), nil, nil), domain.ErrForbidden)
	r := role(t, "cajero", domain.Grant{Permission: "sales.checkout"})
	a := []domain.Assignment{assign(r, domain.LevelCompany, company)}
	roles := map[uuid.UUID]domain.Role{r.ID: r}
	assert.ErrorIs(t, domain.Decide(req("sales.void", atWarehouse, nil), a, roles), domain.ErrForbidden, "permission not granted")
	assert.NoError(t, domain.Decide(req("sales.checkout", atWarehouse, nil), a, roles))
}

// ADR-024: assignments inherit downwards, never upwards or sideways.
func TestScopeInheritance(t *testing.T) {
	r := role(t, "almacenista", domain.Grant{Permission: "inventory.issue"})
	roles := map[uuid.UUID]domain.Role{r.ID: r}

	atBranch := []domain.Assignment{assign(r, domain.LevelBranch, branch1)}
	assert.NoError(t, domain.Decide(req("inventory.issue", atWarehouse, nil), atBranch, roles), "branch -> its warehouse")
	assert.ErrorIs(t, domain.Decide(req("inventory.issue",
		domain.ScopePath{CompanyID: company, BranchID: branch2}, nil), atBranch, roles), domain.ErrForbidden, "sibling branch")
	assert.ErrorIs(t, domain.Decide(req("inventory.issue",
		domain.ScopePath{CompanyID: company}, nil), atBranch, roles), domain.ErrForbidden, "never upwards to the company")

	atWh := []domain.Assignment{assign(r, domain.LevelWarehouse, wh1)}
	assert.NoError(t, domain.Decide(req("inventory.issue", atWarehouse, nil), atWh, roles))
	assert.ErrorIs(t, domain.Decide(req("inventory.issue",
		domain.ScopePath{CompanyID: company, BranchID: branch1}, nil), atWh, roles), domain.ErrForbidden, "never upwards to the branch")

	otherCompany := domain.ScopePath{CompanyID: uuid.New(), BranchID: branch1, WarehouseID: wh1}
	assert.ErrorIs(t, domain.Decide(req("inventory.issue", otherCompany, nil), atBranch, roles), domain.ErrForbidden,
		"SEC-01: an assignment never crosses companies")
}

func TestAssignmentsOfOthersOrForeignRolesAreIgnored(t *testing.T) {
	r := role(t, "cajero", domain.Grant{Permission: "sales.checkout"})
	roles := map[uuid.UUID]domain.Role{r.ID: r}
	other := assign(r, domain.LevelCompany, company)
	other.UserID = uuid.New()
	assert.ErrorIs(t, domain.Decide(req("sales.checkout", atWarehouse, nil), []domain.Assignment{other}, roles), domain.ErrForbidden)

	foreign, err := domain.NewRole(uuid.New(), otherOrg, "cajero", []domain.Grant{{Permission: "sales.checkout"}})
	require.NoError(t, err)
	a := assign(foreign, domain.LevelCompany, company)
	assert.ErrorIs(t, domain.Decide(req("sales.checkout", atWarehouse, nil), []domain.Assignment{a},
		map[uuid.UUID]domain.Role{foreign.ID: foreign}), domain.ErrForbidden, "role of another organization")
	assert.ErrorIs(t, domain.Decide(req("sales.checkout", atWarehouse, nil), []domain.Assignment{assign(r, domain.LevelCompany, company)},
		map[uuid.UUID]domain.Role{}), domain.ErrForbidden, "unknown role")
}

func TestExactLimits(t *testing.T) {
	cajero := role(t, "cajero", domain.Grant{Permission: "sales.discount", Limit: q("10")})
	super := role(t, "supervisor", domain.Grant{Permission: "sales.discount", Limit: q("30.5")})
	roles := map[uuid.UUID]domain.Role{cajero.ID: cajero, super.ID: super}
	onlyCajero := []domain.Assignment{assign(cajero, domain.LevelCompany, company)}

	assert.NoError(t, domain.Decide(req("sales.discount", atWarehouse, q("10")), onlyCajero, roles), "limit is inclusive")
	assert.ErrorIs(t, domain.Decide(req("sales.discount", atWarehouse, q("10.000001")), onlyCajero, roles), domain.ErrLimitExceeded)
	assert.ErrorIs(t, domain.Decide(req("sales.discount", atWarehouse, nil), onlyCajero, roles), domain.ErrLimitExceeded,
		"a limited grant does not allow an action without amount")

	both := append(onlyCajero, assign(super, domain.LevelBranch, branch1))
	assert.NoError(t, domain.Decide(req("sales.discount", atWarehouse, q("30.5")), both, roles), "the most permissive applicable grant wins")

	unlimited := role(t, "propietario", domain.Grant{Permission: "sales.discount"})
	roles[unlimited.ID] = unlimited
	assert.NoError(t, domain.Decide(req("sales.discount", atWarehouse, q("100")),
		[]domain.Assignment{assign(unlimited, domain.LevelCompany, company)}, roles))
}

// ADR-025 / SEC-02: a supervisor authorization is bound to one action and
// can be redeemed once, before it expires, by the same terminal and session.
func TestSupervisorAuthorization(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	terminal, session, supervisor := uuid.New(), uuid.New(), uuid.New()
	scope := domain.Request{OrganizationID: org, UserID: user, Permission: "sales.discount", Target: atWarehouse, Amount: q("25")}
	a, err := domain.NewAuthorization(uuid.New(), supervisor, scope, "sale:42", terminal, session, now, 2*time.Minute)
	require.NoError(t, err)

	redeem := domain.Redemption{Request: scope, EntityRef: "sale:42", TerminalID: terminal, SessionID: session}
	for name, mutate := range map[string]func(*domain.Redemption){
		"other permission": func(r *domain.Redemption) { r.Request.Permission = "sales.void" },
		"other amount":     func(r *domain.Redemption) { r.Request.Amount = q("25.01") },
		"no amount":        func(r *domain.Redemption) { r.Request.Amount = nil },
		"other entity":     func(r *domain.Redemption) { r.EntityRef = "sale:43" },
		"other terminal":   func(r *domain.Redemption) { r.TerminalID = uuid.New() },
		"other session":    func(r *domain.Redemption) { r.SessionID = uuid.New() },
		"other requester":  func(r *domain.Redemption) { r.Request.UserID = uuid.New() },
		"other scope":      func(r *domain.Redemption) { r.Request.Target.WarehouseID = uuid.New() },
	} {
		bad := redeem
		mutate(&bad)
		_, err := a.Redeem(bad, now.Add(time.Second))
		assert.ErrorIs(t, err, domain.ErrForbidden, name)
	}
	_, err = a.Redeem(redeem, now.Add(2*time.Minute))
	assert.ErrorIs(t, err, domain.ErrAuthorizationExpired)

	used, err := a.Redeem(redeem, now.Add(time.Minute))
	require.NoError(t, err)
	require.NotNil(t, used.UsedAt)
	_, err = used.Redeem(redeem, now.Add(time.Minute))
	assert.ErrorIs(t, err, domain.ErrAuthorizationUsed, "SEC-02: never reused")

	_, err = domain.NewAuthorization(uuid.New(), user, scope, "sale:42", terminal, session, now, time.Minute)
	assert.ErrorIs(t, err, domain.ErrForbidden, "a user cannot authorize themselves")
	_, err = domain.NewAuthorization(uuid.New(), supervisor, scope, "sale:42", terminal, session, now, 10*time.Minute)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "ttl above the safe maximum")
}
