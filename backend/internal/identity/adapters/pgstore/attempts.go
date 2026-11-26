package pgstore

import (
	"bytes"
	"context"
	"slices"
	"time"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var _ application.AttemptStore = (*Store)(nil)

var _ application.AttemptBatchStore = (*Store)(nil)

func (s *Store) ReserveAttempt(ctx context.Context, key application.AttemptKey, p domain.AttemptPolicy) (bool, error) {
	return s.ReserveAttempts(ctx, []application.AttemptBudget{{Key: key, Policy: p}})
}

// Limit the number of rows locked per request and use a stable order for
// both inserts and locks, regardless of the caller's input order.
func (s *Store) ReserveAttempts(ctx context.Context, budgets []application.AttemptBudget) (bool, error) {
	if s == nil || s.attemptCapacity < 1 || len(budgets) < 1 || len(budgets) > 3 {
		return false, domain.ErrInvalidArgument
	}
	ordered := slices.Clone(budgets)
	first := ordered[0].Key
	for _, b := range ordered {
		key, p := b.Key, b.Policy
		if err := p.Validate(); err != nil {
			return false, err
		}
		if key.OrganizationID == uuid.Nil || key.OrganizationID != first.OrganizationID ||
			key.Kind != first.Kind || key.Digest == ([32]byte{}) ||
			(key.Kind != "password" && key.Kind != "pin" && key.Kind != "totp") ||
			p.Window%time.Microsecond != 0 || p.Cooldown%time.Microsecond != 0 {
			return false, domain.ErrInvalidArgument
		}
	}
	slices.SortFunc(ordered, func(a, b application.AttemptBudget) int { return bytes.Compare(a.Key.Digest[:], b.Key.Digest[:]) })
	for i := 1; i < len(ordered); i++ {
		if ordered[i].Key == ordered[i-1].Key {
			return false, domain.ErrInvalidArgument
		}
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return false, application.ErrAttemptStorage
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	// Serialize all budget reservations for this organization. The lock
	// precedes all budget-row locks, including those for existing keys.
	// NO KEY UPDATE is compatible with foreign-key reference checks.
	var org uuid.UUID
	if err := tx.QueryRow(ctx, "SELECT id FROM organizations WHERE id=$1 FOR NO KEY UPDATE", first.OrganizationID).Scan(&org); err != nil {
		return false, application.ErrAttemptStorage
	}
	for _, b := range ordered {
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2 AND kind=$3)", b.Key.OrganizationID, b.Key.Digest[:], b.Key.Kind).Scan(&exists); err != nil {
			return false, application.ErrAttemptStorage
		}
		if !exists {
			var count int
			// Limit the scan even if preexisting data exceeds the configured cap.
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM (SELECT 1 FROM identity_attempt_budgets WHERE organization_id=$1 LIMIT $2) AS bounded", org, s.attemptCapacity).Scan(&count); err != nil {
				return false, application.ErrAttemptStorage
			}
			if count >= s.attemptCapacity {
				return false, application.ErrAttemptCapacity
			}
		}
		allowed, err := reserveInTx(ctx, tx, b.Key, b.Policy)
		if err != nil || !allowed {
			return false, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return false, application.ErrAttemptStorage
	}
	return true, nil
}

func reserveInTx(ctx context.Context, tx pgx.Tx, key application.AttemptKey, p domain.AttemptPolicy) (bool, error) {
	_, err := tx.Exec(ctx, `INSERT INTO identity_attempt_budgets(organization_id,budget_key,kind,max_attempts,window_us,cooldown_us)
 VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(organization_id,budget_key,kind) DO NOTHING`,
		key.OrganizationID, key.Digest[:], key.Kind, p.MaxAttempts, p.Window.Microseconds(), p.Cooldown.Microseconds())
	if err != nil {
		return false, application.ErrAttemptStorage
	}
	var max int
	var window, cooldown int64
	var state domain.AttemptState
	var start, last, blocked *time.Time
	err = tx.QueryRow(ctx, `SELECT max_attempts,window_us,cooldown_us,used,window_start,last_reserved,blocked_until
 FROM identity_attempt_budgets WHERE organization_id=$1 AND budget_key=$2 AND kind=$3 FOR UPDATE`,
		key.OrganizationID, key.Digest[:], key.Kind).Scan(&max, &window, &cooldown, &state.Used, &start, &last, &blocked)
	if err != nil {
		return false, application.ErrAttemptStorage
	}
	if max != p.MaxAttempts || window != p.Window.Microseconds() || cooldown != p.Cooldown.Microseconds() {
		return false, application.ErrAttemptPolicyChanged
	}
	if start != nil {
		state.WindowStart = start.UTC()
	}
	if last != nil {
		state.LastReserved = last.UTC()
	}
	if blocked != nil {
		state.BlockedUntil = blocked.UTC()
	}
	// Obtain DB time AFTER acquiring the lock; transaction-start timestamps
	// may be older than the preceding committed reservation.
	var now time.Time
	if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return false, application.ErrAttemptStorage
	}
	next, allowed, err := p.Reserve(state, now)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}
	var until *time.Time
	if !next.BlockedUntil.IsZero() {
		until = &next.BlockedUntil
	}
	tag, err := tx.Exec(ctx, `UPDATE identity_attempt_budgets SET used=$4,window_start=$5,last_reserved=$6,blocked_until=$7
 WHERE organization_id=$1 AND budget_key=$2 AND kind=$3`,
		key.OrganizationID, key.Digest[:], key.Kind, next.Used, next.WindowStart, next.LastReserved, until)
	if err != nil || tag.RowsAffected() != 1 {
		return false, application.ErrAttemptStorage
	}
	return true, nil
}
