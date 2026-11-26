// Package pgstore implements the inventory ports on PostgreSQL with pgx.
// Quantities travel as text and are cast to NUMERIC in SQL, so no value ever
// passes through floating point (ADR-040).
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// MaxAttempts bounds retries of transient errors (serialization/deadlock).
const MaxAttempts = 3

// Store is the unit of work over a pool.
type Store struct {
	pool    *pgxpool.Pool
	retries atomic.Int64
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Retries reports how many transient-error retries happened (for tests and
// diagnostics: lock ordering should keep it at zero).
func (s *Store) Retries() int64 { return s.retries.Load() }

// InTx runs fn in a READ COMMITTED transaction and retries the whole unit
// only on serialization failures or deadlocks, up to MaxAttempts.
func (s *Store) InTx(ctx context.Context, fn func(application.Tx) error) error {
	var err error
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		err = s.once(ctx, fn)
		if !isTransient(err) {
			return err
		}
		s.retries.Add(1)
	}
	return err
}

func (s *Store) once(ctx context.Context, fn func(application.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("pgstore: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(&txAdapter{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isTransient(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && (pe.Code == "40001" || pe.Code == "40P01")
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
	return fmt.Errorf("pgstore: %s: %w", what, err)
}

type txAdapter struct{ tx pgx.Tx }

func parse(s string) (qty.Q, error) {
	q, err := qty.Parse(s)
	if err != nil {
		return qty.Q{}, fmt.Errorf("pgstore: unexpected numeric %q: %w", s, err)
	}
	return q, nil
}

func (t *txAdapter) WarehouseExists(ctx context.Context, company, wh uuid.UUID) (bool, error) {
	var ok bool
	err := t.tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM warehouses WHERE company_id = $1 AND id = $2)`, company, wh).Scan(&ok)
	return ok, mapErr(err, "warehouse")
}

func (t *txAdapter) InsertSKU(ctx context.Context, s domain.SKU) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.CompanyID, s.Code, s.Name, string(s.Kind), s.BaseUnit, s.Precision)
	return mapErr(err, "sku")
}

func (t *txAdapter) GetSKU(ctx context.Context, company, id uuid.UUID) (domain.SKU, error) {
	var s domain.SKU
	var kind string
	err := t.tx.QueryRow(ctx, `
		SELECT id, company_id, code, name, quantity_kind, base_unit, precision
		FROM inventory_skus WHERE company_id = $1 AND id = $2`, company, id).
		Scan(&s.ID, &s.CompanyID, &s.Code, &s.Name, &kind, &s.BaseUnit, &s.Precision)
	s.Kind = domain.QuantityKind(kind)
	return s, mapErr(err, "sku")
}

func (t *txAdapter) InsertPresentation(ctx context.Context, p domain.Presentation) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_presentations (id, company_id, sku_id, name, factor)
		VALUES ($1, $2, $3, $4, $5::numeric)`,
		p.ID, p.CompanyID, p.SKUID, p.Name, p.Factor.String())
	return mapErr(err, "presentation")
}

func (t *txAdapter) GetPresentation(ctx context.Context, company, id uuid.UUID) (domain.Presentation, error) {
	var p domain.Presentation
	var factor string
	err := t.tx.QueryRow(ctx, `
		SELECT id, company_id, sku_id, name, factor::text
		FROM inventory_presentations WHERE company_id = $1 AND id = $2`, company, id).
		Scan(&p.ID, &p.CompanyID, &p.SKUID, &p.Name, &factor)
	if err != nil {
		return p, mapErr(err, "presentation")
	}
	p.Factor, err = parse(factor)
	return p, err
}

func (t *txAdapter) InsertBarcode(ctx context.Context, company uuid.UUID, m application.BarcodeMatch) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_barcodes (company_id, code, sku_id, presentation_id)
		VALUES ($1, $2, $3, $4)`, company, m.Code, m.SKUID, m.PresentationID)
	return mapErr(err, "barcode")
}

func (t *txAdapter) LookupBarcode(ctx context.Context, company uuid.UUID, code string) (application.BarcodeMatch, error) {
	m := application.BarcodeMatch{Code: code}
	err := t.tx.QueryRow(ctx, `
		SELECT sku_id, presentation_id FROM inventory_barcodes
		WHERE company_id = $1 AND code = $2`, company, code).Scan(&m.SKUID, &m.PresentationID)
	return m, mapErr(err, "barcode")
}

func (t *txAdapter) ClaimOperation(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, hash string) ([]byte, bool, error) {
	// A concurrent insert of the same key waits here until the other
	// transaction commits or aborts, so the SELECT below sees its outcome.
	tag, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_operations (company_id, operation_type, operation_id, request_hash)
		VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, company, opType, opID, hash)
	if err != nil {
		return nil, false, mapErr(err, "operation")
	}
	if tag.RowsAffected() == 1 {
		return nil, true, nil
	}
	var storedHash string
	var result []byte
	err = t.tx.QueryRow(ctx, `
		SELECT request_hash, result FROM inventory_operations
		WHERE company_id = $1 AND operation_type = $2 AND operation_id = $3`,
		company, opType, opID).Scan(&storedHash, &result)
	if err != nil {
		return nil, false, mapErr(err, "operation")
	}
	if storedHash != hash {
		return nil, false, fmt.Errorf("%w: %s %s", domain.ErrIdempotencyConflict, opType, opID)
	}
	return result, false, nil
}

func (t *txAdapter) CompleteOperation(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, result []byte) error {
	tag, err := t.tx.Exec(ctx, `
		UPDATE inventory_operations SET result = $4
		WHERE company_id = $1 AND operation_type = $2 AND operation_id = $3 AND result IS NULL`,
		company, opType, opID, result)
	if err != nil {
		return mapErr(err, "operation")
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("pgstore: operation %s %s was not pending", opType, opID)
	}
	return nil
}

func (t *txAdapter) ApplyDelta(ctx context.Context, company uuid.UUID, d application.BalanceDelta) (application.BalanceState, error) {
	if _, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_balances (company_id, warehouse_id, sku_id)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, company, d.Key.WarehouseID, d.Key.SKUID); err != nil {
		return application.BalanceState{}, mapErr(err, "balance")
	}
	// Conditional update (ADR-006): under READ COMMITTED a concurrent
	// writer is waited for and the WHERE is re-evaluated on its result, so
	// two terminals cannot both take the last unit.
	var phys, res, blk, transit string
	err := t.tx.QueryRow(ctx, `
		UPDATE inventory_balances
		SET physical   = physical + $4::numeric,
		    in_transit = in_transit + $5::numeric,
		    reserved   = reserved + $6::numeric,
		    version    = version + 1,
		    updated_at = now()
		WHERE company_id = $1 AND warehouse_id = $2 AND sku_id = $3
		  AND physical + $4::numeric - (reserved + $6::numeric) - blocked >= 0
		  AND in_transit + $5::numeric >= 0
		  AND reserved + $6::numeric >= 0
		RETURNING physical::text, reserved::text, blocked::text, in_transit::text`,
		company, d.Key.WarehouseID, d.Key.SKUID,
		d.Physical.String(), d.InTransit.String(), d.Reserved.String()).
		Scan(&phys, &res, &blk, &transit)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.BalanceState{}, fmt.Errorf("%w: sku %s in warehouse %s",
			domain.ErrInsufficientStock, d.Key.SKUID, d.Key.WarehouseID)
	}
	if err != nil {
		return application.BalanceState{}, mapErr(err, "balance")
	}
	return stateFrom(phys, res, blk, transit)
}

func stateFrom(phys, res, blk, transit string) (application.BalanceState, error) {
	var b application.BalanceState
	var err error
	if b.Physical, err = parse(phys); err != nil {
		return b, err
	}
	if b.Reserved, err = parse(res); err != nil {
		return b, err
	}
	if b.Blocked, err = parse(blk); err != nil {
		return b, err
	}
	b.InTransit, err = parse(transit)
	return b, err
}

func optString(q *qty.Q) *string {
	if q == nil {
		return nil
	}
	s := q.String()
	return &s
}

func (t *txAdapter) AppendMovements(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, ms []application.MovementRecord) error {
	for _, m := range ms {
		var reason *string
		if m.Reason != "" {
			reason = &m.Reason
		}
		_, err := t.tx.Exec(ctx, `
			INSERT INTO inventory_movements (
				company_id, operation_type, operation_id, line_no, movement_type, bucket, direction,
				warehouse_id, sku_id, quantity, balance_after,
				presentation_id, presentation_quantity, presentation_factor, reason, reference_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::numeric, $11::numeric,
				$12, $13::numeric, $14::numeric, $15, $16)`,
			company, opType, opID, m.LineNo, string(m.Type), string(m.Bucket), m.Direction,
			m.WarehouseID, m.SKUID, m.Quantity.String(), m.BalanceAfter.String(),
			m.PresentationID, optString(m.PresentationQuantity), optString(m.PresentationFactor), reason, m.ReferenceID)
		if err != nil {
			return mapErr(err, "movement")
		}
	}
	return nil
}

func (t *txAdapter) GetBalance(ctx context.Context, company, wh, sku uuid.UUID) (application.BalanceState, error) {
	var phys, res, blk, transit string
	err := t.tx.QueryRow(ctx, `
		SELECT physical::text, reserved::text, blocked::text, in_transit::text
		FROM inventory_balances WHERE company_id = $1 AND warehouse_id = $2 AND sku_id = $3`,
		company, wh, sku).Scan(&phys, &res, &blk, &transit)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.BalanceState{}, nil
	}
	if err != nil {
		return application.BalanceState{}, mapErr(err, "balance")
	}
	return stateFrom(phys, res, blk, transit)
}

func (t *txAdapter) InsertReservation(ctx context.Context, r application.Reservation) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_reservations
			(id, company_id, warehouse_id, sku_id, quantity, purpose, owner_ref, status, expires_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $6, $7, $8, $9)`,
		r.ID, r.CompanyID, r.WarehouseID, r.SKUID, r.Quantity.String(), r.Purpose, r.OwnerRef, r.Status, r.ExpiresAt)
	return mapErr(err, "reservation")
}

