package pgstore_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
)

// BP-008 subtask 3 (#80): the identity ports on real PostgreSQL. Synthetic
// data only.

type env struct {
	ctx            context.Context
	store          *pgstore.Store
	svc            *application.Service
	orgA, orgB     uuid.UUID
	compA1, compA2 uuid.UUID
	compB1         uuid.UUID
}

func setup(t *testing.T) env {
	t.Helper()
	pool := pgtest.New(t)
	ctx := context.Background()
	org := organization.NewStore(pool)
	e := env{ctx: ctx, store: pgstore.New(pool)}
	var err error
	e.orgA, err = org.CreateOrganization(ctx, "Org A")
	require.NoError(t, err)
	e.orgB, err = org.CreateOrganization(ctx, "Org B")
	require.NoError(t, err)
	e.compA1, err = org.CreateCompany(ctx, e.orgA, "Empresa A1")
	require.NoError(t, err)
	e.compA2, err = org.CreateCompany(ctx, e.orgA, "Empresa A2")
	require.NoError(t, err)
	e.compB1, err = org.CreateCompany(ctx, e.orgB, "Empresa B1")
	require.NoError(t, err)
	e.svc = application.NewService(e.store, func() time.Time { return time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC) })
	return e
}

func (e env) register(t *testing.T, org uuid.UUID, login string) domain.User {
	t.Helper()
	u, err := e.svc.RegisterUser(e.ctx, org, application.RegisterUserCommand{ID: uuid.New(), Login: login, DisplayName: "Usuario sintético"})
	require.NoError(t, err)
	return u
}

func TestUserRoundTripAndDuplicates(t *testing.T) {
	e := setup(t)
	u := e.register(t, e.orgA, " Ana ")
	var got domain.User
	require.NoError(t, e.store.InTx(e.ctx, func(tx application.Tx) error {
		var err error
		got, err = tx.GetUser(e.ctx, e.orgA, u.ID)
		return err
	}))
	assert.Equal(t, u, got)

	_, err := e.svc.RegisterUser(e.ctx, e.orgA, application.RegisterUserCommand{ID: uuid.New(), Login: "ANA", DisplayName: "x"})
	assert.ErrorIs(t, err, domain.ErrDuplicate)
	e.register(t, e.orgB, "ana")

	err = e.store.InTx(e.ctx, func(tx application.Tx) error { _, err := tx.GetUser(e.ctx, e.orgB, u.ID); return err })
	assert.ErrorIs(t, err, domain.ErrNotFound, "a user is invisible from another organization")
}

// SEC-01 on PostgreSQL.
func TestMembershipsAndPrincipalOnPostgres(t *testing.T) {
	e := setup(t)
	u := e.register(t, e.orgA, "ana")
	require.NoError(t, e.svc.AddMembership(e.ctx, e.orgA, u.ID, e.compA1))
	assert.ErrorIs(t, e.svc.AddMembership(e.ctx, e.orgA, u.ID, e.compA1), domain.ErrDuplicate)
	assert.ErrorIs(t, e.svc.AddMembership(e.ctx, e.orgA, u.ID, e.compB1), domain.ErrForbidden)
	assert.ErrorIs(t, e.svc.AddMembership(e.ctx, e.orgA, u.ID, uuid.New()), domain.ErrNotFound)

	p, err := e.svc.ResolvePrincipal(e.ctx, e.orgA, u.ID, e.compA1)
	require.NoError(t, err)
	assert.Equal(t, domain.Principal{OrganizationID: e.orgA, UserID: u.ID, CompanyID: e.compA1}, p)
	for name, c := range map[string][3]uuid.UUID{
		"not a member":       {e.orgA, u.ID, e.compA2},
		"foreign company":    {e.orgA, u.ID, e.compB1},
		"wrong organization": {e.orgB, u.ID, e.compA1},
		"unknown user":       {e.orgA, uuid.New(), e.compA1},
	} {
		_, err := e.svc.ResolvePrincipal(e.ctx, c[0], c[1], c[2])
		assert.ErrorIs(t, err, domain.ErrForbidden, name)
	}

	require.NoError(t, e.svc.DisableUser(e.ctx, e.orgA, u.ID))
	_, err = e.svc.ResolvePrincipal(e.ctx, e.orgA, u.ID, e.compA1)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.ErrorIs(t, e.svc.DisableUser(e.ctx, e.orgB, u.ID), domain.ErrNotFound)
}

func TestConcurrentRegistrationOfSameLoginHasOneWinner(t *testing.T) {
	e := setup(t)
	var ok, dup atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.svc.RegisterUser(e.ctx, e.orgA, application.RegisterUserCommand{ID: uuid.New(), Login: "caja1", DisplayName: "Caja"})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, domain.ErrDuplicate):
				dup.Add(1)
			default:
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), ok.Load())
	assert.Equal(t, int32(9), dup.Load())
}

func TestConcurrentSameMembershipHasOneWinner(t *testing.T) {
	e := setup(t)
	u := e.register(t, e.orgA, "ana")
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.svc.AddMembership(e.ctx, e.orgA, u.ID, e.compA1); err == nil {
				ok.Add(1)
			} else if !errors.Is(err, domain.ErrDuplicate) {
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), ok.Load())
}

// The race the user lock prevents: another transaction is disabling the
// user and has not committed yet. AddMembership must wait for it and then
// be refused, never insert a membership for a user being disabled.
func TestAddMembershipWaitsForConcurrentDisable(t *testing.T) {
	e := setup(t)
	u := e.register(t, e.orgA, "ana")
	disabling, release, added := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_ = e.store.InTx(e.ctx, func(tx application.Tx) error {
			cur, err := tx.LockUser(e.ctx, e.orgA, u.ID)
			if err != nil {
				return err
			}
			d, err := cur.Disable(time.Now())
			if err != nil {
				return err
			}
			if err := tx.UpdateUser(e.ctx, d); err != nil {
				return err
			}
			close(disabling)
			<-release
			return nil
		})
	}()
	<-disabling
	go func() { added <- e.svc.AddMembership(e.ctx, e.orgA, u.ID, e.compA1) }()
	select {
	case err := <-added:
		t.Fatalf("AddMembership finished while the user was being disabled: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-added:
		assert.ErrorIs(t, err, domain.ErrForbidden, "the committed disable wins")
	case <-time.After(5 * time.Second):
		t.Fatal("AddMembership never finished")
	}
	var n int
	require.NoError(t, e.store.InTx(e.ctx, func(tx application.Tx) error {
		ms, err := tx.ListMemberships(e.ctx, e.orgA, u.ID)
		n = len(ms)
		return err
	}))
	assert.Zero(t, n)
}
