// Package pgstore implements the permission ports on PostgreSQL with pgx
// (BP-009 subtask 3). Every read filters by organization; a row of another
// organization is invisible (application.ErrNotFound or an empty result).
package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// Store is the unit of work over a pool.
type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// InTx runs fn in one READ COMMITTED transaction, committed only if fn
// returns nil.
func (s *Store) InTx(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("permissions pgstore: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(&txAdapter{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func wrap(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", application.ErrNotFound, what)
	}
	return fmt.Errorf("permissions pgstore: %s: %w", what, err)
}

// nullUUID maps uuid.Nil (no branch or warehouse) to SQL NULL.
func nullUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func fromNull(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

func parseQty(s *string) (*qty.Q, error) {
	if s == nil {
		return nil, nil
	}
	q, err := qty.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func qtyText(q *qty.Q) *string {
	if q == nil {
		return nil
	}
	s := q.String()
	return &s
}

type txAdapter struct{ tx pgx.Tx }

func (t *txAdapter) Assignments(ctx context.Context, org, user, company uuid.UUID) ([]domain.Assignment, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT role_id, scope_level, branch_id, warehouse_id FROM permission_assignments
		WHERE organization_id = $1 AND user_id = $2 AND company_id = $3 ORDER BY id`, org, user, company)
	if err != nil {
		return nil, wrap(err, "assignments")
	}
	defer rows.Close()
	var out []domain.Assignment
	for rows.Next() {
		a := domain.Assignment{OrganizationID: org, UserID: user, CompanyID: company}
		var level string
		var branch, wh *uuid.UUID
		if err := rows.Scan(&a.RoleID, &level, &branch, &wh); err != nil {
			return nil, wrap(err, "assignments")
		}
		a.Scope.Level = domain.ScopeLevel(level)
		switch a.Scope.Level {
		case domain.LevelCompany:
			a.Scope.ID = company
		case domain.LevelBranch:
			a.Scope.ID = fromNull(branch)
		case domain.LevelWarehouse:
			a.Scope.ID = fromNull(wh)
		}
		out = append(out, a)
	}
	return out, wrap(rows.Err(), "assignments")
}

func (t *txAdapter) Roles(ctx context.Context, org uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.Role, error) {
	out := map[uuid.UUID]domain.Role{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := t.tx.Query(ctx, `
		SELECT r.id, r.code, g.permission, g.limit_value::text
		FROM permission_roles r LEFT JOIN permission_role_grants g
		  ON g.organization_id = r.organization_id AND g.role_id = r.id
		WHERE r.organization_id = $1 AND r.id = ANY($2)`, org, ids)
	if err != nil {
		return nil, wrap(err, "roles")
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code string
		var perm, limit *string
		if err := rows.Scan(&id, &code, &perm, &limit); err != nil {
			return nil, wrap(err, "roles")
		}
		r, ok := out[id]
		if !ok {
			r = domain.Role{ID: id, OrganizationID: org, Code: code, Grants: map[domain.Permission]*qty.Q{}}
			out[id] = r
		}
		if perm == nil {
			continue
		}
		l, err := parseQty(limit)
		if err != nil {
			return nil, wrap(err, "role limit")
		}
		r.Grants[domain.Permission(*perm)] = l
	}
	return out, wrap(rows.Err(), "roles")
}

func (t *txAdapter) InsertAuthorization(ctx context.Context, a domain.Authorization) error {
	r := a.Request
	_, err := t.tx.Exec(ctx, `
		INSERT INTO supervisor_authorizations (id, organization_id, company_id, requester_user_id,
			supervisor_user_id, permission, target_branch_id, target_warehouse_id, amount,
			entity_ref, terminal_id, session_id, expires_at, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::numeric, $10, $11, $12, $13, $14)`,
		a.ID, r.OrganizationID, r.Target.CompanyID, r.UserID, a.SupervisorID, string(r.Permission),
		nullUUID(r.Target.BranchID), nullUUID(r.Target.WarehouseID), qtyText(r.Amount),
		a.EntityRef, a.TerminalID, a.SessionID, a.ExpiresAt, a.UsedAt)
	return wrap(err, "authorization")
}

func (t *txAdapter) LockAuthorization(ctx context.Context, org, id uuid.UUID) (domain.Authorization, error) {
	a := domain.Authorization{ID: id}
	r := &a.Request
	var perm string
	var branch, wh *uuid.UUID
	var amount *string
	var usedAt *time.Time
	err := t.tx.QueryRow(ctx, `
		SELECT organization_id, company_id, requester_user_id, supervisor_user_id, permission,
			target_branch_id, target_warehouse_id, amount::text, entity_ref, terminal_id, session_id,
			expires_at, used_at
		FROM supervisor_authorizations WHERE organization_id = $1 AND id = $2 FOR UPDATE`, org, id).
		Scan(&r.OrganizationID, &r.Target.CompanyID, &r.UserID, &a.SupervisorID, &perm,
			&branch, &wh, &amount, &a.EntityRef, &a.TerminalID, &a.SessionID, &a.ExpiresAt, &usedAt)
	if err != nil {
		return domain.Authorization{}, wrap(err, "authorization")
	}
	r.Permission = domain.Permission(perm)
	r.Target.BranchID, r.Target.WarehouseID = fromNull(branch), fromNull(wh)
	if r.Amount, err = parseQty(amount); err != nil {
		return domain.Authorization{}, wrap(err, "authorization amount")
	}
	a.ExpiresAt = a.ExpiresAt.UTC()
	if usedAt != nil {
		u := usedAt.UTC()
		a.UsedAt = &u
	}
	return a, nil
}

// MarkAuthorizationUsed sets used_at only if it was unset, so a second use
// fails even without the row lock.
func (t *txAdapter) MarkAuthorizationUsed(ctx context.Context, a domain.Authorization) error {
	if a.UsedAt == nil {
		return fmt.Errorf("%w: used_at is required", domain.ErrInvalidArgument)
	}
	tag, err := t.tx.Exec(ctx, `
		UPDATE supervisor_authorizations SET used_at = $3
		WHERE organization_id = $1 AND id = $2 AND used_at IS NULL`,
		a.Request.OrganizationID, a.ID, *a.UsedAt)
	if err != nil {
		return wrap(err, "authorization")
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrAuthorizationUsed
	}
	return nil
}

func (t *txAdapter) AppendAudit(ctx context.Context, ev application.AuditEvent) error {
	after, err := json.Marshal(ev.After)
	if err != nil {
		return wrap(err, "audit payload")
	}
	var reason *string
	if ev.Reason != "" {
		reason = &ev.Reason
	}
	_, err = t.tx.Exec(ctx, `
		INSERT INTO audit_events (organization_id, company_id, actor_user_id, action, entity_type,
			entity_id, after, reason, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		ev.OrganizationID, nullUUID(ev.CompanyID), nullUUID(ev.ActorUserID), ev.Action, ev.EntityType,
		ev.EntityID, after, reason, ev.CorrelationID)
	return wrap(err, "audit")
}
