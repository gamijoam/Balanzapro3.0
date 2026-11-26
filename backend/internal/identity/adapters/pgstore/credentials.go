package pgstore

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
)

var _ application.PasswordStore = (*Store)(nil)

// Credential errors intentionally discard server details, which may contain
// sensitive values in a failing-row diagnostic.
func credentialError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return domain.ErrDuplicate
		case "23503":
			return domain.ErrNotFound
		case "23514":
			return domain.ErrInvalidArgument
		}
	}
	return application.ErrCredentialStorage
}

func (s *Store) LoadPassword(ctx context.Context, org, user uuid.UUID) (string, error) {
	var verifier string
	err := s.pool.QueryRow(ctx, `SELECT c.password_verifier FROM identity_credentials c
 JOIN identity_users u ON u.organization_id=c.organization_id AND u.id=c.user_id
 WHERE c.organization_id=$1 AND c.user_id=$2 AND u.status='active'`, org, user).Scan(&verifier)
	if err != nil {
		return "", credentialError(err)
	}
	return verifier, nil
}

// credentialWrite serializes state checks with DisableUser's user-row lock.
func (s *Store) credentialWrite(ctx context.Context, org, user uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return credentialError(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var active bool
	err = tx.QueryRow(ctx, `SELECT status='active' FROM identity_users WHERE organization_id=$1 AND id=$2 FOR UPDATE`, org, user).Scan(&active)
	if err != nil {
		return credentialError(err)
	}
	if !active {
		return domain.ErrNotFound
	}
	if err = fn(tx); err != nil {
		return err
	}
	return credentialError(tx.Commit(ctx))
}

func (s *Store) InsertPassword(ctx context.Context, org, user uuid.UUID, verifier string) error {
	return s.credentialWrite(ctx, org, user, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO identity_credentials(organization_id,user_id,password_verifier) VALUES($1,$2,$3)`, org, user, verifier)
		return credentialError(err)
	})
}

func (s *Store) ReplacePassword(ctx context.Context, org, user uuid.UUID, expected, replacement string) error {
	return s.credentialWrite(ctx, org, user, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE identity_credentials SET password_verifier=$4,updated_at=now()
    WHERE organization_id=$1 AND user_id=$2 AND password_verifier=$3`, org, user, expected, replacement)
		if err != nil {
			return credentialError(err)
		}
		if tag.RowsAffected() != 1 {
			return application.ErrCredentialConflict
		}
		return nil
	})
}
