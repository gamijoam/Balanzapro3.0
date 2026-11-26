// Package application orchestrates typed configuration (ADR-027) over
// storage and the Authorizer port (ADR-010).
//
// Organization writes require explicit organization authorization (ADR-174).
// Global writes remain denied. Reading needs no permission here; the calling
// use case is already authorized.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
	permapp "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/application"
	permdomain "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// ErrNotFound is what storage returns for a scope outside the company.
var ErrNotFound = errors.New("configuration: not found")

// ErrVersionConflict is what storage returns when a concurrent change took
// the same version; the caller may retry.
var ErrVersionConflict = errors.New("configuration: version conflict")

// AuditEvent is one row of the append-only audit trail (migration 00006).
type AuditEvent struct {
	OrganizationID uuid.UUID
	CompanyID      uuid.UUID
	ActorUserID    uuid.UUID
	Action         string
	EntityType     string
	EntityID       string
	After          map[string]any
	Reason         string
	CorrelationID  uuid.UUID
}

// Store is the unit of work.
type Store interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

// Tx is what the use cases need from storage; reads filter by organization.
type Tx interface {
	// VerifyScope checks the organization scope when company is nil; otherwise
	// the requested scope must belong to the company and organization.
	// Return ErrNotFound for a missing or foreign scope.
	VerifyScope(ctx context.Context, org, company uuid.UUID, sc domain.Scope) error
	LatestVersion(ctx context.Context, org uuid.UUID, key string, sc domain.Scope) (int, error)
	InsertEntry(ctx context.Context, org uuid.UUID, e domain.Entry) error // ErrVersionConflict
	Entries(ctx context.Context, org uuid.UUID, key string, scopes []domain.Scope) ([]domain.Entry, error)
	AppendAudit(ctx context.Context, ev AuditEvent) error
}

// SetCommand changes a key at one scope. Organization writes have no
// company or branch; BranchID identifies branch or terminal scope.
type SetCommand struct {
	OrganizationID uuid.UUID
	CompanyID      uuid.UUID
	BranchID       uuid.UUID
	ActorUserID    uuid.UUID
	Key            string
	Scope          domain.Scope
	Value          any
	Reason         string
	CorrelationID  uuid.UUID
}

// Service implements the configuration use cases.
type Service struct {
	reg   domain.Registry
	store Store
	authz permapp.Authorizer
	now   func() time.Time
}

// NewService builds the service; now defaults to time.Now.
func NewService(r domain.Registry, s Store, a permapp.Authorizer, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{reg: r, store: s, authz: a, now: now}
}

// target is the permission scope a write at sc needs.
func (c SetCommand) target() (permdomain.ScopePath, error) {
	switch c.Scope.Level {
	case domain.LevelOrganization:
		if c.Scope.ID != c.OrganizationID || c.CompanyID != uuid.Nil || c.BranchID != uuid.Nil {
			return permdomain.ScopePath{}, fmt.Errorf("%w: organization scope mismatch", permdomain.ErrForbidden)
		}
		return permdomain.ScopePath{OrganizationID: c.OrganizationID}, nil
	case domain.LevelCompany, domain.LevelUser:
		return permdomain.ScopePath{CompanyID: c.CompanyID}, nil
	case domain.LevelBranch, domain.LevelTerminal:
		if c.BranchID == uuid.Nil || (c.Scope.Level == domain.LevelBranch && c.Scope.ID != c.BranchID) {
			return permdomain.ScopePath{}, fmt.Errorf("%w: branch does not match scope", domain.ErrInvalidArgument)
		}
		return permdomain.ScopePath{CompanyID: c.CompanyID, BranchID: c.BranchID}, nil
	}
	return permdomain.ScopePath{}, fmt.Errorf("%w: %s writes need an administrative scope", permdomain.ErrForbidden, c.Scope.Level)
}

func display(v any) string {
	if d, ok := v.(qty.Q); ok {
		return d.String()
	}
	return fmt.Sprint(v)
}

// Set validates, authorizes and stores the next version of a value.
func (s *Service) Set(ctx context.Context, c SetCommand) error {
	if c.OrganizationID == uuid.Nil || c.ActorUserID == uuid.Nil || c.CorrelationID == uuid.Nil {
		return fmt.Errorf("%w: organization, actor and correlation are required", domain.ErrInvalidArgument)
	}
	if c.Scope.Level != domain.LevelOrganization && c.Scope.Level != domain.LevelGlobal && c.CompanyID == uuid.Nil {
		return fmt.Errorf("%w: company is required", domain.ErrInvalidArgument)
	}
	if _, err := s.reg.Validate(c.Key, c.Value); err != nil {
		return err
	}
	if c.Scope.Level == domain.LevelCompany && c.Scope.ID != c.CompanyID {
		return fmt.Errorf("%w: company does not match scope", domain.ErrInvalidArgument)
	}
	target, err := c.target()
	if err != nil {
		return err
	}
	if err := s.authz.Authorize(ctx, permdomain.Request{OrganizationID: c.OrganizationID, UserID: c.ActorUserID,
		Permission: domain.PermissionManage, Target: target}); err != nil {
		return err
	}
	return s.store.InTx(ctx, func(tx Tx) error {
		err := tx.VerifyScope(ctx, c.OrganizationID, c.CompanyID, c.Scope)
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: scope", permdomain.ErrForbidden)
		}
		if err != nil {
			return err
		}
		current, err := tx.LatestVersion(ctx, c.OrganizationID, c.Key, c.Scope)
		if err != nil {
			return err
		}
		e, err := s.reg.NewChange(c.Key, c.Scope, c.Value, c.ActorUserID, c.Reason, current, s.now())
		if err != nil {
			return err
		}
		if err := tx.InsertEntry(ctx, c.OrganizationID, e); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, AuditEvent{OrganizationID: c.OrganizationID, CompanyID: c.CompanyID,
			ActorUserID: c.ActorUserID, Action: "configuration.set", EntityType: "config_entry", EntityID: c.Key,
			After: map[string]any{"level": string(e.Scope.Level), "scope_id": e.Scope.ID.String(),
				"version": e.Version, "value": display(e.Value)},
			Reason: e.Reason, CorrelationID: c.CorrelationID})
	})
}

// Resolve returns the effective value of key for the chain.
func (s *Service) Resolve(ctx context.Context, key string, c domain.Chain) (domain.Resolved, error) {
	var out domain.Resolved
	err := s.store.InTx(ctx, func(tx Tx) error {
		entries, err := tx.Entries(ctx, c.OrganizationID, key, c.Scopes())
		if err != nil {
			return err
		}
		out, err = s.reg.Resolve(key, c, entries)
		return err
	})
	return out, err
}
