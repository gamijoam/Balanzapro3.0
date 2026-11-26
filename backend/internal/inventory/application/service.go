package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// Service implements the inventory use cases.
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

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{domain.ErrInvalidArgument}, a...)...)
}

func requireScope(sc Scope) error {
	if sc.CompanyID == uuid.Nil {
		return invalid("scope without company")
	}
	return nil
}

// RegisterSKU creates a SKU in the scope's company.
func (s *Service) RegisterSKU(ctx context.Context, sc Scope, c RegisterSKUCommand) (domain.SKU, error) {
	if err := requireScope(sc); err != nil {
		return domain.SKU{}, err
	}
	sku, err := domain.NewSKU(c.ID, sc.CompanyID, c.Code, c.Name, c.Kind, c.BaseUnit, c.Precision)
	if err != nil {
		return domain.SKU{}, err
	}
	err = s.store.InTx(ctx, func(tx Tx) error { return tx.InsertSKU(ctx, sku) })
	return sku, err
}

// AddPresentation adds an exact packaging to a SKU.
func (s *Service) AddPresentation(ctx context.Context, sc Scope, c AddPresentationCommand) (domain.Presentation, error) {
	if err := requireScope(sc); err != nil {
		return domain.Presentation{}, err
	}
	factor, err := qty.Parse(c.Factor)
	if err != nil {
		return domain.Presentation{}, invalid("factor: %v", err)
	}
	var p domain.Presentation
	err = s.store.InTx(ctx, func(tx Tx) error {
		sku, err := tx.GetSKU(ctx, sc.CompanyID, c.SKUID)
		if err != nil {
			return err
		}
		if p, err = domain.NewPresentation(c.ID, sku, c.Name, factor); err != nil {
			return err
		}
		return tx.InsertPresentation(ctx, p)
	})
	return p, err
}

// AddBarcode links a unique code to a SKU or one of its presentations.
func (s *Service) AddBarcode(ctx context.Context, sc Scope, c AddBarcodeCommand) error {
	if err := requireScope(sc); err != nil {
		return err
	}
	code, err := domain.NormalizeBarcode(c.Code)
	if err != nil {
		return err
	}
	return s.store.InTx(ctx, func(tx Tx) error {
		if _, err := tx.GetSKU(ctx, sc.CompanyID, c.SKUID); err != nil {
			return err
		}
		if c.PresentationID != nil {
			p, err := tx.GetPresentation(ctx, sc.CompanyID, *c.PresentationID)
			if err != nil {
				return err
			}
			if p.SKUID != c.SKUID {
				return invalid("presentation %s does not belong to SKU %s", p.ID, c.SKUID)
			}
		}
		return tx.InsertBarcode(ctx, sc.CompanyID, BarcodeMatch{Code: code, SKUID: c.SKUID, PresentationID: c.PresentationID})
	})
}

// LookupBarcode resolves a scanned code within the company.
func (s *Service) LookupBarcode(ctx context.Context, sc Scope, code string) (BarcodeMatch, error) {
	if err := requireScope(sc); err != nil {
		return BarcodeMatch{}, err
	}
	c, err := domain.NormalizeBarcode(code)
	if err != nil {
		return BarcodeMatch{}, err
	}
	var m BarcodeMatch
	err = s.store.InTx(ctx, func(tx Tx) error {
		var err error
		m, err = tx.LookupBarcode(ctx, sc.CompanyID, c)
		return err
	})
	return m, err
}

// Balance returns the stock of a SKU in a warehouse of the company.
func (s *Service) Balance(ctx context.Context, sc Scope, warehouse, sku uuid.UUID) (BalanceState, error) {
	if err := requireScope(sc); err != nil {
		return BalanceState{}, err
	}
	var b BalanceState
	err := s.store.InTx(ctx, func(tx Tx) error {
		if err := s.checkWarehouse(ctx, tx, sc, warehouse); err != nil {
			return err
		}
		if _, err := tx.GetSKU(ctx, sc.CompanyID, sku); err != nil {
			return err
		}
		var err error
		b, err = tx.GetBalance(ctx, sc.CompanyID, warehouse, sku)
		return err
	})
	return b, err
}

func (s *Service) checkWarehouse(ctx context.Context, tx Tx, sc Scope, wh uuid.UUID) error {
	ok, err := tx.WarehouseExists(ctx, sc.CompanyID, wh)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: warehouse %s", domain.ErrNotFound, wh)
	}
	return nil
}

