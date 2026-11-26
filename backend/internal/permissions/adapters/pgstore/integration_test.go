package pgstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// BP-009 subtask 3 (#84): permission ports on real PostgreSQL. Roles and
// assignments are seeded with SQL because their admin API is #85.
// Synthetic data only.

var now = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

type env struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	store        *pgstore.Store
	svc          *application.Service
	orgA, orgB   uuid.UUID
	compA, compB uuid.UUID
	compA2       uuid.UUID
	branchA      uuid.UUID
	branchA2     uuid.UUID
	whA          uuid.UUID // in branchA
	cashier      uuid.UUID
	supervisor   uuid.UUID
	userB        uuid.UUID
}

func setup(t *testing.T) env {
	t.Helper()
	pool := pgtest.New(t)
	ctx := context.Background()
	org := organization.NewStore(pool)
	e := env{ctx: ctx, pool: pool, store: pgstore.New(pool)}
	e.svc = application.NewService(e.store, func() time.Time { return now })
	must := func(id uuid.UUID, err error) uuid.UUID { require.NoError(t, err); return id }
	e.orgA = must(org.CreateOrganization(ctx, "Org A"))
	e.orgB = must(org.CreateOrganization(ctx, "Org B"))
	e.compA = must(org.CreateCompany(ctx, e.orgA, "Empresa A"))
	e.compA2 = must(org.CreateCompany(ctx, e.orgA, "Empresa A2"))
	e.compB = must(org.CreateCompany(ctx, e.orgB, "Empresa B"))
	e.branchA = must(org.CreateBranch(ctx, e.compA, "Sucursal 1"))
	e.branchA2 = must(org.CreateBranch(ctx, e.compA, "Sucursal 2"))
	e.whA = must(org.CreateWarehouse(ctx, e.compA, e.branchA, "Almacén 1"))
	e.cashier = e.user(t, e.orgA, "cajero", e.compA, e.compA2)
	e.supervisor = e.user(t, e.orgA, "supervisor", e.compA)
	e.userB = e.user(t, e.orgB, "otro", e.compB)
	return e
}

func (e env) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := e.pool.Exec(e.ctx, sql, args...)
	require.NoError(t, err)
}

func (e env) user(t *testing.T, org uuid.UUID, login string, companies ...uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	e.exec(t, `INSERT INTO identity_users (id, organization_id, login, display_name) VALUES ($1, $2, $3, 'Sintético')`, id, org, login)
	for _, c := range companies {
		e.exec(t, `INSERT INTO identity_memberships (organization_id, user_id, company_id) VALUES ($1, $2, $3)`, org, id, c)
	}
	return id
}

// role inserts a role with grants; a "" limit means unlimited.
func (e env) role(t *testing.T, org uuid.UUID, code string, grants map[string]string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	e.exec(t, `INSERT INTO permission_roles (id, organization_id, code) VALUES ($1, $2, $3)`, id, org, code)
	for p, l := range grants {
		var limit *string
		if l != "" {
			limit = &l
		}
		e.exec(t, `INSERT INTO permission_role_grants (organization_id, role_id, permission, limit_value)
			VALUES ($1, $2, $3, $4)`, org, id, p, limit)
	}
	return id
}

