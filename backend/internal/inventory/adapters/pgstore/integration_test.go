package pgstore_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// env is a synthetic company with two warehouses (fixtures, never real data).
type env struct {
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	store  *pgstore.Store
	svc    *application.Service
	clock  *fakeClock
	sc     application.Scope
	whA    uuid.UUID
	whB    uuid.UUID
	other  application.Scope
	otherW uuid.UUID
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := pgtest.New(t)
	ctx := context.Background()
	org := organization.NewStore(pool)

	mk := func(name string) (uuid.UUID, uuid.UUID) {
		o, err := org.CreateOrganization(ctx, "Org "+name)
		require.NoError(t, err)
		c, err := org.CreateCompany(ctx, o, "Empresa "+name)
		require.NoError(t, err)
		b, err := org.CreateBranch(ctx, c, "Sucursal "+name)
		require.NoError(t, err)
		w, err := org.CreateWarehouse(ctx, c, b, "Almacén "+name)
		require.NoError(t, err)
		return c, w
	}
	company, whA := mk("A")
	other, otherW := mk("Z")
	var branch uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT branch_id FROM warehouses WHERE id=$1`, whA).Scan(&branch))
	whB, err := org.CreateWarehouse(ctx, company, branch, "Almacén B")
	require.NoError(t, err)

	clock := &fakeClock{now: time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)}
	store := pgstore.New(pool)
	return &env{
		t: t, ctx: ctx, pool: pool, store: store,
		svc:   application.NewService(store, clock.Now),
		clock: clock,
		sc:    application.Scope{CompanyID: company}, whA: whA, whB: whB,
		other: application.Scope{CompanyID: other}, otherW: otherW,
	}
}

func (e *env) sku(code string, kind domain.QuantityKind, unit string, precision int32) domain.SKU {
	e.t.Helper()
	s, err := e.svc.RegisterSKU(e.ctx, e.sc, application.RegisterSKUCommand{
		ID: uuid.New(), Code: code, Name: code, Kind: kind, BaseUnit: unit, Precision: precision,
	})
	require.NoError(e.t, err)
	return s
}

func (e *env) post(kind application.StockKind, wh uuid.UUID, reason string, lines ...application.Line) (application.OperationResult, error) {
	return e.svc.PostStock(e.ctx, e.sc, application.StockCommand{
		OperationID: uuid.New(), Kind: kind, WarehouseID: wh, Lines: lines, Reason: reason,
	})
}

func line(sku uuid.UUID, q string) application.Line { return application.Line{SKUID: sku, Quantity: q} }

func (e *env) balance(wh, sku uuid.UUID) application.BalanceState {
	e.t.Helper()
	b, err := e.svc.Balance(e.ctx, e.sc, wh, sku)
	require.NoError(e.t, err)
	return b
}

// assertKardexCoherent: for every balance, the signed sum of kardex
// movements per bucket equals the stored balance (ADR-016).
func (e *env) assertKardexCoherent() {
	e.t.Helper()
	var mismatches int
	require.NoError(e.t, e.pool.QueryRow(e.ctx, `
		WITH k AS (
			SELECT warehouse_id, sku_id,
			       COALESCE(SUM(direction * quantity) FILTER (WHERE bucket = 'physical'), 0) AS physical,
			       COALESCE(SUM(direction * quantity) FILTER (WHERE bucket = 'in_transit'), 0) AS in_transit
			FROM inventory_movements GROUP BY warehouse_id, sku_id)
		SELECT count(*) FROM inventory_balances b
		LEFT JOIN k USING (warehouse_id, sku_id)
		WHERE b.physical <> COALESCE(k.physical, 0) OR b.in_transit <> COALESCE(k.in_transit, 0)`).
		Scan(&mismatches))
	assert.Zero(e.t, mismatches, "kardex and balances diverge")
}

func (e *env) movementCount() int {
	var n int
	require.NoError(e.t, e.pool.QueryRow(e.ctx, `SELECT count(*) FROM inventory_movements`).Scan(&n))
	return n
}

// BP-006: exact decimal round trip Go → PostgreSQL → Go.
func TestDecimalRoundTripThroughPostgres(t *testing.T) {
	e := setup(t)
	for _, v := range []string{"0.000001", "0.1", "0.333333", "123456789012345678.123456", "1.5", "120"} {
		var out string
		require.NoError(t, e.pool.QueryRow(e.ctx, `SELECT ($1::numeric(24,6))::text`, v).Scan(&out))
		got, err := qty.Parse(out)
		require.NoError(t, err)
		assert.Equal(t, qty.MustParse(v).String(), got.String())
	}
	kg := e.sku("QUESO", domain.KindWeight, "kg", 6)
	_, err := e.post(application.StockOpening, e.whA, "", line(kg.ID, "0.1"), line(kg.ID, "0.2"))
	require.NoError(t, err)
	assert.Equal(t, "0.3", e.balance(e.whA, kg.ID).Physical.String())
}

// INV-01: dos (o veinte) cajas por la última unidad → una sola confirmación.
func TestINV01LastUnitOnlyOneConfirmation(t *testing.T) {
	e := setup(t)
	s := e.sku("ULTIMA", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "1"))
	require.NoError(t, err)

	const terminals = 20
	var ok, insufficient atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < terminals; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := e.post(application.StockIssue, e.whA, "", line(s.ID, "1"))
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, domain.ErrInsufficientStock):
				insufficient.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), ok.Load())
	assert.Equal(t, int32(terminals-1), insufficient.Load())
	assert.True(t, e.balance(e.whA, s.ID).Physical.IsZero())
	e.assertKardexCoherent()
}

func TestINV01ManyUnitsNeverOversold(t *testing.T) {
	e := setup(t)
	s := e.sku("ARROZ", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "50"))
	require.NoError(t, err)

	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.post(application.StockIssue, e.whA, "", line(s.ID, "1")); err == nil {
				ok.Add(1)
			} else if !errors.Is(err, domain.ErrInsufficientStock) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(50), ok.Load())
	assert.True(t, e.balance(e.whA, s.ID).Physical.IsZero())
	e.assertKardexCoherent()
}

// SALE-01 / ADR-006: reintento con el mismo ID devuelve el resultado original.
func TestIdempotentReplayAndConflict(t *testing.T) {
	e := setup(t)
	s := e.sku("ACEITE", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)

	cmd := application.StockCommand{OperationID: uuid.New(), Kind: application.StockIssue, WarehouseID: e.whA,
		Lines: []application.Line{line(s.ID, "3")}}
	first, err := e.svc.PostStock(e.ctx, e.sc, cmd)
	require.NoError(t, err)
	assert.False(t, first.Replayed)
	before := e.movementCount()

	second, err := e.svc.PostStock(e.ctx, e.sc, cmd)
	require.NoError(t, err)
	assert.True(t, second.Replayed)
	assert.Equal(t, first.Movements, second.Movements)
	assert.Equal(t, before, e.movementCount(), "a replay must not add movements")
	assert.Equal(t, "7", e.balance(e.whA, s.ID).Physical.String())

	changed := cmd
	changed.Lines = []application.Line{line(s.ID, "4")}
	_, err = e.svc.PostStock(e.ctx, e.sc, changed)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	assert.Equal(t, "7", e.balance(e.whA, s.ID).Physical.String())
}

func TestIdempotentConcurrentSameOperation(t *testing.T) {
	e := setup(t)
	s := e.sku("CAFE", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)
	cmd := application.StockCommand{OperationID: uuid.New(), Kind: application.StockIssue, WarehouseID: e.whA,
		Lines: []application.Line{line(s.ID, "2")}}

	var replays atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := e.svc.PostStock(e.ctx, e.sc, cmd)
			if !assert.NoError(t, err) {
				return
			}
			if r.Replayed {
				replays.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(9), replays.Load())
	assert.Equal(t, "8", e.balance(e.whA, s.ID).Physical.String())
	e.assertKardexCoherent()
}

// Atomicidad: si una línea no alcanza, ninguna se aplica.
func TestMultiLineOperationIsAllOrNothing(t *testing.T) {
	e := setup(t)
	a := e.sku("A", domain.KindUnit, "und", 0)
	b := e.sku("B", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(a.ID, "5"), line(b.ID, "1"))
	require.NoError(t, err)
	before := e.movementCount()

	_, err = e.post(application.StockIssue, e.whA, "", line(a.ID, "2"), line(b.ID, "2"))
	require.ErrorIs(t, err, domain.ErrInsufficientStock)
	assert.Equal(t, "5", e.balance(e.whA, a.ID).Physical.String())
	assert.Equal(t, "1", e.balance(e.whA, b.ID).Physical.String())
	assert.Equal(t, before, e.movementCount())
}

// Locks en orden estable: líneas en orden opuesto no producen deadlocks.
func TestOppositeLineOrderDoesNotDeadlock(t *testing.T) {
	e := setup(t)
	a := e.sku("X1", domain.KindUnit, "und", 0)
	b := e.sku("X2", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(a.ID, "1000"), line(b.ID, "1000"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l1, l2 := line(a.ID, "1"), line(b.ID, "1")
			if i%2 == 1 {
				l1, l2 = l2, l1
			}
			_, err := e.post(application.StockIssue, e.whA, "", l1, l2)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
	assert.Zero(t, e.store.Retries(), "ordered locking should need no deadlock retries")
	assert.Equal(t, "960", e.balance(e.whA, a.ID).Physical.String())
	assert.Equal(t, "960", e.balance(e.whA, b.ID).Physical.String())
	e.assertKardexCoherent()
}

// INV-03: Caja24, paquete6, unidad y fracción descuentan un único stock.
func TestINV03PresentationsAndBarcodes(t *testing.T) {
	e := setup(t)
	s := e.sku("REFRESCO", domain.KindUnit, "und", 0)
	caja, err := e.svc.AddPresentation(e.ctx, e.sc, application.AddPresentationCommand{ID: uuid.New(), SKUID: s.ID, Name: "Caja x24", Factor: "24"})
	require.NoError(t, err)
	paq, err := e.svc.AddPresentation(e.ctx, e.sc, application.AddPresentationCommand{ID: uuid.New(), SKUID: s.ID, Name: "Paquete x6", Factor: "6"})
	require.NoError(t, err)

	require.NoError(t, e.svc.AddBarcode(e.ctx, e.sc, application.AddBarcodeCommand{Code: "07591234000017", SKUID: s.ID, PresentationID: &caja.ID}))
	require.NoError(t, e.svc.AddBarcode(e.ctx, e.sc, application.AddBarcodeCommand{Code: "7591234000010", SKUID: s.ID}))
	err = e.svc.AddBarcode(e.ctx, e.sc, application.AddBarcodeCommand{Code: "7591234000010", SKUID: s.ID, PresentationID: &paq.ID})
	assert.ErrorIs(t, err, domain.ErrDuplicate, "a code identifies exactly one SKU/presentation per company")

	m, err := e.svc.LookupBarcode(e.ctx, e.sc, " 07591234000017 ")
	require.NoError(t, err)
	assert.Equal(t, s.ID, m.SKUID)
	require.NotNil(t, m.PresentationID)
	assert.Equal(t, caja.ID, *m.PresentationID)

	_, err = e.post(application.StockOpening, e.whA, "", application.Line{SKUID: s.ID, PresentationID: &caja.ID, Quantity: "5"})
	require.NoError(t, err)
	assert.Equal(t, "120", e.balance(e.whA, s.ID).Physical.String(), "5 cajas = 120 unidades, no se suman stocks")

	res, err := e.post(application.StockIssue, e.whA, "",
		application.Line{SKUID: s.ID, PresentationID: &paq.ID, Quantity: "2"},
		line(s.ID, "3"),
		application.Line{SKUID: s.ID, PresentationID: &caja.ID, Quantity: "0.5"})
	require.NoError(t, err)
	assert.Equal(t, "93", e.balance(e.whA, s.ID).Physical.String(), "120 - 12 - 3 - 12")
	require.Len(t, res.Movements, 3)
	require.NotNil(t, res.Movements[0].PresentationFactor)
	assert.Equal(t, "6", res.Movements[0].PresentationFactor.String(), "factor snapshot")
	assert.Equal(t, "12", res.Movements[0].Quantity.String())

	_, err = e.post(application.StockIssue, e.whA, "", line(s.ID, "0.5"))
	assert.ErrorIs(t, err, domain.ErrPrecision, "half a unit of a unit SKU is not representable")

	_, err = e.pool.Exec(e.ctx, `UPDATE inventory_presentations SET factor = 12 WHERE id = $1`, caja.ID)
	assert.Error(t, err, "presentation factor is immutable")
	e.assertKardexCoherent()
}

func TestWeightSKUFractions(t *testing.T) {
	e := setup(t)
	kg := e.sku("JAMON", domain.KindWeight, "kg", 3)
	_, err := e.post(application.StockOpening, e.whA, "", line(kg.ID, "2.5"))
	require.NoError(t, err)
	_, err = e.post(application.StockIssue, e.whA, "", line(kg.ID, "0.250"))
	require.NoError(t, err)
	assert.Equal(t, "2.25", e.balance(e.whA, kg.ID).Physical.String())
	_, err = e.post(application.StockIssue, e.whA, "", line(kg.ID, "0.0001"))
	assert.ErrorIs(t, err, domain.ErrPrecision)
	_, err = e.post(application.StockIssue, e.whA, "", line(kg.ID, "1e-1"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestAdjustmentsRequireReason(t *testing.T) {
	e := setup(t)
	s := e.sku("JABON", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockAdjustmentIn, e.whA, "", line(s.ID, "3"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	res, err := e.post(application.StockAdjustmentIn, e.whA, "Conteo físico 26/11", line(s.ID, "3"))
	require.NoError(t, err)
	assert.Equal(t, "Conteo físico 26/11", res.Movements[0].Reason)
	_, err = e.post(application.StockAdjustmentOut, e.whA, "Merma", line(s.ID, "1"))
	require.NoError(t, err)
	assert.Equal(t, "2", e.balance(e.whA, s.ID).Physical.String())
}

// ADR-068: reservas con expiración; liberación o consumo únicos.
func TestReservationsLifecycle(t *testing.T) {
	e := setup(t)
	s := e.sku("TV", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)

	reserve := func(q string, ttl time.Duration) uuid.UUID {
		id := uuid.New()
		_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{
			OperationID: uuid.New(), ReservationID: id, WarehouseID: e.whA, Line: line(s.ID, q),
			Purpose: "apartado", OwnerRef: "cliente-sintetico-1", ExpiresAt: e.clock.Now().Add(ttl),
		})
		require.NoError(t, err)
		return id
	}

	r1 := reserve("7", time.Hour)
	b := e.balance(e.whA, s.ID)
	assert.Equal(t, "10", b.Physical.String())
	assert.Equal(t, "3", b.Available().String())

	_, err = e.post(application.StockIssue, e.whA, "", line(s.ID, "4"))
	assert.ErrorIs(t, err, domain.ErrInsufficientStock, "reserved stock is not available")
	_, err = e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: uuid.New(), ReservationID: uuid.New(),
		WarehouseID: e.whA, Line: line(s.ID, "4"), Purpose: "x", ExpiresAt: e.clock.Now().Add(time.Hour)})
	assert.ErrorIs(t, err, domain.ErrInsufficientStock)

	res, err := e.svc.ConsumeReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: r1})
	require.NoError(t, err)
	require.Len(t, res.Movements, 1)
	assert.Equal(t, domain.MoveReservationOut, res.Movements[0].Type)
	b = e.balance(e.whA, s.ID)
	assert.Equal(t, "3", b.Physical.String())
	assert.True(t, b.Reserved.IsZero())

	_, err = e.svc.ReleaseReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: r1})
	assert.ErrorIs(t, err, domain.ErrReservationNotActive)

	r2 := reserve("2", time.Minute)
	e.clock.Advance(2 * time.Minute)
	_, err = e.svc.ConsumeReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: r2})
	assert.ErrorIs(t, err, domain.ErrReservationNotActive, "an expired reservation cannot be consumed")
	n, err := e.svc.ExpireReservations(e.ctx, e.sc, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = e.svc.ExpireReservations(e.ctx, e.sc, 100)
	require.NoError(t, err)
	assert.Zero(t, n, "expiry releases only once")
	assert.Equal(t, "3", e.balance(e.whA, s.ID).Available().String())
	e.assertKardexCoherent()
}

func TestReservationConsumeAndReleaseRaceHasOneWinner(t *testing.T) {
	e := setup(t)
	s := e.sku("CEL", domain.KindUnit, "und", 0)
	const rounds = 10
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)
	var consumed int64
	for i := 0; i < rounds; i++ {
		id := uuid.New()
		_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: uuid.New(), ReservationID: id,
			WarehouseID: e.whA, Line: line(s.ID, "1"), Purpose: "pedido", ExpiresAt: e.clock.Now().Add(time.Hour)})
		require.NoError(t, err)

		var wins atomic.Int32
		var wg sync.WaitGroup
		for _, f := range []func(context.Context, application.Scope, application.ReservationCommand) (application.OperationResult, error){
			e.svc.ConsumeReservation, e.svc.ReleaseReservation,
		} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if r, err := f(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: id}); err == nil {
					wins.Add(1)
					if len(r.Movements) == 1 {
						atomic.AddInt64(&consumed, 1)
					}
				} else {
					assert.ErrorIs(t, err, domain.ErrReservationNotActive)
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, int32(1), wins.Load())
	}
	b := e.balance(e.whA, s.ID)
	assert.True(t, b.Reserved.IsZero())
	assert.Equal(t, qty.FromInt(rounds-consumed), b.Physical, "only consumptions reduce physical stock")
	e.assertKardexCoherent()
}

// INV-07: transferencia conserva unidades; solo recibir confirma destino.
func TestINV07TransferConservesUnits(t *testing.T) {
	e := setup(t)
	s := e.sku("HARINA", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)

	trID := uuid.New()
	ship := application.ShipTransferCommand{OperationID: uuid.New(), TransferID: trID,
		FromWarehouseID: e.whA, ToWarehouseID: e.whB, Lines: []application.Line{line(s.ID, "4")}}
	_, err = e.svc.ShipTransfer(e.ctx, e.sc, ship)
	require.NoError(t, err)

	a, b := e.balance(e.whA, s.ID), e.balance(e.whB, s.ID)
	assert.Equal(t, "6", a.Physical.String())
	assert.True(t, b.Physical.IsZero(), "destination is not confirmed until received")
	assert.Equal(t, "4", b.InTransit.String())
	assert.Equal(t, "10", a.Physical.Add(b.Physical).Add(b.InTransit).String())

	_, err = e.svc.ReceiveTransfer(e.ctx, e.sc, application.ReceiveTransferCommand{OperationID: uuid.New(), TransferID: trID})
	require.NoError(t, err)
	b = e.balance(e.whB, s.ID)
	assert.Equal(t, "4", b.Physical.String())
	assert.True(t, b.InTransit.IsZero())

	_, err = e.svc.ReceiveTransfer(e.ctx, e.sc, application.ReceiveTransferCommand{OperationID: uuid.New(), TransferID: trID})
	assert.ErrorIs(t, err, domain.ErrTransferNotShipped, "a transfer is received once")

	same := ship
	same.OperationID = uuid.New()
	same.ToWarehouseID = e.whA
	_, err = e.svc.ShipTransfer(e.ctx, e.sc, same)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)

	tooMuch := application.ShipTransferCommand{OperationID: uuid.New(), TransferID: uuid.New(),
		FromWarehouseID: e.whA, ToWarehouseID: e.whB, Lines: []application.Line{line(s.ID, "7")}}
	_, err = e.svc.ShipTransfer(e.ctx, e.sc, tooMuch)
	assert.ErrorIs(t, err, domain.ErrInsufficientStock)
	e.assertKardexCoherent()
}

// SEC-01 (núcleo): una empresa no puede leer ni mover datos de otra.
func TestCompanyIsolation(t *testing.T) {
	e := setup(t)
	s := e.sku("PRIVADO", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "5"))
	require.NoError(t, err)
	require.NoError(t, e.svc.AddBarcode(e.ctx, e.sc, application.AddBarcodeCommand{Code: "111", SKUID: s.ID}))

	_, err = e.svc.PostStock(e.ctx, e.other, application.StockCommand{OperationID: uuid.New(),
		Kind: application.StockIssue, WarehouseID: e.whA, Lines: []application.Line{line(s.ID, "1")}})
	assert.ErrorIs(t, err, domain.ErrNotFound, "foreign warehouse")
	_, err = e.svc.PostStock(e.ctx, e.other, application.StockCommand{OperationID: uuid.New(),
		Kind: application.StockIssue, WarehouseID: e.otherW, Lines: []application.Line{line(s.ID, "1")}})
	assert.ErrorIs(t, err, domain.ErrNotFound, "foreign SKU")
	_, err = e.svc.LookupBarcode(e.ctx, e.other, "111")
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = e.svc.Balance(e.ctx, e.other, e.whA, s.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// The same barcode may exist in another company.
	s2, err := e.svc.RegisterSKU(e.ctx, e.other, application.RegisterSKUCommand{ID: uuid.New(), Code: "PRIVADO",
		Name: "Otro", Kind: domain.KindUnit, BaseUnit: "und"})
	require.NoError(t, err)
	require.NoError(t, e.svc.AddBarcode(e.ctx, e.other, application.AddBarcodeCommand{Code: "111", SKUID: s2.ID}))
	assert.Equal(t, "5", e.balance(e.whA, s.ID).Physical.String())
}

// ADR-016: el kardex nunca se edita.
func TestKardexIsAppendOnly(t *testing.T) {
	e := setup(t)
	s := e.sku("FIJO", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "5"))
	require.NoError(t, err)
	_, err = e.pool.Exec(e.ctx, `UPDATE inventory_movements SET quantity = 50`)
	assert.Error(t, err)
	_, err = e.pool.Exec(e.ctx, `DELETE FROM inventory_movements`)
	assert.Error(t, err)
	_, err = e.pool.Exec(e.ctx, `TRUNCATE inventory_movements CASCADE`)
	assert.Error(t, err)
	assert.Equal(t, 1, e.movementCount())
}

func TestDuplicateSKUCodeRejected(t *testing.T) {
	e := setup(t)
	e.sku("UNICO", domain.KindUnit, "und", 0)
	_, err := e.svc.RegisterSKU(e.ctx, e.sc, application.RegisterSKUCommand{ID: uuid.New(), Code: "UNICO",
		Name: "x", Kind: domain.KindUnit, BaseUnit: "und"})
	assert.ErrorIs(t, err, domain.ErrDuplicate)
}
