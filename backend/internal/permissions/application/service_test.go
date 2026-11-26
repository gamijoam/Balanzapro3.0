package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// memStore implements the ports in memory; the PostgreSQL adapter (#84)
// is checked against a real database.
type memStore struct {
	mu      sync.Mutex
	assigns []domain.Assignment
	roles   map[uuid.UUID]domain.Role
	auths   map[uuid.UUID]domain.Authorization
	audit   []application.AuditEvent
}

func (s *memStore) InTx(_ context.Context, fn func(application.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s)
}
func (s *memStore) Assignments(_ context.Context, org, user, company uuid.UUID) ([]domain.Assignment, error) {
	var out []domain.Assignment
	for _, a := range s.assigns {
		if a.OrganizationID == org && a.UserID == user && a.CompanyID == company {
			out = append(out, a)
		}
	}
	return out, nil
}
func (s *memStore) Roles(_ context.Context, org uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.Role, error) {
	out := map[uuid.UUID]domain.Role{}
	for _, id := range ids {
		if r, ok := s.roles[id]; ok && r.OrganizationID == org {
			out[id] = r
		}
	}
	return out, nil
}
func (s *memStore) InsertAuthorization(_ context.Context, a domain.Authorization) error {
	s.auths[a.ID] = a
	return nil
}
func (s *memStore) LockAuthorization(_ context.Context, org, id uuid.UUID) (domain.Authorization, error) {
	a, ok := s.auths[id]
	if !ok || a.Request.OrganizationID != org {
		return domain.Authorization{}, application.ErrNotFound
	}
	return a, nil
}
func (s *memStore) MarkAuthorizationUsed(_ context.Context, a domain.Authorization) error {
	s.auths[a.ID] = a
	return nil
}
func (s *memStore) AppendAudit(_ context.Context, ev application.AuditEvent) error {
	s.audit = append(s.audit, ev)
	return nil
}

type env struct {
	svc                      *application.Service
	store                    *memStore
	org, company, branch, wh uuid.UUID
	cashier, supervisor      uuid.UUID
	now                      time.Time
}

func q(s string) *qty.Q { v := qty.MustParse(s); return &v }

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{store: &memStore{roles: map[uuid.UUID]domain.Role{}, auths: map[uuid.UUID]domain.Authorization{}},
		org: uuid.New(), company: uuid.New(), branch: uuid.New(), wh: uuid.New(),
		cashier: uuid.New(), supervisor: uuid.New(), now: time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)}
	mk := func(code string, limit string) domain.Role {
		r, err := domain.NewRole(uuid.New(), e.org, code, []domain.Grant{{Permission: "sales.discount", Limit: q(limit)}})
		require.NoError(t, err)
		e.store.roles[r.ID] = r
		return r
	}
	cajero, super := mk("cajero", "10"), mk("supervisor", "50")
	e.store.assigns = []domain.Assignment{
		{OrganizationID: e.org, UserID: e.cashier, RoleID: cajero.ID, CompanyID: e.company, Scope: domain.ScopeRef{Level: domain.LevelCompany, ID: e.company}},
		{OrganizationID: e.org, UserID: e.supervisor, RoleID: super.ID, CompanyID: e.company, Scope: domain.ScopeRef{Level: domain.LevelBranch, ID: e.branch}},
	}
	e.svc = application.NewService(e.store, func() time.Time { return e.now })
	return e
}

func (e *env) request(user uuid.UUID, amount string) domain.Request {
	return domain.Request{OrganizationID: e.org, UserID: user, Permission: "sales.discount",
		Target: domain.ScopePath{CompanyID: e.company, BranchID: e.branch, WarehouseID: e.wh}, Amount: q(amount)}
}

func TestAuthorizeUsesStoredAssignments(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	assert.NoError(t, e.svc.Authorize(ctx, e.request(e.cashier, "10")))
	assert.ErrorIs(t, e.svc.Authorize(ctx, e.request(e.cashier, "25")), domain.ErrLimitExceeded)
	stranger := e.request(uuid.New(), "1")
	assert.ErrorIs(t, e.svc.Authorize(ctx, stranger), domain.ErrForbidden)
	var _ application.Authorizer = e.svc // the ADR-010 port
}

func TestSupervisorFlowIsBoundSingleUseAndAudited(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	terminal, session, corr := uuid.New(), uuid.New(), uuid.New()
	want := e.request(e.cashier, "25")

	a, err := e.svc.ApproveBySupervisor(ctx, application.ApproveCommand{ID: uuid.New(), SupervisorID: e.supervisor,
		Request: want, EntityRef: "sale:42", TerminalID: terminal, SessionID: session, TTL: 2 * time.Minute, CorrelationID: corr})
	require.NoError(t, err)
	require.Len(t, e.store.audit, 1)
	assert.Equal(t, "permissions.authorization.grant", e.store.audit[0].Action)
	assert.Equal(t, e.supervisor, e.store.audit[0].ActorUserID)
	assert.Equal(t, corr, e.store.audit[0].CorrelationID)

	red := domain.Redemption{Request: want, EntityRef: "sale:42", TerminalID: terminal, SessionID: session}
	require.NoError(t, e.svc.Redeem(ctx, e.org, a.ID, red, corr))
	require.Len(t, e.store.audit, 2)
	assert.Equal(t, "permissions.authorization.redeem", e.store.audit[1].Action)
	assert.Equal(t, e.cashier, e.store.audit[1].ActorUserID)

	assert.ErrorIs(t, e.svc.Redeem(ctx, e.org, a.ID, red, corr), domain.ErrAuthorizationUsed, "SEC-02")
	assert.ErrorIs(t, e.svc.Redeem(ctx, uuid.New(), a.ID, red, corr), domain.ErrForbidden, "other organization")
	assert.ErrorIs(t, e.svc.Redeem(ctx, e.org, uuid.New(), red, corr), domain.ErrForbidden, "unknown authorization")
}

func TestSupervisorMustHoldThePermissionAndLimit(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	cmd := application.ApproveCommand{ID: uuid.New(), SupervisorID: e.supervisor, Request: e.request(e.cashier, "60"),
		EntityRef: "sale:1", TerminalID: uuid.New(), SessionID: uuid.New(), TTL: time.Minute, CorrelationID: uuid.New()}
	_, err := e.svc.ApproveBySupervisor(ctx, cmd)
	assert.ErrorIs(t, err, domain.ErrLimitExceeded, "60 is above the supervisor's limit of 50")

	cmd.SupervisorID, cmd.Request = e.cashier, e.request(e.cashier, "5")
	_, err = e.svc.ApproveBySupervisor(ctx, cmd)
	assert.ErrorIs(t, err, domain.ErrForbidden, "no self-approval")
	assert.Empty(t, e.store.audit, "nothing recorded for refused approvals")

	e.now = e.now.Add(time.Hour)
	ok := cmd
	ok.SupervisorID, ok.Request, ok.ID = e.supervisor, e.request(e.cashier, "20"), uuid.New()
	a, err := e.svc.ApproveBySupervisor(ctx, ok)
	require.NoError(t, err)
	e.now = e.now.Add(2 * time.Minute)
	err = e.svc.Redeem(ctx, e.org, a.ID, domain.Redemption{Request: ok.Request, EntityRef: ok.EntityRef,
		TerminalID: ok.TerminalID, SessionID: ok.SessionID}, uuid.New())
	assert.ErrorIs(t, err, domain.ErrAuthorizationExpired)
}