const reservationCols = `id, company_id, warehouse_id, sku_id, quantity::text, purpose, owner_ref, status, expires_at`

func scanReservation(row pgx.Row) (application.Reservation, error) {
	var r application.Reservation
	var q string
	if err := row.Scan(&r.ID, &r.CompanyID, &r.WarehouseID, &r.SKUID, &q, &r.Purpose, &r.OwnerRef, &r.Status, &r.ExpiresAt); err != nil {
		return r, err
	}
	var err error
	r.Quantity, err = parse(q)
	return r, err
}

func (t *txAdapter) LockReservation(ctx context.Context, company, id uuid.UUID) (application.Reservation, error) {
	r, err := scanReservation(t.tx.QueryRow(ctx,
		`SELECT `+reservationCols+` FROM inventory_reservations WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		company, id))
	return r, mapErr(err, "reservation")
}

func (t *txAdapter) SetReservationStatus(ctx context.Context, company, id uuid.UUID, status string) error {
	tag, err := t.tx.Exec(ctx, `
		UPDATE inventory_reservations SET status = $3, closed_at = now()
		WHERE company_id = $1 AND id = $2 AND status = 'active'`, company, id, status)
	if err != nil {
		return mapErr(err, "reservation")
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: %s", domain.ErrReservationNotActive, id)
	}
	return nil
}

func (t *txAdapter) ExpireDueReservations(ctx context.Context, company uuid.UUID, now time.Time, limit int) ([]application.Reservation, error) {
	rows, err := t.tx.Query(ctx, `
		WITH due AS (
			SELECT id FROM inventory_reservations
			WHERE company_id = $1 AND status = 'active' AND expires_at <= $2
			ORDER BY id
			LIMIT $3
			FOR UPDATE SKIP LOCKED)
		UPDATE inventory_reservations r
		SET status = 'expired', closed_at = now()
		FROM due WHERE r.id = due.id
		RETURNING r.id, r.company_id, r.warehouse_id, r.sku_id, r.quantity::text,
		          r.purpose, r.owner_ref, r.status, r.expires_at`, company, now, limit)
	if err != nil {
		return nil, mapErr(err, "reservation")
	}
	defer rows.Close()
	var out []application.Reservation
	for rows.Next() {
		r, err := scanReservation(rows)
		if err != nil {
			return nil, mapErr(err, "reservation")
		}
		out = append(out, r)
	}
	return out, mapErr(rows.Err(), "reservation")
}

func (t *txAdapter) InsertTransfer(ctx context.Context, tr application.Transfer) error {
	if _, err := t.tx.Exec(ctx, `
		INSERT INTO inventory_transfers (id, company_id, from_warehouse_id, to_warehouse_id, status)
		VALUES ($1, $2, $3, $4, $5)`,
		tr.ID, tr.CompanyID, tr.FromWarehouseID, tr.ToWarehouseID, tr.Status); err != nil {
		return mapErr(err, "transfer")
	}
	for i, l := range tr.Lines {
		if _, err := t.tx.Exec(ctx, `
			INSERT INTO inventory_transfer_lines (company_id, transfer_id, line_no, sku_id, quantity)
			VALUES ($1, $2, $3, $4, $5::numeric)`, tr.CompanyID, tr.ID, i, l.SKUID, l.Quantity.String()); err != nil {
			return mapErr(err, "transfer line")
		}
	}
	return nil
}

func (t *txAdapter) LockTransfer(ctx context.Context, company, id uuid.UUID) (application.Transfer, error) {
	tr := application.Transfer{}
	err := t.tx.QueryRow(ctx, `
		SELECT id, company_id, from_warehouse_id, to_warehouse_id, status
		FROM inventory_transfers WHERE company_id = $1 AND id = $2 FOR UPDATE`, company, id).
		Scan(&tr.ID, &tr.CompanyID, &tr.FromWarehouseID, &tr.ToWarehouseID, &tr.Status)
	if err != nil {
		return tr, mapErr(err, "transfer")
	}
	rows, err := t.tx.Query(ctx, `
		SELECT sku_id, quantity::text FROM inventory_transfer_lines
		WHERE company_id = $1 AND transfer_id = $2 ORDER BY line_no`, company, id)
	if err != nil {
		return tr, mapErr(err, "transfer line")
	}
	defer rows.Close()
	for rows.Next() {
		var l application.TransferLine
		var q string
		if err := rows.Scan(&l.SKUID, &q); err != nil {
			return tr, mapErr(err, "transfer line")
		}
		if l.Quantity, err = parse(q); err != nil {
			return tr, err
		}
		tr.Lines = append(tr.Lines, l)
	}
	return tr, mapErr(rows.Err(), "transfer line")
}

func (t *txAdapter) SetTransferStatus(ctx context.Context, company, id uuid.UUID, status string) error {
	_, err := t.tx.Exec(ctx, `
		UPDATE inventory_transfers SET status = $3,
			received_at = CASE WHEN $3 = 'received' THEN now() ELSE received_at END
		WHERE company_id = $1 AND id = $2`, company, id, status)
	return mapErr(err, "transfer")
}
