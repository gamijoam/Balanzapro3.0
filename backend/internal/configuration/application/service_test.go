package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
	permdomain "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// ADR-027 use cases with in-memory fakes. Synthetic data only.

var now = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

type fakeStore struct {
	entries []domain.Entry
	audit   []application.AuditEvent
	// scopes maps (company, scope id) pairs that belong together.
	scopes map[[2]uuid.UUID]bool
}

type fakeTx struct {
	s       *fakeStore
	entries []domain.Entry
	audit   []application.AuditEvent
}

func (s *fakeStore) InTx(_ context.Context, fn func(application.Tx) error) error {
	tx := &fakeTx{s: s}
	if err := fn(tx); err != nil {
		return err
	}
	s.entries = append(s.entries, tx.entries...)
	s.audit = append(s.audit, tx.audit...)
	return nil
}

func (t *fakeTx) VerifyScope(_ context.Context, _, company uuid.UUID, sc domain.Scope) error {
	if sc.Level == domain.LevelCompany && sc.ID == company || t.s.scopes[[2]uuid.UUID{company, sc.ID}] {
		return nil
	}
	return application.ErrNotFound
}

func (t *fakeTx) LatestVersion(_ context.Context, _ uuid.UUID, key string, sc domain.Scope) (int, error) {
	v := 0
	for _, e := range t.s.entries {
		if e.Key == key && e.Scope == sc && e.Version > v {
			v = e.Version
		}
	}
	return v, nil
}

func (t *fakeTx) InsertEntry(_ context.Context, _ uuid.UUID, e domain.Entry) error {
	t.entries = append(t.entries, e)
	return nil
}

func (t *fakeTx) Entries(_ context.Context, _ uuid.UUID, key string, scopes []domain.Scope) ([]domain.Entry, error) {
	var out []domain.Entry
	for _, e := range t.s.entries {
		for _, sc := range scopes {
			if e.Key == key && e.Scope == sc {
				out = append(out, e)
			}
		}
	}
	return out, nil
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

type env struct {
	svc                 *application.Service
	store               *fakeStore
	authz               *fakeAuthz
	org, comp, branch   uuid.UUID
	terminal, admin, us uuid.UUID
}

func setup(t *testing.T) env {
	t.Helper()
	e := env{org: uuid.New(), comp: uuid.New(), branch: uuid.New(), terminal: uuid.New(), admin: uuid.New(), us: uuid.New()}
	e.store = &fakeStore{scopes: map[[2]uuid.UUID]bool{{e.comp, e.branch}: true, {e.comp, e.terminal}: true, {e.comp, e.us}: true}}
	e.authz = &fakeAuthz{allowed: map[uuid.UUID]bool{e.admin: true}}
	all := []domain.Level{domain.LevelGlobal, domain.LevelOrganization, domain.LevelCompany, domain.LevelBranch,
		domain.LevelTerminal, domain.LevelUser}
	reg, err := domain.NewRegistry(
		domain.Key{Name: "ui.page_size", Type: domain.TypeInteger, Default: int64(20), Levels: all},
		domain.Key{Name: "sales.max_discount", Type: domain.TypeDecimal, Default: qty.MustParse("0"),
			Levels: []domain.Level{domain.LevelCompany, domain.LevelBranch}})
	require.NoError(t, err)
	e.svc = application.NewService(reg, e.store, e.authz, func() time.Time { return now })
	return e
}

func (e env) set(actor uuid.UUID, key string, sc domain.Scope, v any) application.SetCommand {
	return application.SetCommand{OrganizationID: e.org, CompanyID: e.comp, BranchID: e.branch, ActorUserID: actor,
		Key: key, Scope: sc, Value: v, Reason: "ajuste", CorrelationID: uuid.New()}
}

func TestSetVersionsAuditsAndResolves(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	compScope := domain.Scope{Level: domain.LevelCompany, ID: e.comp}

	require.NoError(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", compScope, int64(40))))
	require.NoError(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", compScope, int64(50))))
	require.Len(t, e.store.entries, 2)
	assert.Equal(t, []int{1, 2}, []int{e.store.entries[0].Version, e.store.entries[1].Version})
	assert.Equal(t, e.admin, e.store.entries[1].ActorUserID)
	assert.Equal(t, now, e.store.entries[1].CreatedAt)

	require.Len(t, e.store.audit, 2)
	ev := e.store.audit[1]
	assert.Equal(t, "configuration.set", ev.Action)
	assert.Equal(t, "config_entry", ev.EntityType)
	assert.Equal(t, "ui.page_size", ev.EntityID)
	assert.Equal(t, e.comp, ev.CompanyID)
	assert.Equal(t, map[string]any{"level": "company", "scope_id": e.comp.String(), "version": 2, "value": "50"}, ev.After)

	assert.Equal(t, permdomain.Request{OrganizationID: e.org, UserID: e.admin, Permission: domain.PermissionManage,
		Target: permdomain.ScopePath{CompanyID: e.comp}}, e.authz.got[0], "company-level writes need company scope")

	got, err := e.svc.Resolve(ctx, "ui.page_size", domain.Chain{OrganizationID: e.org, CompanyID: e.comp, BranchID: e.branch})
	require.NoError(t, err)
	assert.Equal(t, domain.Resolved{Value: int64(50), Source: domain.LevelCompany, Version: 2}, got)
}

func TestSetAtBranchChecksBranchScope(t *testing.T) {
	e := setup(t)
	require.NoError(t, e.svc.Set(context.Background(), e.set(e.admin, "sales.max_discount",
		domain.Scope{Level: domain.LevelBranch, ID: e.branch}, qty.MustParse("7.5"))))
	assert.Equal(t, permdomain.ScopePath{CompanyID: e.comp, BranchID: e.branch}, e.authz.got[0].Target)
}

func TestSetDeniesWithoutPermissionOrForeignScope(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	assert.ErrorIs(t, e.svc.Set(ctx, e.set(e.us, "ui.page_size", domain.Scope{Level: domain.LevelCompany, ID: e.comp}, int64(40))),
		permdomain.ErrForbidden)
	assert.ErrorIs(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelBranch, ID: uuid.New()}, int64(40))),
		domain.ErrInvalidArgument, "the command's branch must match a branch scope")
	foreign := uuid.New()
	c := e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelBranch, ID: foreign}, int64(40))
	c.BranchID = foreign
	assert.ErrorIs(t, e.svc.Set(ctx, c), permdomain.ErrForbidden, "a branch outside the company is reported as forbidden")
	c = e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelUser, ID: uuid.New()}, int64(40))
	assert.ErrorIs(t, e.svc.Set(ctx, c), permdomain.ErrForbidden, "a user outside the company is reported as forbidden")
	asked := len(e.authz.got)
	assert.ErrorIs(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelGlobal}, int64(40))),
		permdomain.ErrForbidden, "global writes are denied until an administrative scope exists")
	assert.ErrorIs(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelOrganization, ID: e.org}, int64(40))),
		permdomain.ErrForbidden, "organization writes are denied until an administrative scope exists")
	assert.Len(t, e.authz.got, asked, "denied before any company role is consulted")
	assert.ErrorIs(t, e.svc.Set(ctx, e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelCompany, ID: e.comp}, "40")),
		domain.ErrInvalidValue)
	c = e.set(e.admin, "ui.page_size", domain.Scope{Level: domain.LevelCompany, ID: e.comp}, int64(40))
	c.CorrelationID = uuid.Nil
	assert.ErrorIs(t, e.svc.Set(ctx, c), domain.ErrInvalidArgument)
	assert.Empty(t, e.store.entries)
	assert.Empty(t, e.store.audit)
}