// resolved is a line converted to base units with its presentation snapshot.
type resolved struct {
	sku          domain.SKU
	base         qty.Q
	presentation *domain.Presentation
	presQty      *qty.Q
}

func (s *Service) resolveLine(ctx context.Context, tx Tx, sc Scope, l Line) (resolved, error) {
	n, err := qty.Parse(l.Quantity)
	if err != nil {
		return resolved{}, invalid("quantity: %v", err)
	}
	sku, err := tx.GetSKU(ctx, sc.CompanyID, l.SKUID)
	if err != nil {
		return resolved{}, err
	}
	r := resolved{sku: sku, base: n}
	if l.PresentationID != nil {
		p, err := tx.GetPresentation(ctx, sc.CompanyID, *l.PresentationID)
		if err != nil {
			return resolved{}, err
		}
		if p.SKUID != sku.ID {
			return resolved{}, invalid("presentation %s does not belong to SKU %s", p.ID, sku.ID)
		}
		if !n.IsPositive() {
			return resolved{}, invalid("quantity must be positive")
		}
		r.base = p.ToBase(n)
		r.presentation, r.presQty = &p, &n
	}
	if err := sku.ValidateQuantity(r.base); err != nil {
		return resolved{}, err
	}
	return r, nil
}

func (s *Service) resolveLines(ctx context.Context, tx Tx, sc Scope, lines []Line) ([]resolved, error) {
	if len(lines) == 0 {
		return nil, invalid("at least one line is required")
	}
	out := make([]resolved, 0, len(lines))
	for i, l := range lines {
		r, err := s.resolveLine(ctx, tx, sc, l)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// canonicalLine rewrites the quantity in canonical form ("2.50" → "2.5") so
// equal requests hash equally (idempotency).
func canonicalLine(l Line) (Line, error) {
	q, err := qty.Parse(l.Quantity)
	if err != nil {
		return l, invalid("quantity: %v", err)
	}
	l.Quantity = q.String()
	return l, nil
}

func canonicalLines(lines []Line) ([]Line, error) {
	out := make([]Line, len(lines))
	for i, l := range lines {
		c, err := canonicalLine(l)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i, err)
		}
		out[i] = c
	}
	return out, nil
}

func hashPayload(opType string, payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(opType))
	h.Write([]byte{0})
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// run executes an idempotent operation: claim the key, run fn, store the
// result, all in one transaction. An identical retry returns the stored
// result without effects; a different payload is ErrIdempotencyConflict.
// Business failures roll back the claim, so a retry is evaluated again.
func (s *Service) run(ctx context.Context, sc Scope, opType string, opID uuid.UUID, payload any,
	fn func(Tx) (OperationResult, error)) (OperationResult, error) {
	if err := requireScope(sc); err != nil {
		return OperationResult{}, err
	}
	if opID == uuid.Nil {
		return OperationResult{}, invalid("operation id is required")
	}
	hash, err := hashPayload(opType, payload)
	if err != nil {
		return OperationResult{}, err
	}
	var res OperationResult
	err = s.store.InTx(ctx, func(tx Tx) error {
		res = OperationResult{}
		prior, claimed, err := tx.ClaimOperation(ctx, sc.CompanyID, opType, opID, hash)
		if err != nil {
			return err
		}
		if !claimed {
			if err := json.Unmarshal(prior, &res); err != nil {
				return fmt.Errorf("inventory: stored result of %s: %w", opID, err)
			}
			res.Replayed = true
			return nil
		}
		if res, err = fn(tx); err != nil {
			return err
		}
		res.OperationID, res.OperationType = opID, opType
		if res.Movements == nil {
			res.Movements = []MovementRecord{}
		}
		b, err := json.Marshal(res)
		if err != nil {
			return err
		}
		return tx.CompleteOperation(ctx, sc.CompanyID, opType, opID, b)
	})
	if err != nil {
		return OperationResult{}, err
	}
	return res, nil
}

// apply plans the deltas (merge + stable lock order) and applies them.
func apply(ctx context.Context, tx Tx, sc Scope, deltas []BalanceDelta) (map[BalanceKey]BalanceState, error) {
	states := map[BalanceKey]BalanceState{}
	for _, d := range PlanDeltas(deltas) {
		st, err := tx.ApplyDelta(ctx, sc.CompanyID, d)
		if err != nil {
			return nil, err
		}
		states[d.Key] = st
	}
	return states, nil
}

