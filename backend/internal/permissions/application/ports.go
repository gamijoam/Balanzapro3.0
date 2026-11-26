// Package application implements the Authorizer port (ADR-010) and the
// supervisor authorization flow (ADR-025) over storage ports. Verifying the
// supervisor's PIN belongs to identity (ADR-026) and happens before
// ApproveBySupervisor is called.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
)

// ErrNotFound is what storage returns for a row outside the organization.
var ErrNotFound = errors.New("permissions: not found")

// AuditEvent is one row of the append-only audit trail (migration 00006).
// It never carries secrets such as PINs.
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

// Tx is what the use cases need from storage; every read is filtered by
// organization.
type Tx interface {
	// Assignments returns grants only for an active actor and a verified target.
	// Organization inheritance into a company requires current membership.
	Assignments(ctx context.Context, org, user uuid.UUID, target domain.ScopePath) ([]domain.Assignment, error)
	Roles(ctx context.Context, org uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.Role, error)
	InsertAuthorization(ctx context.Context, a domain.Authorization) error
	// LockAuthorization holds the row so it is redeemed once (SEC-02).
	LockAuthorization(ctx context.Context, org, id uuid.UUID) (domain.Authorization, error)
	MarkAuthorizationUsed(ctx context.Context, a domain.Authorization) error
	AppendAudit(ctx context.Context, ev AuditEvent) error
}

// Authorizer is the port every use case calls before acting (ADR-010).
type Authorizer interface {
	Authorize(ctx context.Context, r domain.Request) error
}

// ApproveCommand is a supervisor approving one concrete action.
type ApproveCommand struct {
	ID            uuid.UUID
	SupervisorID  uuid.UUID
	Request       domain.Request
	EntityRef     string
	TerminalID    uuid.UUID
	SessionID     uuid.UUID
	TTL           time.Duration
	CorrelationID uuid.UUID
}

// Service implements Authorizer and the supervisor flow.
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

func decide(ctx context.Context, tx Tx, r domain.Request) error {
	as, err := tx.Assignments(ctx, r.OrganizationID, r.UserID, r.Target)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(as))
	for _, a := range as {
		ids = append(ids, a.RoleID)
	}
	roles, err := tx.Roles(ctx, r.OrganizationID, ids)
	if err != nil {
		return err
	}
	return domain.Decide(r, as, roles)
}

// Authorize denies by default (ADR-024).
func (s *Service) Authorize(ctx context.Context, r domain.Request) error {
	return s.store.InTx(ctx, func(tx Tx) error { return decide(ctx, tx, r) })
}

// ApproveBySupervisor records an authorization for the requested action
// only if the supervisor may perform it themselves, within their limit.
func (s *Service) ApproveBySupervisor(ctx context.Context, c ApproveCommand) (domain.Authorization, error) {
	if c.CorrelationID == uuid.Nil {
		return domain.Authorization{}, fmt.Errorf("%w: correlation id is required", domain.ErrInvalidArgument)
	}
	a, err := domain.NewAuthorization(c.ID, c.SupervisorID, c.Request, c.EntityRef, c.TerminalID, c.SessionID, s.now(), c.TTL)
	if err != nil {
		return domain.Authorization{}, err
	}
	err = s.store.InTx(ctx, func(tx Tx) error {
		asSupervisor := c.Request
		asSupervisor.UserID = c.SupervisorID
		if err := decide(ctx, tx, asSupervisor); err != nil {
			return err
		}
		if err := tx.InsertAuthorization(ctx, a); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, auditOf(a, "permissions.authorization.grant", c.SupervisorID, c.CorrelationID))
	})
	if err != nil {
		return domain.Authorization{}, err
	}
	return a, nil
}

// Redeem uses an authorization once for exactly the approved action.
// Unknown or foreign authorizations are reported as ErrForbidden.
func (s *Service) Redeem(ctx context.Context, org, id uuid.UUID, r domain.Redemption, correlation uuid.UUID) error {
	return s.store.InTx(ctx, func(tx Tx) error {
		a, err := tx.LockAuthorization(ctx, org, id)
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: authorization", domain.ErrForbidden)
		}
		if err != nil {
			return err
		}
		used, err := a.Redeem(r, s.now())
		if err != nil {
			return err
		}
		if err := tx.MarkAuthorizationUsed(ctx, used); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, auditOf(used, "permissions.authorization.redeem", r.Request.UserID, correlation))
	})
}

func auditOf(a domain.Authorization, action string, actor, correlation uuid.UUID) AuditEvent {
	after := map[string]any{
		"permission": string(a.Request.Permission), "requester": a.Request.UserID.String(),
		"supervisor": a.SupervisorID.String(), "entity": a.EntityRef,
		"terminal": a.TerminalID.String(), "expires_at": a.ExpiresAt.Format(time.RFC3339),
	}
	if a.Request.Amount != nil {
		after["amount"] = a.Request.Amount.String()
	}
	return AuditEvent{OrganizationID: a.Request.OrganizationID, CompanyID: a.Request.Target.CompanyID,
		ActorUserID: actor, Action: action, EntityType: "supervisor_authorization", EntityID: a.ID.String(),
		After: after, CorrelationID: correlation}
}