func (e env) assign(t *testing.T, org, user, role, company uuid.UUID, level string, branch, wh *uuid.UUID) {
	t.Helper()
	e.exec(t, `INSERT INTO permission_assignments (organization_id, user_id, role_id, company_id, scope_level, branch_id, warehouse_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, org, user, role, company, level, branch, wh)
}

func q(s string) *qty.Q { v := qty.MustParse(s); return &v }

func (e env) req(user uuid.UUID, p string, target domain.ScopePath, amount *qty.Q) domain.Request {
	return domain.Request{OrganizationID: e.orgA, UserID: user, Permission: domain.Permission(p), Target: target, Amount: amount}
}

func TestAuthorizeReadsRolesAndInheritanceFromPostgres(t *testing.T) {
	e := setup(t)
	cashierRole := e.role(t, e.orgA, "cajero", map[string]string{"sales.discount": "10", "sales.sell": ""})
	e.assign(t, e.orgA, e.cashier, cashierRole, e.compA, "branch", &e.branchA, nil)

	atWarehouse := domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA, WarehouseID: e.whA}
	assert.NoError(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.sell", atWarehouse, nil)), "branch role inherits down to its warehouse")
	assert.NoError(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.discount", atWarehouse, q("10.000000"))), "exact limit")
	assert.ErrorIs(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.discount", atWarehouse, q("10.000001"))), domain.ErrLimitExceeded)
	assert.ErrorIs(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.sell",
		domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA2}, nil)), domain.ErrForbidden, "sibling branch")
	assert.ErrorIs(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.sell",
		domain.ScopePath{CompanyID: e.compA}, nil)), domain.ErrForbidden, "no upward inheritance")
	assert.ErrorIs(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "sales.sell",
		domain.ScopePath{CompanyID: e.compA2}, nil)), domain.ErrForbidden, "another company of the same organization")
	assert.ErrorIs(t, e.svc.Authorize(e.ctx, e.req(e.cashier, "inventory.adjust", atWarehouse, nil)), domain.ErrForbidden, "default deny")
}

func TestReadsAreIsolatedByOrganization(t *testing.T) {
	e := setup(t)
	roleB := e.role(t, e.orgB, "admin", map[string]string{"sales.sell": ""})
	e.assign(t, e.orgB, e.userB, roleB, e.compB, "company", nil, nil)

	require.NoError(t, e.store.InTx(e.ctx, func(tx application.Tx) error {
		as, err := tx.Assignments(e.ctx, e.orgA, e.userB, e.compB)
		require.NoError(t, err)
		assert.Empty(t, as, "assignments of another organization are invisible")
		roles, err := tx.Roles(e.ctx, e.orgA, []uuid.UUID{roleB})
		require.NoError(t, err)
		assert.Empty(t, roles, "roles of another organization are invisible")
		_, err = tx.LockAuthorization(e.ctx, e.orgA, uuid.New())
		assert.ErrorIs(t, err, application.ErrNotFound)
		return nil
	}))
	err := e.svc.Authorize(e.ctx, domain.Request{OrganizationID: e.orgA, UserID: e.userB, Permission: "sales.sell",
		Target: domain.ScopePath{CompanyID: e.compB}})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestSchemaRejectsCrossScopeAssignments(t *testing.T) {
	e := setup(t)
	roleA := e.role(t, e.orgA, "cajero", map[string]string{"sales.sell": ""})
	roleB := e.role(t, e.orgB, "cajero", nil)
	ins := `INSERT INTO permission_assignments (organization_id, user_id, role_id, company_id, scope_level, branch_id, warehouse_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	bad := func(msg string, args ...any) {
		t.Helper()
		_, err := e.pool.Exec(e.ctx, ins, args...)
		assert.Error(t, err, msg)
	}
	bad("role of another organization", e.orgA, e.cashier, roleB, e.compA, "company", nil, nil)
	bad("company of another organization", e.orgA, e.cashier, roleA, e.compB, "company", nil, nil)
	bad("user without membership in the company", e.orgA, e.supervisor, roleA, e.compA2, "company", nil, nil)
	bad("company level carries no branch", e.orgA, e.cashier, roleA, e.compA, "company", e.branchA, nil)
	bad("branch level needs a branch", e.orgA, e.cashier, roleA, e.compA, "branch", nil, nil)
	bad("warehouse level needs only a warehouse", e.orgA, e.cashier, roleA, e.compA, "warehouse", e.branchA, e.whA)
	bad("branch of another company", e.orgA, e.cashier, roleA, e.compA2, "branch", e.branchA, nil)
	bad("unknown level", e.orgA, e.cashier, roleA, e.compA, "terminal", nil, nil)
	e.assign(t, e.orgA, e.cashier, roleA, e.compA, "warehouse", nil, &e.whA)
	bad("duplicate assignment", e.orgA, e.cashier, roleA, e.compA, "warehouse", nil, e.whA)

	_, err := e.pool.Exec(e.ctx, `INSERT INTO permission_role_grants (organization_id, role_id, permission, limit_value)
		VALUES ($1, $2, 'sales.discount', -1)`, e.orgA, roleA)
	assert.Error(t, err, "negative limit")
	_, err = e.pool.Exec(e.ctx, `INSERT INTO permission_role_grants (organization_id, role_id, permission)
		VALUES ($1, $2, 'Sales')`, e.orgA, roleA)
	assert.Error(t, err, "malformed permission code")
	_, err = e.pool.Exec(e.ctx, `INSERT INTO permission_roles (id, organization_id, code) VALUES ($1, $2, 'cajero')`, uuid.New(), e.orgA)
	assert.Error(t, err, "role codes are unique per organization")
}

func (e env) approve(t *testing.T) (domain.Authorization, domain.Redemption) {
	t.Helper()
	e.assign(t, e.orgA, e.cashier, e.role(t, e.orgA, "cajero", map[string]string{"sales.discount": "10"}), e.compA, "company", nil, nil)
	e.assign(t, e.orgA, e.supervisor, e.role(t, e.orgA, "supervisor", map[string]string{"sales.discount": "50"}), e.compA, "company", nil, nil)
	r := e.req(e.cashier, "sales.discount", domain.ScopePath{CompanyID: e.compA, BranchID: e.branchA}, q("25"))
	require.ErrorIs(t, e.svc.Authorize(e.ctx, r), domain.ErrLimitExceeded)
	a, err := e.svc.ApproveBySupervisor(e.ctx, application.ApproveCommand{ID: uuid.New(), SupervisorID: e.supervisor,
		Request: r, EntityRef: "sale:42", TerminalID: uuid.New(), SessionID: uuid.New(), TTL: 2 * time.Minute, CorrelationID: uuid.New()})
	require.NoError(t, err)
	return a, domain.Redemption{Request: r, EntityRef: a.EntityRef, TerminalID: a.TerminalID, SessionID: a.SessionID}
}

func TestSupervisorAuthorizationRoundTripAndAudit(t *testing.T) {
	e := setup(t)
	a, red := e.approve(t)
	require.NoError(t, e.store.InTx(e.ctx, func(tx application.Tx) error {
		got, err := tx.LockAuthorization(e.ctx, e.orgA, a.ID)
		require.NoError(t, err)
		assert.Equal(t, a.Request.Amount.String(), got.Request.Amount.String())
		got.Request.Amount, a.Request.Amount = nil, nil
		assert.Equal(t, a, got, "every bound field survives the round trip")
		return nil
	}))
	corr := uuid.New()
	require.NoError(t, e.svc.Redeem(e.ctx, e.orgA, a.ID, red, corr))
	assert.ErrorIs(t, e.svc.Redeem(e.ctx, e.orgA, a.ID, red, uuid.New()), domain.ErrAuthorizationUsed)

	var action, entity string
	var actor uuid.UUID
	var after []byte
	require.NoError(t, e.pool.QueryRow(e.ctx, `SELECT action, entity_id, actor_user_id, after FROM audit_events
		WHERE correlation_id = $1 AND company_id = $2`, corr, e.compA).Scan(&action, &entity, &actor, &after))
	assert.Equal(t, "permissions.authorization.redeem", action)
	assert.Equal(t, a.ID.String(), entity)
	assert.Equal(t, e.cashier, actor)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(after, &payload))
	assert.Equal(t, "25", payload["amount"])
	assert.Equal(t, e.supervisor.String(), payload["supervisor"])

	var grants int
	var grantActor uuid.UUID
	require.NoError(t, e.pool.QueryRow(e.ctx, `SELECT count(*), min(actor_user_id::text)::uuid FROM audit_events
		WHERE action = 'permissions.authorization.grant' AND entity_id = $1`, a.ID.String()).Scan(&grants, &grantActor))
	assert.Equal(t, 1, grants)
	assert.Equal(t, e.supervisor, grantActor, "the grant is attributed to the supervisor")
}

func TestRedeemIsSingleUseUnderConcurrency(t *testing.T) {
	e := setup(t)
	a, red := e.approve(t)
	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = e.svc.Redeem(e.ctx, e.orgA, a.ID, red, uuid.New())
		}()
	}
	close(start)
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
			continue
		}
		assert.ErrorIs(t, err, domain.ErrAuthorizationUsed)
	}
	assert.Equal(t, 1, ok, "exactly one redemption succeeds (SEC-02)")
	var redeems int
	require.NoError(t, e.pool.QueryRow(e.ctx, `SELECT count(*) FROM audit_events WHERE entity_id = $1
		AND action = 'permissions.authorization.redeem'`, a.ID.String()).Scan(&redeems))
	assert.Equal(t, 1, redeems)
}

