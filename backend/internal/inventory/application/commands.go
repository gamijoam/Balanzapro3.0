package application

import (
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
)

// Operation types (idempotency key = company + type + operation id).
const (
	OpStock              = "inventory.stock"
	OpTransferShip       = "inventory.transfer.ship"
	OpTransferReceive    = "inventory.transfer.receive"
	OpReserve            = "inventory.reservation.create"
	OpReservationRelease = "inventory.reservation.release"
	OpReservationConsume = "inventory.reservation.consume"
)

// StockKind is the purpose of a direct stock posting.
type StockKind string

const (
	StockOpening       StockKind = "opening"
	StockAdjustmentIn  StockKind = "adjustment_in"
	StockAdjustmentOut StockKind = "adjustment_out"
	StockIssue         StockKind = "issue"
)

// Line is a quantity of a SKU, in base units or in a presentation.
// Quantity is a decimal string (ADR-040).
type Line struct {
	SKUID          uuid.UUID  `json:"sku_id"`
	PresentationID *uuid.UUID `json:"presentation_id,omitempty"`
	Quantity       string     `json:"quantity"`
}

type RegisterSKUCommand struct {
	ID        uuid.UUID
	Code      string
	Name      string
	Kind      domain.QuantityKind
	BaseUnit  string
	Precision int32
}

type AddPresentationCommand struct {
	ID     uuid.UUID
	SKUID  uuid.UUID
	Name   string
	Factor string
}

type AddBarcodeCommand struct {
	Code           string
	SKUID          uuid.UUID
	PresentationID *uuid.UUID
}

// StockCommand posts entries or exits in one warehouse, all lines or none.
type StockCommand struct {
	OperationID uuid.UUID `json:"-"`
	Kind        StockKind `json:"kind"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Lines       []Line    `json:"lines"`
	Reason      string    `json:"reason,omitempty"`
}

type ShipTransferCommand struct {
	OperationID     uuid.UUID `json:"-"`
	TransferID      uuid.UUID `json:"transfer_id"`
	FromWarehouseID uuid.UUID `json:"from_warehouse_id"`
	ToWarehouseID   uuid.UUID `json:"to_warehouse_id"`
	Lines           []Line    `json:"lines"`
}

type ReceiveTransferCommand struct {
	OperationID uuid.UUID `json:"-"`
	TransferID  uuid.UUID `json:"transfer_id"`
}

type ReserveCommand struct {
	OperationID   uuid.UUID `json:"-"`
	ReservationID uuid.UUID `json:"reservation_id"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	Line          Line      `json:"line"`
	Purpose       string    `json:"purpose"`
	OwnerRef      string    `json:"owner_ref"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type ReservationCommand struct {
	OperationID   uuid.UUID `json:"-"`
	ReservationID uuid.UUID `json:"reservation_id"`
}

// OperationResult is stored durably with the operation and returned as-is on
// an identical retry (Replayed=true).
type OperationResult struct {
	OperationID   uuid.UUID        `json:"operation_id"`
	OperationType string           `json:"operation_type"`
	Movements     []MovementRecord `json:"movements"`
	ReservationID *uuid.UUID       `json:"reservation_id,omitempty"`
	TransferID    *uuid.UUID       `json:"transfer_id,omitempty"`
	Replayed      bool             `json:"-"`
}