func movementFor(lineNo int, t domain.MovementType, b domain.Bucket, dir int, wh uuid.UUID, r resolved) MovementRecord {
	m := MovementRecord{LineNo: lineNo, Type: t, Bucket: b, Direction: dir, WarehouseID: wh, SKUID: r.sku.ID, Quantity: r.base}
	if r.presentation != nil {
		id, f := r.presentation.ID, r.presentation.Factor
		m.PresentationID, m.PresentationFactor, m.PresentationQuantity = &id, &f, r.presQty
	}
	return m
}

func after(m MovementRecord, st BalanceState) MovementRecord {
	if m.Bucket == domain.BucketInTransit {
		m.BalanceAfter = st.InTransit
	} else {
		m.BalanceAfter = st.Physical
	}
	return m
}

// PostStock posts opening stock, adjustments or issues in one warehouse.
// All lines apply or none; balance_after is the balance once the whole
// operation is applied.
func (s *Service) PostStock(ctx context.Context, sc Scope, c StockCommand) (OperationResult, error) {
	var mt domain.MovementType
	dir := 1
	switch c.Kind {
	case StockOpening:
		mt = domain.MoveOpening
	case StockAdjustmentIn:
		mt = domain.MoveAdjustmentIn
	case StockAdjustmentOut:
		mt, dir = domain.MoveAdjustmentOut, -1
	case StockIssue:
		mt, dir = domain.MoveIssue, -1
	default:
		return OperationResult{}, invalid("unknown stock kind %q", c.Kind)
	}
	reason := strings.TrimSpace(c.Reason)
	if (c.Kind == StockAdjustmentIn || c.Kind == StockAdjustmentOut) && reason == "" {
		return OperationResult{}, invalid("adjustments require a reason")
	}
	var err error
	if c.Lines, err = canonicalLines(c.Lines); err != nil {
		return OperationResult{}, err
	}
	return s.run(ctx, sc, OpStock, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		if err := s.checkWarehouse(ctx, tx, sc, c.WarehouseID); err != nil {
			return OperationResult{}, err
		}
		lines, err := s.resolveLines(ctx, tx, sc, c.Lines)
		if err != nil {
			return OperationResult{}, err
		}
		deltas := make([]BalanceDelta, 0, len(lines))
		for _, r := range lines {
			q := r.base
			if dir < 0 {
				q = q.Neg()
			}
			deltas = append(deltas, BalanceDelta{Key: BalanceKey{c.WarehouseID, r.sku.ID}, Physical: q})
		}
		states, err := apply(ctx, tx, sc, deltas)
		if err != nil {
			return OperationResult{}, err
		}
		res := OperationResult{}
		for i, r := range lines {
			m := movementFor(i, mt, domain.BucketPhysical, dir, c.WarehouseID, r)
			m.Reason = reason
			res.Movements = append(res.Movements, after(m, states[BalanceKey{c.WarehouseID, r.sku.ID}]))
		}
		return res, tx.AppendMovements(ctx, sc.CompanyID, OpStock, c.OperationID, res.Movements)
	})
}

