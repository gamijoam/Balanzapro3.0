package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
)

// memStore is an in-memory implementation of the ports for unit tests; the
// PostgreSQL adapter (BP-008 subtask 3) gets the same behaviour checked
// against a real database.
type memStore struct {
	mu        sync.Mutex
	users     map[uuid.UUID]domain.User
	companies map[uuid.UUID]uuid.UUID // company -> organization
	members   map[[2]uuid.UUID]domain.Membership
}

func newMem() *memStore {
	return &memStore{users: map[uuid.UUID]domain.User{}, companies: map[uuid.UUID]uuid.UUID{},
		members: map[[2]uuid.UUID]domain.Membership{}}
}

func (s *memStore) InTx(ctx context.Context, fn func(application.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s)
}

func (s *memStore) InsertUser(_ context.Context, u domain.User) error {
	for _, x := range s.users {
		if x.OrganizationID == u.OrganizationID && x.Login == u.Login {
			return domain.ErrDuplicate
		}
	}
	s.users[u.ID] = u
	return nil
}

func (s *memStore) GetUser(_ context.Context, org, id uuid.UUID) (domain.User, error) {
	u, ok := s.users[id]
	if !ok || u.OrganizationID != org {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (s *memStore) LockUser(ctx context.Context, org, id uuid.UUID) (domain.User, error) {
	return s.GetUser(ctx, org, id) // InTx already serializes the whole store
}

func (s *memStore) UpdateUser(_ context.Context, u domain.User) error {
	s.users[u.ID] = u
	return nil
}

func (s *memStore) CompanyOrganization(_ context.Context, company uuid.UUID) (uuid.UUID, error) {
	o, ok := s.companies[company]
	if !ok {
		return uuid.Nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *memStore) InsertMembership(_ context.Context, m domain.Membership) error {
	k := [2]uuid.UUID{m.UserID, m.CompanyID}
	if _, ok := s.members[k]; ok {
		return domain.ErrDuplicate
	}
	s.members[k] = m
	return nil
}

func (s *memStore) ListMemberships(_ context.Context, org, user uuid.UUID) ([]domain.Membership, error) {
	var out []domain.Membership
	for _, m := range s.members {
		if m.OrganizationID == org && m.UserID == user {
			out = append(out, m)
		}
	}
	return out, nil
}

type env struct {
	svc            *application.Service
	store          *memStore
	orgA, orgB     uuid.UUID
	compA1, compA2 uuid.UUID
	compB1         uuid.UUID
	now            time.Time
}

func setup() env {
	e := env{store: newMem(), orgA: uuid.New(), orgB: uuid.New(), compA1: uuid.New(), compA2: uuid.New(),
		compB1: uuid.New(), now: time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)}
	e.store.companies[e.compA1], e.store.companies[e.compA2], e.store.companies[e.compB1] = e.orgA, e.orgA, e.orgB
	e.svc = application.NewService(e.store, func() time.Time { return e.now })
	return e
}

func (e env) register(t *testing.T, org uuid.UUID, login string) domain.User {
	t.Helper()
	u, err := e.svc.RegisterUser(context.Background(), org,
		application.RegisterUserCommand{ID: uuid.New(), Login: login, DisplayName: "Usuario sintético"})
	require.NoError(t, err)
	return u
}

func TestRegisterUserNormalizesAndRejectsDuplicates(t *testing.T) {
	e := setup()
	u := e.register(t, e.orgA, " Ana ")
	assert.Equal(t, "ana", u.Login)
	_, err := e.svc.RegisterUser(context.Background(), e.orgA,
		application.RegisterUserCommand{ID: uuid.New(), Login: "ANA", DisplayName: "Otra"})
	assert.ErrorIs(t, err, domain.ErrDuplicate)
	e.register(t, e.orgB, "ana") // same login, other organization
	_, err = e.svc.RegisterUser(context.Background(), uuid.Nil,
		application.RegisterUserCommand{ID: uuid.New(), Login: "x", DisplayName: "x"})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestAddMembershipStaysInsideTheOrganization(t *testing.T) {
	e := setup()
	ctx := context.Background()
	u := e.register(t, e.orgA, "ana")
	require.NoError(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compA1))
	assert.ErrorIs(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compA1), domain.ErrDuplicate)
	assert.ErrorIs(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compB1), domain.ErrForbidden, "SEC-01")
	assert.ErrorIs(t, e.svc.AddMembership(ctx, e.orgB, u.ID, e.compB1), domain.ErrNotFound,
		"a user of org A is invisible from org B")
	assert.ErrorIs(t, e.svc.AddMembership(ctx, e.orgA, u.ID, uuid.New()), domain.ErrNotFound, "unknown company")
}

// SEC-01: ResolvePrincipal denies by default, and every denial looks the
// same so an attacker cannot probe which users or companies exist.
func TestResolvePrincipalDeniesByDefault(t *testing.T) {
	e := setup()
	ctx := context.Background()
	u := e.register(t, e.orgA, "ana")
	require.NoError(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compA1))

	p, err := e.svc.ResolvePrincipal(ctx, e.orgA, u.ID, e.compA1)
	require.NoError(t, err)
	assert.Equal(t, domain.Principal{OrganizationID: e.orgA, UserID: u.ID, CompanyID: e.compA1}, p)

	for name, call := range map[string][3]uuid.UUID{
		"not a member":       {e.orgA, u.ID, e.compA2},
		"foreign company":    {e.orgA, u.ID, e.compB1},
		"wrong organization": {e.orgB, u.ID, e.compA1},
		"unknown user":       {e.orgA, uuid.New(), e.compA1},
		"unknown company":    {e.orgA, u.ID, uuid.New()},
	} {
		_, err := e.svc.ResolvePrincipal(ctx, call[0], call[1], call[2])
		assert.ErrorIs(t, err, domain.ErrForbidden, name)
		assert.NotErrorIs(t, err, domain.ErrNotFound, "%s must not reveal existence", name)
	}
}

func TestDisabledUserLosesAccess(t *testing.T) {
	e := setup()
	ctx := context.Background()
	u := e.register(t, e.orgA, "ana")
	require.NoError(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compA1))
	require.NoError(t, e.svc.DisableUser(ctx, e.orgA, u.ID))

	got, err := e.store.GetUser(ctx, e.orgA, u.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDisabled, got.Status)
	require.NotNil(t, got.DisabledAt)
	assert.Equal(t, e.now, *got.DisabledAt)

	_, err = e.svc.ResolvePrincipal(ctx, e.orgA, u.ID, e.compA1)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.ErrorIs(t, e.svc.AddMembership(ctx, e.orgA, u.ID, e.compA2), domain.ErrForbidden)
	assert.ErrorIs(t, e.svc.DisableUser(ctx, e.orgA, u.ID), domain.ErrInvalidArgument, "already disabled")
	assert.ErrorIs(t, e.svc.DisableUser(ctx, e.orgB, u.ID), domain.ErrNotFound, "cross-organization")
}
