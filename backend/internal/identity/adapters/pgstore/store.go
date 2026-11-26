// Package pgstore implements the identity ports on PostgreSQL with pgx
// (BP-008 subtask 3). Every query filters by organization; a row of another
// organization is reported as domain.ErrNotFound.
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
)

// Store is the unit of work over a pool.
type Store struct {
	pool            *pgxpool.Pool
	attemptCapacity int
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// NewAttemptStore requires an explicit capacity. All replicas must use
// the same trusted configuration. The ceiling is a technical guardrail,
// not a production sizing recommendation. New does not enable attempts.
func NewAttemptStore(pool *pgxpool.Pool, capacity int) (*Store, error) {
	if pool == nil || capacity < 1 || capacity > 100000 {
		return nil, domain.ErrInvalidArgument
	}
	return &Store{pool: pool, attemptCapacity: capacity}, nil
}

// InTx runs fn in one READ COMMITTED transaction, committed only if fn
// returns nil.
func (s *Store) InTx(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("identity pgstore: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(&txAdapter{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mapErr(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, what)
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrDuplicate, what)
		case "23503":
			return fmt.Errorf("%w: %s (reference)", domain.ErrNotFound, what)
		}
	}
	return fmt.Errorf("identity pgstore: %s: %w", what, err)
}

type txAdapter struct{ tx pgx.Tx }

const userCols = `id, organization_id, login, display_name, status, disabled_at`

func (t *txAdapter) InsertUser(ctx context.Context, u domain.User) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO identity_users (id, organization_id, login, display_name, status, disabled_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.OrganizationID, u.Login, u.DisplayName, string(u.Status), u.DisabledAt)
	return mapErr(err, "user")
}

func (t *txAdapter) getUser(ctx context.Context, org, id uuid.UUID, lock string) (domain.User, error) {
	var u domain.User
	var status string
	var disabledAt *time.Time
	err := t.tx.QueryRow(ctx, `SELECT `+userCols+` FROM identity_users
		WHERE organization_id = $1 AND id = $2 `+lock, org, id).
		Scan(&u.ID, &u.OrganizationID, &u.Login, &u.DisplayName, &status, &disabledAt)
	if err != nil {
		return domain.User{}, mapErr(err, "user")
	}
	u.Status = domain.Status(status)
	if disabledAt != nil {
		d := disabledAt.UTC()
		u.DisabledAt = &d
	}
	return u, nil
}

func (t *txAdapter) GetUser(ctx context.Context, org, id uuid.UUID) (domain.User, error) {
	return t.getUser(ctx, org, id, "")
}

func (t *txAdapter) LockUser(ctx context.Context, org, id uuid.UUID) (domain.User, error) {
	return t.getUser(ctx, org, id, "FOR UPDATE")
}

func (t *txAdapter) UpdateUser(ctx context.Context, u domain.User) error {
	tag, err := t.tx.Exec(ctx, `
		UPDATE identity_users SET display_name = $3, status = $4, disabled_at = $5
		WHERE organization_id = $1 AND id = $2`,
		u.OrganizationID, u.ID, u.DisplayName, string(u.Status), u.DisabledAt)
	if err != nil {
		return mapErr(err, "user")
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: user %s", domain.ErrNotFound, u.ID)
	}
	return nil
}

func (t *txAdapter) CompanyOrganization(ctx context.Context, company uuid.UUID) (uuid.UUID, error) {
	var org uuid.UUID
	err := t.tx.QueryRow(ctx, `SELECT organization_id FROM companies WHERE id = $1`, company).Scan(&org)
	return org, mapErr(err, "company")
}

func (t *txAdapter) InsertMembership(ctx context.Context, m domain.Membership) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO identity_memberships (organization_id, user_id, company_id) VALUES ($1, $2, $3)`,
		m.OrganizationID, m.UserID, m.CompanyID)
	return mapErr(err, "membership")
}

func (t *txAdapter) ListMemberships(ctx context.Context, org, user uuid.UUID) ([]domain.Membership, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT organization_id, user_id, company_id FROM identity_memberships
		WHERE organization_id = $1 AND user_id = $2 ORDER BY company_id`, org, user)
	if err != nil {
		return nil, mapErr(err, "membership")
	}
	defer rows.Close()
	var out []domain.Membership
	for rows.Next() {
		var m domain.Membership
		if err := rows.Scan(&m.OrganizationID, &m.UserID, &m.CompanyID); err != nil {
			return nil, mapErr(err, "membership")
		}
		out = append(out, m)
	}
	return out, mapErr(rows.Err(), "membership")
}