func TestRedeemFromAnotherOrganizationIsForbidden(t *testing.T) {
	e := setup(t)
	a, red := e.approve(t)
	err := e.store.InTx(e.ctx, func(tx application.Tx) error { _, err := tx.LockAuthorization(e.ctx, e.orgB, a.ID); return err })
	assert.ErrorIs(t, err, application.ErrNotFound, "storage does not reveal another organization's authorization")
	red.Request.OrganizationID = e.orgB
	err = e.svc.Redeem(e.ctx, e.orgB, a.ID, red, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLockAuthorizationHoldsTheRow(t *testing.T) {
	e := setup(t)
	a, _ := e.approve(t)
	locked, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- e.store.InTx(e.ctx, func(tx application.Tx) error {
			if _, err := tx.LockAuthorization(e.ctx, e.orgA, a.ID); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	ctx, cancel := context.WithTimeout(e.ctx, 300*time.Millisecond)
	defer cancel()
	err := e.store.InTx(ctx, func(tx application.Tx) error { _, err := tx.LockAuthorization(ctx, e.orgA, a.ID); return err })
	assert.Error(t, err, "a second locker waits while the first transaction holds the row")
	close(release)
	require.NoError(t, <-done)
}

func TestAuthorizationRowsAreImmutableExceptSingleUse(t *testing.T) {
	e := setup(t)
	a, _ := e.approve(t)
	for name, sql := range map[string]string{
		"amount":     `UPDATE supervisor_authorizations SET amount = 1000 WHERE id = $1`,
		"expiry":     `UPDATE supervisor_authorizations SET expires_at = expires_at + interval '1 hour' WHERE id = $1`,
		"delete":     `DELETE FROM supervisor_authorizations WHERE id = $1`,
		"supervisor": `UPDATE supervisor_authorizations SET supervisor_user_id = requester_user_id WHERE id = $1`,
	} {
		_, err := e.pool.Exec(e.ctx, sql, a.ID)
		assert.Error(t, err, name)
	}
	e.exec(t, `UPDATE supervisor_authorizations SET used_at = now() WHERE id = $1`, a.ID)
	_, err := e.pool.Exec(e.ctx, `UPDATE supervisor_authorizations SET used_at = NULL WHERE id = $1`, a.ID)
	assert.Error(t, err, "a used authorization cannot be reset")
	_, err = e.pool.Exec(e.ctx, `UPDATE supervisor_authorizations SET used_at = now() + interval '1 second' WHERE id = $1`, a.ID)
	assert.Error(t, err, "used_at is set once")
}

func TestMarkUsedTwiceInStorageFails(t *testing.T) {
	e := setup(t)
	a, _ := e.approve(t)
	used := now
	a.UsedAt = &used
	require.NoError(t, e.store.InTx(e.ctx, func(tx application.Tx) error { return tx.MarkAuthorizationUsed(e.ctx, a) }))
	err := e.store.InTx(e.ctx, func(tx application.Tx) error { return tx.MarkAuthorizationUsed(e.ctx, a) })
	assert.True(t, errors.Is(err, domain.ErrAuthorizationUsed), "got %v", err)
}
