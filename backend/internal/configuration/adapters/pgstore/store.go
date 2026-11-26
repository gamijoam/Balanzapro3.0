// Package pgstore persists administrative configuration and its audit trail.
// Terminal and user preferences remain denied until their scope is defined.
package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
)

type Store struct {
	pool *pgxpool.Pool
	reg  domain.Registry
}

func New(pool *pgxpool.Pool, reg domain.Registry) *Store { return &Store{pool: pool, reg: reg} }

var _ application.Store = (*Store)(nil)

var errStorage = errors.New("configuration: storage failed")

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		if pe.Code == "23505" || pe.Code == "40001" {
			return application.ErrVersionConflict
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return errStorage
}

func (s *Store) InTx(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = fn(&adapter{tx: tx, reg: s.reg}); err != nil {
		return err
	}
	return mapErr(tx.Commit(ctx))
}

type adapter struct {
	tx  pgx.Tx
	reg domain.Registry
}

var _ application.Tx = (*adapter)(nil)

func nullable(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// owner verifies each administrative scope against the organization and
// returns its actual company (nil at organization level).
func (a *adapter) owner(ctx context.Context, org uuid.UUID, sc domain.Scope) (uuid.UUID, error) {
	if org == uuid.Nil || sc.ID == uuid.Nil {
		return uuid.Nil, application.ErrNotFound
	}
	var company uuid.UUID
	var err error
	switch sc.Level {
	case domain.LevelOrganization:
		if sc.ID != org {
			return uuid.Nil, application.ErrNotFound
		}
		var ok bool
		err = a.tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)", org).Scan(&ok)
		if err != nil {
			return uuid.Nil, mapErr(err)
		}
		if !ok {
			return uuid.Nil, application.ErrNotFound
		}
		return uuid.Nil, nil
	case domain.LevelCompany:
		err = a.tx.QueryRow(ctx, "SELECT id FROM companies WHERE organization_id=$1 AND id=$2", org, sc.ID).Scan(&company)
	case domain.LevelBranch:
		err = a.tx.QueryRow(ctx, `SELECT b.company_id FROM branches b JOIN companies c ON c.id=b.company_id
    WHERE c.organization_id=$1 AND b.id=$2`, org, sc.ID).Scan(&company)
	default:
		return uuid.Nil, application.ErrNotFound
	}
	return company, mapErr(err)
}

func (a *adapter) VerifyScope(ctx context.Context, org, company uuid.UUID, sc domain.Scope) error {
	actual, err := a.owner(ctx, org, sc)
	if err != nil {
		return err
	}
	if actual != company {
		return application.ErrNotFound
	}
	return nil
}
func (a *adapter) LatestVersion(ctx context.Context, org uuid.UUID, key string, sc domain.Scope) (int, error) {
	if _, err := a.owner(ctx, org, sc); err != nil {
		return 0, err
	}
	var v int
	err := a.tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) FROM config_entries
 WHERE organization_id=$1 AND key=$2 AND scope_level=$3 AND scope_id=$4`, org, key, string(sc.Level), sc.ID).Scan(&v)
	return v, mapErr(err)
}
func (a *adapter) InsertEntry(ctx context.Context, org uuid.UUID, e domain.Entry) error {
	if _, err := a.owner(ctx, org, e.Scope); err != nil {
		return err
	}
	if e.Version < 1 {
		return domain.ErrInvalidArgument
	}
	var active bool
	if err := a.tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM identity_users WHERE organization_id=$1 AND id=$2 AND status='active')", org, e.ActorUserID).Scan(&active); err != nil {
		return mapErr(err)
	}
	if !active {
		return application.ErrNotFound
	}
	// Validate type, allowed level and metadata at the storage boundary too.
	checked, err := a.reg.NewChange(e.Key, e.Scope, e.Value, e.ActorUserID, e.Reason, e.Version-1, e.CreatedAt)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(checked.Value)
	if err != nil {
		return domain.ErrInvalidValue
	}
	_, err = a.tx.Exec(ctx, `INSERT INTO config_entries(organization_id,key,scope_level,scope_id,version,value,actor_user_id,reason,created_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, org, checked.Key, string(checked.Scope.Level), checked.Scope.ID, checked.Version, raw, checked.ActorUserID, checked.Reason, checked.CreatedAt)
	return mapErr(err)
}
func (a *adapter) Entries(ctx context.Context, org uuid.UUID, key string, scopes []domain.Scope) ([]domain.Entry, error) {
	// Require a real organization and reject inconsistent chains before
	// fetching any value, including a global default.
	if _, err := a.owner(ctx, org, domain.Scope{Level: domain.LevelOrganization, ID: org}); err != nil {
		return nil, err
	}
	var company uuid.UUID
	for _, sc := range scopes {
		if sc.Level == domain.LevelCompany {
			company = sc.ID
		}
	}
	for _, sc := range scopes {
		if sc.Level == domain.LevelGlobal {
			if sc.ID != uuid.Nil {
				return nil, application.ErrNotFound
			}
			continue
		}
		actual, err := a.owner(ctx, org, sc)
		if err != nil {
			return nil, err
		}
		if sc.Level != domain.LevelOrganization && (company == uuid.Nil || company != actual) {
			return nil, application.ErrNotFound
		}
	}
	var out []domain.Entry
	for _, sc := range scopes {
		e := domain.Entry{Key: key, Scope: sc}
		var raw []byte
		var actor *uuid.UUID
		scopeOrg := nullable(org)
		if sc.Level == domain.LevelGlobal {
			scopeOrg = nil
		}
		err := a.tx.QueryRow(ctx, `SELECT version,value,actor_user_id,reason,created_at FROM config_entries
   WHERE organization_id IS NOT DISTINCT FROM $1::uuid AND key=$2 AND scope_level=$3
     AND scope_id IS NOT DISTINCT FROM $4::uuid ORDER BY version DESC LIMIT 1`, scopeOrg, key, string(sc.Level), nullable(sc.ID)).
			Scan(&e.Version, &raw, &actor, &e.Reason, &e.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, mapErr(err)
		}
		if actor != nil {
			e.ActorUserID = *actor
		}
		e.Value, err = a.reg.DecodeValue(key, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
func (a *adapter) AppendAudit(ctx context.Context, ev application.AuditEvent) error {
	raw, err := json.Marshal(ev.After)
	if err != nil {
		return errStorage
	}
	_, err = a.tx.Exec(ctx, `INSERT INTO audit_events(organization_id,company_id,actor_user_id,action,entity_type,entity_id,after,reason,correlation_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, ev.OrganizationID, nullable(ev.CompanyID), nullable(ev.ActorUserID), ev.Action, ev.EntityType, ev.EntityID, raw, ev.Reason, ev.CorrelationID)
	return mapErr(err)
}