// ShipTransfer takes stock out of the origin and puts it in transit at the
// destination. The destination physical stock changes only on receipt.
func (s *Service) ShipTransfer(ctx context.Context, sc Scope, c ShipTransferCommand) (OperationResult, error) {
	if c.TransferID == uuid.Nil {
		return OperationResult{}, invalid("transfer id is required")
	}
	if c.FromWarehouseID == c.ToWarehouseID {
		return OperationResult{}, invalid("origin and destination must differ")
	}
	var err error
	if c.Lines, err = canonicalLines(c.Lines); err != nil {
		return OperationResult{}, err
	}
	return s.run(ctx, sc, OpTransferShip, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		for _, wh := range []uuid.UUID{c.FromWarehouseID, c.ToWarehouseID} {
			if err := s.checkWarehouse(ctx, tx, sc, wh); err != nil {
				return OperationResult{}, err
			}
		}
		lines, err := s.resolveLines(ctx, tx, sc, c.Lines)
		if err != nil {
			return OperationResult{}, err
		}
		tr := Transfer{ID: c.TransferID, CompanyID: sc.CompanyID, FromWarehouseID: c.FromWarehouseID,
			ToWarehouseID: c.ToWarehouseID, Status: TransferShipped}
		var deltas []BalanceDelta
		for _, r := range lines {
			tr.Lines = append(tr.Lines, TransferLine{SKUID: r.sku.ID, Quantity: r.base})
			deltas = append(deltas,
				BalanceDelta{Key: BalanceKey{c.FromWarehouseID, r.sku.ID}, Physical: r.base.Neg()},
				BalanceDelta{Key: BalanceKey{c.ToWarehouseID, r.sku.ID}, InTransit: r.base})
		}
		states, err := apply(ctx, tx, sc, deltas)
		if err != nil {
			return OperationResult{}, err
		}
		if err := tx.InsertTransfer(ctx, tr); err != nil {
			return OperationResult{}, err
		}
		res := OperationResult{TransferID: &tr.ID}
		for i, r := range lines {
			out := movementFor(i, domain.MoveTransferOut, domain.BucketPhysical, -1, c.FromWarehouseID, r)
			in := movementFor(i, domain.MoveTransitIn, domain.BucketInTransit, 1, c.ToWarehouseID, r)
			out.ReferenceID, in.ReferenceID = &tr.ID, &tr.ID
			res.Movements = append(res.Movements,
				after(out, states[BalanceKey{c.FromWarehouseID, r.sku.ID}]),
				after(in, states[BalanceKey{c.ToWarehouseID, r.sku.ID}]))
		}
		return res, tx.AppendMovements(ctx, sc.CompanyID, OpTransferShip, c.OperationID, res.Movements)
	})
}

// ReceiveTransfer confirms the full transfer at the destination, once.
func (s *Service) ReceiveTransfer(ctx context.Context, sc Scope, c ReceiveTransferCommand) (OperationResult, error) {
	return s.run(ctx, sc, OpTransferReceive, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		tr, err := tx.LockTransfer(ctx, sc.CompanyID, c.TransferID)
		if err != nil {
			return OperationResult{}, err
		}
		if tr.Status != TransferShipped {
			return OperationResult{}, fmt.Errorf("%w: %s is %s", domain.ErrTransferNotShipped, tr.ID, tr.Status)
		}
		var deltas []BalanceDelta
		for _, l := range tr.Lines {
			deltas = append(deltas, BalanceDelta{Key: BalanceKey{tr.ToWarehouseID, l.SKUID}, InTransit: l.Quantity.Neg(), Physical: l.Quantity})
		}
		states, err := apply(ctx, tx, sc, deltas)
		if err != nil {
			return OperationResult{}, err
		}
		if err := tx.SetTransferStatus(ctx, sc.CompanyID, tr.ID, TransferReceived); err != nil {
			return OperationResult{}, err
		}
		res := OperationResult{TransferID: &tr.ID}
		for i, l := range tr.Lines {
			st := states[BalanceKey{tr.ToWarehouseID, l.SKUID}]
			base := MovementRecord{LineNo: i, WarehouseID: tr.ToWarehouseID, SKUID: l.SKUID, Quantity: l.Quantity, ReferenceID: &tr.ID}
			out, in := base, base
			out.Type, out.Bucket, out.Direction, out.BalanceAfter = domain.MoveTransitOut, domain.BucketInTransit, -1, st.InTransit
			in.Type, in.Bucket, in.Direction, in.BalanceAfter = domain.MoveTransferIn, domain.BucketPhysical, 1, st.Physical
			res.Movements = append(res.Movements, out, in)
		}
		return res, tx.AppendMovements(ctx, sc.CompanyID, OpTransferReceive, c.OperationID, res.Movements)
	})
}

