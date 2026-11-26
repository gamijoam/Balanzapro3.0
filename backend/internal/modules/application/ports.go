// Package application orchestrates module activation (ADR-023) over storage,
// license and authorization ports. Changing activation needs the license,
// active dependencies and the actor's permission: three separate checks.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/domain"
	permapp "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/application"
	permdomain "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
)

// ErrNotFound is what storage returns for a company outside the organization.
var ErrNotFound = errors.New("modules: not found")

// AuditEvent is one row of the append-only audit trail (migration 00006).
type AuditEvent struct {
	OrganizationID uuid.UUID
	CompanyID      uuid.UUID
	ActorUserID    uuid.UUID
	Action         string
	EntityType     string
	EntityID       string
	Before         map[string]any
	After          map[string]any
	Reason         string
	CorrelationID  uuid.UUID
}

// Store is the unit of work.
type Store interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

// Tx is what the use cases need from storage.
type Tx interface {
	// LockModules returns the company's activation state and holds it until
	// the transaction ends, so concurrent changes of one company serialize.
	LockModules(ctx context.Context, org, company uuid.UUID) (domain.State, error)
	SaveModule(ctx context.Context, company uuid.UUID, module string, active bool) error
	AppendAudit(ctx context.Context, ev AuditEvent) error
}

// Licenses gives what a company's license includes (ADR-147). Verifying
// signed licenses is BP-024.
type Licenses interface {
	Entitlements(ctx context.Context, org, company uuid.UUID) (domain.Entitlements, error)
}

// ChangeCommand activates or deactivates one module in one company.
type ChangeCommand struct {
	OrganizationID uuid.UUID
	ActorUserID    uuid.UUID
	CompanyID      uuid.UUID
	ModuleID       string
	Reason         string
	CorrelationID  uuid.UUID
}

// Service implements the activation use cases.
type Service struct {
	catalog  domain.Catalog
	store    Store
	authz    permapp.Authorizer
	licenses Licenses
}

func NewService(c domain.Catalog, s Store, a permapp.Authorizer, l Licenses) *Service {
	return &Service{catalog: c, store: s, authz: a, licenses: l}
}

// Activate turns a licensed module on for a company.
func (s *Service) Activate(ctx context.Context, c ChangeCommand) error {
	return s.change(ctx, c, "modules.activate", func(st domain.State) (domain.State, error) {
		e, err := s.licenses.Entitlements(ctx, c.OrganizationID, c.CompanyID)
		if err != nil {
			return nil, err
		}
		return s.catalog.Activate(st, c.ModuleID, e)
	})
}

// Deactivate turns a module off for a company; its data stays.
func (s *Service) Deactivate(ctx context.Context, c ChangeCommand) error {
	return s.change(ctx, c, "modules.deactivate", func(st domain.State) (domain.State, error) {
		return s.catalog.Deactivate(st, c.ModuleID)
	})
}

func (s *Service) change(ctx context.Context, c ChangeCommand, action string,
	apply func(domain.State) (domain.State, error)) error {
	if c.OrganizationID == uuid.Nil || c.ActorUserID == uuid.Nil || c.CompanyID == uuid.Nil ||
		c.CorrelationID == uuid.Nil || strings.TrimSpace(c.Reason) == "" {
		return fmt.Errorf("%w: organization, actor, company, reason and correlation are required", domain.ErrInvalidArgument)
	}
	if err := s.authz.Authorize(ctx, permdomain.Request{OrganizationID: c.OrganizationID, UserID: c.ActorUserID,
		Permission: domain.PermissionManage, Target: permdomain.ScopePath{CompanyID: c.CompanyID}}); err != nil {
		return err
	}
	return s.store.InTx(ctx, func(tx Tx) error {
		st, err := tx.LockModules(ctx, c.OrganizationID, c.CompanyID)
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: company", permdomain.ErrForbidden)
		}
		if err != nil {
			return err
		}
		next, err := apply(st)
		if err != nil {
			return err
		}
		was, now := st.IsActive(c.ModuleID), next.IsActive(c.ModuleID)
		if was == now {
			return nil
		}
		if err := tx.SaveModule(ctx, c.CompanyID, c.ModuleID, now); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, AuditEvent{OrganizationID: c.OrganizationID, CompanyID: c.CompanyID,
			ActorUserID: c.ActorUserID, Action: action, EntityType: "company_module", EntityID: c.ModuleID,
			Before: map[string]any{"active": was}, After: map[string]any{"active": now},
			Reason: c.Reason, CorrelationID: c.CorrelationID})
	})
}
