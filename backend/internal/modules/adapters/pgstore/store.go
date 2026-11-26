// Package pgstore persists module activation and audit events (ADR-023).
package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/domain"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

var _ application.Store = (*Store)(nil)
var errStorage = errors.New("modules: storage failed")

func (s *Store) InTx(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return errStorage
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	a := &adapter{tx: tx}
	if err := fn(a); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errStorage
	}
	return nil
}

type adapter struct {
	tx           pgx.Tx
	org, company uuid.UUID
}

var _ application.Tx = (*adapter)(nil)

// Lock the company even when no activation rows exist. Read committed
// ensures a waiter sees the predecessor's committed state after locking.
func (a *adapter) LockModules(ctx context.Context, org, company uuid.UUID) (domain.State, error) {
	if org == uuid.Nil || company == uuid.Nil {
		return nil, application.ErrNotFound
	}
	if a.company != uuid.Nil && (a.company != company || a.org != org) {
		return nil, domain.ErrInvalidArgument
	}
	var id uuid.UUID
	err := a.tx.QueryRow(ctx, "SELECT id FROM companies WHERE organization_id=$1 AND id=$2 FOR UPDATE", org, company).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, errStorage
	}
	rows, err := a.tx.Query(ctx, "SELECT module_id,status FROM company_modules WHERE company_id=$1", company)
	if err != nil {
		return nil, errStorage
	}
	defer rows.Close()
	state := domain.State{}
	for rows.Next() {
		var module, status string
		if err := rows.Scan(&module, &status); err != nil {
			return nil, errStorage
		}
		state[module] = status == "active"
	}
	if rows.Err() != nil {
		return nil, errStorage
	}
	a.org, a.company = org, company
	return state, nil
}
func (a *adapter) SaveModule(ctx context.Context, company uuid.UUID, module string, active bool) error {
	if a.company == uuid.Nil || company != a.company {
		return application.ErrNotFound
	}
	for _, m := range domain.BaseManifests() {
		if m.ID == module {
			return domain.ErrBaseModule
		}
	}
	status := "inactive"
	if active {
		status = "active"
	}
	_, err := a.tx.Exec(ctx, `INSERT INTO company_modules(company_id,module_id,status) VALUES($1,$2,$3)
 ON CONFLICT(company_id,module_id) DO UPDATE SET status=EXCLUDED.status,version=company_modules.version+1,updated_at=now()
 WHERE company_modules.status<>EXCLUDED.status`, company, module, status)
	if err != nil {
		return errStorage
	}
	return nil
}
func (a *adapter) AppendAudit(ctx context.Context, ev application.AuditEvent) error {
	if a.company == uuid.Nil || ev.CompanyID != a.company || ev.OrganizationID != a.org {
		return application.ErrNotFound
	}
	var actor uuid.UUID
	err := a.tx.QueryRow(ctx, `SELECT u.id FROM identity_users u JOIN identity_memberships m
 ON m.organization_id=u.organization_id AND m.user_id=u.id
 WHERE u.organization_id=$1 AND u.id=$2 AND m.company_id=$3 AND u.status='active'`, a.org, ev.ActorUserID, a.company).Scan(&actor)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	if err != nil {
		return errStorage
	}
	before, err := json.Marshal(ev.Before)
	if err != nil {
		return fmt.Errorf("%w: audit encoding", errStorage)
	}
	after, err := json.Marshal(ev.After)
	if err != nil {
		return fmt.Errorf("%w: audit encoding", errStorage)
	}
	_, err = a.tx.Exec(ctx, `INSERT INTO audit_events(organization_id,company_id,actor_user_id,action,entity_type,entity_id,before,after,reason,correlation_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, a.org, a.company, actor, ev.Action, ev.EntityType, ev.EntityID, before, after, ev.Reason, ev.CorrelationID)
	if err != nil {
		return errStorage
	}
	return nil
}