// Reserve holds available stock until ExpiresAt.
func (s *Service) Reserve(ctx context.Context, sc Scope, c ReserveCommand) (OperationResult, error) {
	purpose := strings.TrimSpace(c.Purpose)
	switch {
	case c.ReservationID == uuid.Nil:
		return OperationResult{}, invalid("reservation id is required")
	case purpose == "":
		return OperationResult{}, invalid("reservation purpose is required")
	}
	var err error
	if c.Line, err = canonicalLine(c.Line); err != nil {
		return OperationResult{}, err
	}
	c.ExpiresAt = c.ExpiresAt.UTC()
	return s.run(ctx, sc, OpReserve, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		// Checked after the idempotency claim: an identical retry that
		// arrives after the expiry replays the stored result.
		if !c.ExpiresAt.After(s.now()) {
			return OperationResult{}, invalid("reservation must expire in the future")
		}
		if err := s.checkWarehouse(ctx, tx, sc, c.WarehouseID); err != nil {
			return OperationResult{}, err
		}
		r, err := s.resolveLine(ctx, tx, sc, c.Line)
		if err != nil {
			return OperationResult{}, err
		}
		key := BalanceKey{c.WarehouseID, r.sku.ID}
		states, err := apply(ctx, tx, sc, []BalanceDelta{{Key: key, Reserved: r.base}})
		if err != nil {
			return OperationResult{}, err
		}
		res := Reservation{ID: c.ReservationID, CompanyID: sc.CompanyID, WarehouseID: c.WarehouseID,
			SKUID: r.sku.ID, Quantity: r.base, Purpose: purpose, OwnerRef: strings.TrimSpace(c.OwnerRef),
			Status: ReservationActive, ExpiresAt: c.ExpiresAt}
		if err := tx.InsertReservation(ctx, res); err != nil {
			return OperationResult{}, err
		}
		return OperationResult{ReservationID: &c.ReservationID},
			recordEvent(ctx, tx, sc, res, ReservationEventCreated, states[key], OpReserve, c.OperationID)
	})
}

// lockActive locks a reservation that is still active (not closed) and
// reports whether its expiry already passed without being swept.
func (s *Service) lockActive(ctx context.Context, tx Tx, sc Scope, id uuid.UUID) (r Reservation, expired bool, err error) {
	r, err = tx.LockReservation(ctx, sc.CompanyID, id)
	if err != nil {
		return r, false, err
	}
	if r.Status != ReservationActive {
		return r, false, fmt.Errorf("%w: %s is %s", domain.ErrReservationNotActive, id, r.Status)
	}
	return r, !s.now().Before(r.ExpiresAt), nil
}

// ReleaseReservation returns the reserved quantity to available, once. An
// expired reservation not yet swept is released too and closed as expired,
// so stock never stays held waiting for the sweeper.
func (s *Service) ReleaseReservation(ctx context.Context, sc Scope, c ReservationCommand) (OperationResult, error) {
	return s.run(ctx, sc, OpReservationRelease, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		r, expired, err := s.lockActive(ctx, tx, sc, c.ReservationID)
		if err != nil {
			return OperationResult{}, err
		}
		key := BalanceKey{r.WarehouseID, r.SKUID}
		states, err := apply(ctx, tx, sc, []BalanceDelta{{Key: key, Reserved: r.Quantity.Neg()}})
		if err != nil {
			return OperationResult{}, err
		}
		status := ReservationReleased
		if expired {
			status = ReservationExpired
		}
		if err := tx.SetReservationStatus(ctx, sc.CompanyID, r.ID, status); err != nil {
			return OperationResult{}, err
		}
		return OperationResult{ReservationID: &r.ID},
			recordEvent(ctx, tx, sc, r, status, states[key], OpReservationRelease, c.OperationID)
	})
}

// ConsumeReservation turns the reservation into a physical exit, atomically.
func (s *Service) ConsumeReservation(ctx context.Context, sc Scope, c ReservationCommand) (OperationResult, error) {
	return s.run(ctx, sc, OpReservationConsume, c.OperationID, c, func(tx Tx) (OperationResult, error) {
		r, expired, err := s.lockActive(ctx, tx, sc, c.ReservationID)
		if err != nil {
			return OperationResult{}, err
		}
		if expired {
			return OperationResult{}, fmt.Errorf("%w: %s expired at %s", domain.ErrReservationNotActive, r.ID, r.ExpiresAt.Format(time.RFC3339))
		}
		key := BalanceKey{r.WarehouseID, r.SKUID}
		states, err := apply(ctx, tx, sc, []BalanceDelta{{Key: key, Reserved: r.Quantity.Neg(), Physical: r.Quantity.Neg()}})
		if err != nil {
			return OperationResult{}, err
		}
		if err := tx.SetReservationStatus(ctx, sc.CompanyID, r.ID, ReservationConsumed); err != nil {
			return OperationResult{}, err
		}
		m := MovementRecord{Type: domain.MoveReservationOut, Bucket: domain.BucketPhysical, Direction: -1,
			WarehouseID: r.WarehouseID, SKUID: r.SKUID, Quantity: r.Quantity, BalanceAfter: states[key].Physical, ReferenceID: &r.ID}
		res := OperationResult{ReservationID: &r.ID, Movements: []MovementRecord{m}}
		if err := recordEvent(ctx, tx, sc, r, ReservationEventConsumed, states[key], OpReservationConsume, c.OperationID); err != nil {
			return OperationResult{}, err
		}
		return res, tx.AppendMovements(ctx, sc.CompanyID, OpReservationConsume, c.OperationID, res.Movements)
	})
}

