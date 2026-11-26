package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/domain"
	permdomain "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
)

// ADR-023 use cases with in-memory fakes. Synthetic data only.

type fakeStore struct {
	orgOf   map[uuid.UUID]uuid.UUID
	states  map[uuid.UUID]domain.State
	audit   []application.AuditEvent
	saves   int
	failOn  string
	commits int
}

type fakeTx struct {
	s       *fakeStore
	pending map[uuid.UUID]domain.State
	audit   []application.AuditEvent
}

func (s *fakeStore) InTx(_ context.Context, fn func(application.Tx) error) error {
	tx := &fakeTx{s: s, pending: map[uuid.UUID]domain.State{}}
	if err := fn(tx); err != nil {
		return err
	}
	for c, st := range tx.pending {
		s.states[c] = st
	}
	s.audit = append(s.audit, tx.audit...)
	s.commits++
	return nil
}

func (t *fakeTx) LockModules(_ context.Context, org, company uuid.UUID) (domain.State, error) {
	if t.s.orgOf[company] != org {
		return nil, application.ErrNotFound
	}
	out := domain.State{}
	for k, v := range t.s.states[company] {
		out[k] = v
	}
	return out, nil
}

func (t *fakeTx) SaveModule(_ context.Context, company uuid.UUID, module string, active bool) error {
	if t.s.failOn == "save" {
		return errors.New("boom")
	}
	st := t.pending[company]
	if st == nil {
		st = domain.State{}
		for k, v := range t.s.states[company] {
			st[k] = v
		}
	}
	st[module] = active
	t.pending[company] = st
	t.s.saves++
	return nil
}

func (t *fakeTx) AppendAudit(_ context.Context, ev application.AuditEvent) error {
	t.audit = append(t.audit, ev)
	return nil
}

type fakeAuthz struct {
	allowed map[uuid.UUID]bool
	got     []permdomain.Request
}

func (a *fakeAuthz) Authorize(_ context.Context, r permdomain.Request) error {
	a.got = append(a.got, r)
	if a.allowed[r.UserID] {
		return nil
	}
	return permdomain.ErrForbidden
}

type fakeLicenses map[uuid.UUID]domain.Entitlements

func (f fakeLicenses) Entitlements(_ context.Context, _, company uuid.UUID) (domain.Entitlements, error) {
	return f[company], nil
}

type env struct {
	svc          *application.Service
	store        *fakeStore
	authz        *fakeAuthz
	org, orgB    uuid.UUID
	compA, compB uuid.UUID
	admin, clerk uuid.UUID
}

func setup(t *testing.T) env {
	t.Helper()
	e := env{org: uuid.New(), orgB: uuid.New(), compA: uuid.New(), compB: uuid.New(), admin: uuid.New(), clerk: uuid.New()}
	e.store = &fakeStore{orgOf: map[uuid.UUID]uuid.UUID{e.compA: e.org, e.compB: e.org}, states: map[uuid.UUID]domain.State{}}
	e.authz = &fakeAuthz{allowed: map[uuid.UUID]bool{e.admin: true}}
	c, err := domain.NewCatalog(append(domain.BaseManifests(),
		domain.Manifest{ID: "inventory", Version: "1.0.0"},
		domain.Manifest{ID: "sales", Version: "1.0.0", Requires: []string{"inventory"}})...)
	require.NoError(t, err)
	lic := fakeLicenses{e.compA: {"inventory": true, "sales": true}, e.compB: {"inventory": true}}
	e.svc = application.NewService(c, e.store, e.authz, lic)
	return e
}

func (e env) cmd(actor, company uuid.UUID, module string) application.ChangeCommand {
	return application.ChangeCommand{OrganizationID: e.org, ActorUserID: actor, CompanyID: company,
		ModuleID: module, Reason: "alta del módulo", CorrelationID: uuid.New()}
}

func TestActivateChecksPermissionLicenseAndAudits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	err := e.svc.Activate(ctx, e.cmd(e.clerk, e.compA, "inventory"))
	assert.ErrorIs(t, err, permdomain.ErrForbidden)
	require.NotEmpty(t, e.authz.got)
	assert.Equal(t, permdomain.Request{OrganizationID: e.org, UserID: e.clerk, Permission: domain.PermissionManage,
		Target: permdomain.ScopePath{CompanyID: e.compA}}, e.authz.got[0], "permission is checked at company scope")

	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "inventory")))
	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "sales")))
	assert.True(t, e.store.states[e.compA].IsActive("sales"))
	require.Len(t, e.store.audit, 2)
	ev := e.store.audit[1]
	assert.Equal(t, "modules.activate", ev.Action)
	assert.Equal(t, "company_module", ev.EntityType)
	assert.Equal(t, "sales", ev.EntityID)
	assert.Equal(t, e.admin, ev.ActorUserID)
	assert.Equal(t, e.compA, ev.CompanyID)
	assert.Equal(t, "alta del módulo", ev.Reason)

	assert.ErrorIs(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compB, "sales")), domain.ErrNotLicensed,
		"each company has its own license")
	assert.False(t, e.store.states[e.compB].IsActive("inventory"), "company B is unaffected by company A (SEC-01)")
}

func TestActivateNoOpWritesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "inventory")))
	saves, events := e.store.saves, len(e.store.audit)
	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "inventory")))
	assert.Equal(t, saves, e.store.saves)
	assert.Len(t, e.store.audit, events)
}

func TestDeactivateKeepsRulesAndAudits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "inventory")))
	require.NoError(t, e.svc.Activate(ctx, e.cmd(e.admin, e.compA, "sales")))

	assert.ErrorIs(t, e.svc.Deactivate(ctx, e.cmd(e.admin, e.compA, "inventory")), domain.ErrDependencyInUse)
	assert.ErrorIs(t, e.svc.Deactivate(ctx, e.cmd(e.admin, e.compA, "identity")), domain.ErrBaseModule)
	assert.ErrorIs(t, e.svc.Deactivate(ctx, e.cmd(e.clerk, e.compA, "sales")), permdomain.ErrForbidden)
	require.NoError(t, e.svc.Deactivate(ctx, e.cmd(e.admin, e.compA, "sales")))
	assert.False(t, e.store.states[e.compA].IsActive("sales"))
	assert.Equal(t, "modules.deactivate", e.store.audit[len(e.store.audit)-1].Action)
}

func TestCommandValidationAndForeignCompany(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.cmd(e.admin, e.compA, "inventory")
	c.Reason = "  "
	assert.ErrorIs(t, e.svc.Activate(ctx, c), domain.ErrInvalidArgument)
	c = e.cmd(e.admin, e.compA, "inventory")
	c.CorrelationID = uuid.Nil
	assert.ErrorIs(t, e.svc.Activate(ctx, c), domain.ErrInvalidArgument)

	c = e.cmd(e.admin, e.compA, "inventory")
	c.OrganizationID = e.orgB
	e.authz.allowed[e.admin] = true
	assert.ErrorIs(t, e.svc.Activate(ctx, c), permdomain.ErrForbidden, "a company of another organization is reported as forbidden")
}

func TestStorageFailureRollsBack(t *testing.T) {
	e := setup(t)
	e.store.failOn = "save"
	assert.Error(t, e.svc.Activate(context.Background(), e.cmd(e.admin, e.compA, "inventory")))
	assert.Empty(t, e.store.audit)
	assert.Zero(t, e.store.commits)
}
