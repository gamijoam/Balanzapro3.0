// Package application orchestrates identity use cases (BP-008) over the
// storage ports a PostgreSQL adapter implements (subtask #80). Every call
// carries the organization it acts for; storage filters by it.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
)

// Store is the unit of work: fn runs in one transaction.
type Store interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

// Tx is what the use cases need from storage. A user of another
// organization is reported as domain.ErrNotFound.
type Tx interface {
	InsertUser(ctx context.Context, u domain.User) error // ErrDuplicate on login
	GetUser(ctx context.Context, org, id uuid.UUID) (domain.User, error)
	UpdateUser(ctx context.Context, u domain.User) error
	CompanyOrganization(ctx context.Context, company uuid.UUID) (uuid.UUID, error)
	InsertMembership(ctx context.Context, m domain.Membership) error // ErrDuplicate
	ListMemberships(ctx context.Context, org, user uuid.UUID) ([]domain.Membership, error)
}

// RegisterUserCommand creates an active user with a client-provided ID.
type RegisterUserCommand struct {
	ID          uuid.UUID
	Login       string
	DisplayName string
}

// Service implements the identity use cases.
type Service struct {
	store Store
	now   func() time.Time
}

// NewService builds the service; now defaults to time.Now.
func NewService(store Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

func requireOrg(org uuid.UUID) error {
	if org == uuid.Nil {
		return fmt.Errorf("%w: organization is required", domain.ErrInvalidArgument)
	}
	return nil
}

// RegisterUser creates an active user in the organization.
func (s *Service) RegisterUser(ctx context.Context, org uuid.UUID, c RegisterUserCommand) (domain.User, error) {
	if err := requireOrg(org); err != nil {
		return domain.User{}, err
	}
	u, err := domain.NewUser(c.ID, org, c.Login, c.DisplayName)
	if err != nil {
		return domain.User{}, err
	}
	return u, s.store.InTx(ctx, func(tx Tx) error { return tx.InsertUser(ctx, u) })
}

// AddMembership lets an active user act in a company of its organization.
func (s *Service) AddMembership(ctx context.Context, org, user, company uuid.UUID) error {
	if err := requireOrg(org); err != nil {
		return err
	}
	return s.store.InTx(ctx, func(tx Tx) error {
		u, err := tx.GetUser(ctx, org, user)
		if err != nil {
			return err
		}
		companyOrg, err := tx.CompanyOrganization(ctx, company)
		if err != nil {
			return err
		}
		m, err := domain.NewMembership(u, companyOrg, company)
		if err != nil {
			return err
		}
		return tx.InsertMembership(ctx, m)
	})
}

// DisableUser disables a user; it keeps its history and memberships but
// can no longer act (ResolvePrincipal denies it).
func (s *Service) DisableUser(ctx context.Context, org, user uuid.UUID) error {
	if err := requireOrg(org); err != nil {
		return err
	}
	return s.store.InTx(ctx, func(tx Tx) error {
		u, err := tx.GetUser(ctx, org, user)
		if err != nil {
			return err
		}
		if u, err = u.Disable(s.now()); err != nil {
			return err
		}
		return tx.UpdateUser(ctx, u)
	})
}

// ResolvePrincipal returns who acts in which company, denying by default
// (SEC-01). Unknown users or companies, other organizations, missing
// memberships and disabled users all yield the same ErrForbidden, so a
// caller cannot probe what exists. Infrastructure errors pass through.
func (s *Service) ResolvePrincipal(ctx context.Context, org, user, company uuid.UUID) (domain.Principal, error) {
	var p domain.Principal
	err := s.store.InTx(ctx, func(tx Tx) error {
		u, err := tx.GetUser(ctx, org, user)
		if err != nil {
			return err
		}
		ms, err := tx.ListMemberships(ctx, org, user)
		if err != nil {
			return err
		}
		p, err = domain.Authorize(u, ms, company)
		return err
	})
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidArgument) || errors.Is(err, domain.ErrForbidden) {
		return domain.Principal{}, fmt.Errorf("%w: access denied", domain.ErrForbidden)
	}
	return p, err
}