// ExpireReservations releases up to limit reservations whose expiry passed
// and reports how many it released. Each one is released exactly once.
func (s *Service) ExpireReservations(ctx context.Context, sc Scope, limit int) (int, error) {
	if err := requireScope(sc); err != nil {
		return 0, err
	}
	if limit <= 0 {
		return 0, invalid("limit must be positive")
	}
	var n int
	err := s.store.InTx(ctx, func(tx Tx) error {
		due, err := tx.ExpireDueReservations(ctx, sc.CompanyID, s.now(), limit)
		if err != nil {
			return err
		}
		// One delta per reservation, in stable (balance, reservation) order,
		// so each event records the reserved balance right after it.
		sort.Slice(due, func(i, j int) bool {
			ki, kj := BalanceKey{due[i].WarehouseID, due[i].SKUID}, BalanceKey{due[j].WarehouseID, due[j].SKUID}
			if ki != kj {
				return lessKey(ki, kj)
			}
			return bytes.Compare(due[i].ID[:], due[j].ID[:]) < 0
		})
		batch := uuid.New()
		for _, r := range due {
			key := BalanceKey{r.WarehouseID, r.SKUID}
			st, err := tx.ApplyDelta(ctx, sc.CompanyID, BalanceDelta{Key: key, Reserved: r.Quantity.Neg()})
			if err != nil {
				return err
			}
			if err := recordEvent(ctx, tx, sc, r, ReservationEventExpired, st, OpReservationExpire, batch); err != nil {
				return err
			}
		}
		n = len(due)
		return nil
	})
	return n, err
}

// SweepExpiredReservations runs ExpireReservations for each company in
// batches until no due reservation is left, and returns how many it freed.
// bpserver runs it periodically.
func (s *Service) SweepExpiredReservations(ctx context.Context, companies []uuid.UUID, batch int) (int, error) {
	total := 0
	var errs []error
	for _, company := range companies {
		for {
			n, err := s.ExpireReservations(ctx, Scope{CompanyID: company}, batch)
			total += n
			if err != nil {
				// Keep sweeping the other companies; report all failures.
				errs = append(errs, fmt.Errorf("company %s: %w", company, err))
				break
			}
			if n < batch {
				break
			}
		}
	}
	return total, errors.Join(errs...)
}

func recordEvent(ctx context.Context, tx Tx, sc Scope, r Reservation, typ string, st BalanceState, opType string, opID uuid.UUID) error {
	return tx.AppendReservationEvent(ctx, sc.CompanyID, ReservationEvent{
		ReservationID: r.ID, Type: typ, WarehouseID: r.WarehouseID, SKUID: r.SKUID,
		Quantity: r.Quantity, ReservedAfter: st.Reserved, OperationType: opType, OperationID: opID,
	})
}

// ReservationHistory returns the append-only events of a reservation of the
// company, oldest first.
func (s *Service) ReservationHistory(ctx context.Context, sc Scope, id uuid.UUID) ([]ReservationEvent, error) {
	if err := requireScope(sc); err != nil {
		return nil, err
	}
	var evs []ReservationEvent
	err := s.store.InTx(ctx, func(tx Tx) error {
		var err error
		if evs, err = tx.ListReservationEvents(ctx, sc.CompanyID, id); err != nil {
			return err
		}
		if len(evs) == 0 {
			// Existing reservation without events (e.g. written before the
			// history existed): empty history, not "not found".
			_, err := tx.GetReservation(ctx, sc.CompanyID, id)
			return err
		}
		return nil
	})
	return evs, err
}
