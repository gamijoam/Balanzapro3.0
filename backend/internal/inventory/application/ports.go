// Package application orchestrates inventory use cases: validation, scope,
// idempotency, lock ordering and kardex, over the ports a storage adapter
// implements. Every operation runs in one transaction (ADR-006/016).
package application

import (
	"bytes"
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// Scope is the minimal identity port until Fase 03 (ADR-169): every call
// carries the company it acts for, and storage filters by it.
type Scope struct {
	CompanyID uuid.UUID
}

// BalanceKey identifies one stock balance (one writer per key, ADR-005).
type BalanceKey struct {
	WarehouseID uuid.UUID
	SKUID       uuid.UUID
}

// BalanceDelta is a signed change to the buckets of one balance.
type BalanceDelta struct {
	Key       BalanceKey
	Physical  qty.Q
	InTransit qty.Q
	Reserved  qty.Q
}

// BalanceState is a balance after a change.
type BalanceState struct {
	Physical  qty.Q `json:"physical"`
	Reserved  qty.Q `json:"reserved"`
	Blocked   qty.Q `json:"blocked"`
	InTransit qty.Q `json:"in_transit"`
}

// Available = physical - reserved - blocked.
func (b BalanceState) Available() qty.Q { return b.Physical.Sub(b.Reserved).Sub(b.Blocked) }

// MovementRecord is one append-only kardex entry.
type MovementRecord struct {
	LineNo               int                 `json:"line_no"`
	Type                 domain.MovementType `json:"type"`
	Bucket               domain.Bucket       `json:"bucket"`
	Direction            int                 `json:"direction"`
	WarehouseID          uuid.UUID           `json:"warehouse_id"`
	SKUID                uuid.UUID           `json:"sku_id"`
	Quantity             qty.Q               `json:"quantity"`
	BalanceAfter         qty.Q               `json:"balance_after"`
	PresentationID       *uuid.UUID          `json:"presentation_id,omitempty"`
	PresentationQuantity *qty.Q              `json:"presentation_quantity,omitempty"`
	PresentationFactor   *qty.Q              `json:"presentation_factor,omitempty"`
	Reason               string              `json:"reason,omitempty"`
	ReferenceID          *uuid.UUID          `json:"reference_id,omitempty"`
}

// Reservation statuses.
const (
	ReservationActive   = "active"
	ReservationReleased = "released"
	ReservationConsumed = "consumed"
	ReservationExpired  = "expired"
)

// Reservation holds available stock for a purpose until it expires (ADR-068).
type Reservation struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	WarehouseID uuid.UUID
	SKUID       uuid.UUID
	Quantity    qty.Q
	Purpose     string
	OwnerRef    string
	Status      string
	ExpiresAt   time.Time
}

// Transfer statuses.
const (
	TransferShipped  = "shipped"
	TransferReceived = "received"
)

// Transfer moves stock between warehouses of one company; only receiving
// confirms the destination (ADR-067).
type Transfer struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	FromWarehouseID uuid.UUID
	ToWarehouseID   uuid.UUID
	Status          string
	Lines           []TransferLine
}

// TransferLine is a base-unit quantity of a SKU.
type TransferLine struct {
	SKUID    uuid.UUID
	Quantity qty.Q
}

// BarcodeMatch is what a scanned code identifies.
type BarcodeMatch struct {
	Code           string     `json:"code"`
	SKUID          uuid.UUID  `json:"sku_id"`
	PresentationID *uuid.UUID `json:"presentation_id,omitempty"`
}

// Store is the unit of work: fn runs inside one transaction, committed only
// if fn returns nil.
type Store interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

// Tx is everything the use cases need from storage. Every method filters by
// company; a row of another company is reported as domain.ErrNotFound.
type Tx interface {
	WarehouseExists(ctx context.Context, company, warehouse uuid.UUID) (bool, error)

	InsertSKU(ctx context.Context, s domain.SKU) error
	GetSKU(ctx context.Context, company, id uuid.UUID) (domain.SKU, error)
	InsertPresentation(ctx context.Context, p domain.Presentation) error
	GetPresentation(ctx context.Context, company, id uuid.UUID) (domain.Presentation, error)
	InsertBarcode(ctx context.Context, company uuid.UUID, m BarcodeMatch) error
	LookupBarcode(ctx context.Context, company uuid.UUID, code string) (BarcodeMatch, error)

	// ClaimOperation registers (company, type, id) with the payload hash.
	// claimed=false means it already exists with the same hash and prior
	// holds its stored result; a different hash is ErrIdempotencyConflict.
	ClaimOperation(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, hash string) (prior []byte, claimed bool, err error)
	CompleteOperation(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, result []byte) error

	// ApplyDelta changes one balance with a conditional update that never
	// leaves available, reserved or in-transit negative; otherwise it
	// returns ErrInsufficientStock and changes nothing.
	ApplyDelta(ctx context.Context, company uuid.UUID, d BalanceDelta) (BalanceState, error)
	AppendMovements(ctx context.Context, company uuid.UUID, opType string, opID uuid.UUID, ms []MovementRecord) error
	GetBalance(ctx context.Context, company, warehouse, sku uuid.UUID) (BalanceState, error)

	InsertReservation(ctx context.Context, r Reservation) error
	LockReservation(ctx context.Context, company, id uuid.UUID) (Reservation, error)
	SetReservationStatus(ctx context.Context, company, id uuid.UUID, status string) error
	// ExpireDueReservations marks up to limit active reservations whose
	// expiry passed as expired, skipping rows locked by other transactions.
	ExpireDueReservations(ctx context.Context, company uuid.UUID, now time.Time, limit int) ([]Reservation, error)

	InsertTransfer(ctx context.Context, t Transfer) error
	LockTransfer(ctx context.Context, company, id uuid.UUID) (Transfer, error)
	SetTransferStatus(ctx context.Context, company, id uuid.UUID, status string) error
}

// PlanDeltas merges deltas of the same balance and orders them by
// (warehouse, sku) so every transaction locks balances in the same order
// and concurrent operations cannot deadlock (ADR-006).
func PlanDeltas(in []BalanceDelta) []BalanceDelta {
	merged := map[BalanceKey]*BalanceDelta{}
	var keys []BalanceKey
	for _, d := range in {
		m, ok := merged[d.Key]
		if !ok {
			c := BalanceDelta{Key: d.Key}
			merged[d.Key] = &c
			m = &c
			keys = append(keys, d.Key)
		}
		m.Physical = m.Physical.Add(d.Physical)
		m.InTransit = m.InTransit.Add(d.InTransit)
		m.Reserved = m.Reserved.Add(d.Reserved)
	}
	sort.Slice(keys, func(i, j int) bool {
		if c := bytes.Compare(keys[i].WarehouseID[:], keys[j].WarehouseID[:]); c != 0 {
			return c < 0
		}
		return bytes.Compare(keys[i].SKUID[:], keys[j].SKUID[:]) < 0
	})
	out := make([]BalanceDelta, 0, len(keys))
	for _, k := range keys {
		out = append(out, *merged[k])
	}
	return out
}
