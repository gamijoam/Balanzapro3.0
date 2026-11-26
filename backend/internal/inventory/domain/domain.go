// Package domain holds the inventory core rules (ADR-005, 016, 059–068):
// SKUs with one base unit, exact presentations, barcodes as text, movement
// types and the errors the application layer reports. It has no I/O.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

var (
	ErrInvalidArgument      = errors.New("inventory: invalid argument")
	ErrPrecision            = errors.New("inventory: quantity exceeds SKU precision")
	ErrNotFound             = errors.New("inventory: not found")
	ErrDuplicate            = errors.New("inventory: duplicate")
	ErrInsufficientStock    = errors.New("inventory: insufficient available stock")
	ErrIdempotencyConflict  = errors.New("inventory: operation id reused with a different payload")
	ErrReservationNotActive = errors.New("inventory: reservation is not active")
	ErrTransferNotShipped   = errors.New("inventory: transfer is not in shipped state")
)

// QuantityKind is the physical dimension of a SKU's base unit.
type QuantityKind string

const (
	KindUnit   QuantityKind = "unit"
	KindWeight QuantityKind = "weight"
	KindLength QuantityKind = "length"
	KindVolume QuantityKind = "volume"
)

func (k QuantityKind) valid() bool {
	switch k {
	case KindUnit, KindWeight, KindLength, KindVolume:
		return true
	}
	return false
}

// baseUnits is the initial catalog of base units per dimension (ADR-171,
// pending ratification of the concrete list). A SKU's stock is always kept
// in one of these; packagings such as "caja" or "docena" are presentations
// with an exact factor, never base units. Migration 00003 mirrors it.
var baseUnits = map[string]QuantityKind{
	"und": KindUnit,
	"kg":  KindWeight, "g": KindWeight,
	"m": KindLength, "cm": KindLength,
	"l": KindVolume, "ml": KindVolume,
}

// UnitKind returns the dimension of a base unit code, or "" if unknown.
func UnitKind(code string) QuantityKind { return baseUnits[strings.ToLower(strings.TrimSpace(code))] }

// SKU is the stock-keeping unit. All stock is held in its base unit.
type SKU struct {
	ID        uuid.UUID    `json:"id"`
	CompanyID uuid.UUID    `json:"company_id"`
	Code      string       `json:"code"`
	Name      string       `json:"name"`
	Kind      QuantityKind `json:"quantity_kind"`
	BaseUnit  string       `json:"base_unit"`
	Precision int32        `json:"precision"`
}

// NewSKU validates and builds a SKU.
func NewSKU(id, company uuid.UUID, code, name string, kind QuantityKind, baseUnit string, precision int32) (SKU, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	baseUnit = strings.ToLower(strings.TrimSpace(baseUnit))
	switch {
	case id == uuid.Nil || company == uuid.Nil:
		return SKU{}, fmt.Errorf("%w: id and company are required", ErrInvalidArgument)
	case code == "" || name == "" || baseUnit == "":
		return SKU{}, fmt.Errorf("%w: code, name and base unit are required", ErrInvalidArgument)
	case !kind.valid():
		return SKU{}, fmt.Errorf("%w: unknown quantity kind %q", ErrInvalidArgument, kind)
	case UnitKind(baseUnit) != kind:
		return SKU{}, fmt.Errorf("%w: base unit %q is not a %s unit", ErrInvalidArgument, baseUnit, kind)
	case precision < 0 || precision > qty.MaxScale:
		return SKU{}, fmt.Errorf("%w: precision must be between 0 and %d", ErrInvalidArgument, qty.MaxScale)
	}
	return SKU{ID: id, CompanyID: company, Code: code, Name: name, Kind: kind, BaseUnit: baseUnit, Precision: precision}, nil
}

// ValidateQuantity checks a positive base-unit quantity fits the SKU precision.
func (s SKU) ValidateQuantity(q qty.Q) error {
	if !q.IsPositive() {
		return fmt.Errorf("%w: quantity must be positive", ErrInvalidArgument)
	}
	if !q.Fits() {
		return fmt.Errorf("%w: quantity %s exceeds the storable range", ErrInvalidArgument, q)
	}
	if q.Scale() > s.Precision {
		return fmt.Errorf("%w: %s has %d decimals, SKU %s allows %d", ErrPrecision, q, q.Scale(), s.Code, s.Precision)
	}
	return nil
}

// Presentation is a sellable/purchasable packaging of a SKU with an exact
// factor of base units (ADR-060/061). Its factor never changes once used;
// movements keep a snapshot of it.
type Presentation struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
	SKUID     uuid.UUID `json:"sku_id"`
	Name      string    `json:"name"`
	Factor    qty.Q     `json:"factor"`
}

// NewPresentation validates and builds a presentation for sku.
func NewPresentation(id uuid.UUID, sku SKU, name string, factor qty.Q) (Presentation, error) {
	name = strings.TrimSpace(name)
	if id == uuid.Nil || name == "" {
		return Presentation{}, fmt.Errorf("%w: id and name are required", ErrInvalidArgument)
	}
	if err := sku.ValidateQuantity(factor); err != nil {
		return Presentation{}, err
	}
	return Presentation{ID: id, CompanyID: sku.CompanyID, SKUID: sku.ID, Name: name, Factor: factor}, nil
}

// ToBase converts n presentations into base units, exactly.
func (p Presentation) ToBase(n qty.Q) qty.Q { return n.Mul(p.Factor) }

// Breakdown expresses a base quantity as whole presentations plus remainder.
func Breakdown(base qty.Q, p Presentation) (whole, rem qty.Q) { return base.QuoRem(p.Factor) }

// MaxBarcodeLen bounds stored identifiers.
const MaxBarcodeLen = 64

// NormalizeBarcode trims surrounding spaces and keeps the code as text:
// leading zeros and letter case are significant.
func NormalizeBarcode(code string) (string, error) {
	c := strings.TrimSpace(code)
	if c == "" || len(c) > MaxBarcodeLen {
		return "", fmt.Errorf("%w: barcode must have 1..%d characters", ErrInvalidArgument, MaxBarcodeLen)
	}
	for _, r := range c {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", fmt.Errorf("%w: barcode contains spaces or control characters", ErrInvalidArgument)
		}
	}
	return c, nil
}

// MovementType classifies a kardex entry.
type MovementType string

const (
	MoveOpening        MovementType = "opening"
	MoveAdjustmentIn   MovementType = "adjustment_in"
	MoveAdjustmentOut  MovementType = "adjustment_out"
	MoveIssue          MovementType = "issue"
	MoveReservationOut MovementType = "reservation_consume"
	MoveTransferOut    MovementType = "transfer_out"
	MoveTransitIn      MovementType = "transfer_transit_in"
	MoveTransitOut     MovementType = "transfer_transit_out"
	MoveTransferIn     MovementType = "transfer_in"
)

// Bucket is the balance column a movement affects.
type Bucket string

const (
	BucketPhysical  Bucket = "physical"
	BucketInTransit Bucket = "in_transit"
)
