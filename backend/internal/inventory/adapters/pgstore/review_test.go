package pgstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
)

// Regression tests for the findings of the review of PR #1.

// Finding 1: an identical retry after the expiry passed must replay.
func TestReserveRetryAfterExpiryReplaysOriginalResult(t *testing.T) {
	e := setup(t)
	s := e.sku("R1", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "5"))
	require.NoError(t, err)
	cmd := application.ReserveCommand{OperationID: uuid.New(), ReservationID: uuid.New(), WarehouseID: e.whA,
		Line: line(s.ID, "2"), Purpose: "apartado", ExpiresAt: e.clock.Now().Add(30 * time.Second)}
	first, err := e.svc.Reserve(e.ctx, e.sc, cmd)
	require.NoError(t, err)

	e.clock.Advance(40 * time.Second)
	again, err := e.svc.Reserve(e.ctx, e.sc, cmd)
	require.NoError(t, err)
	assert.True(t, again.Replayed)
	assert.Equal(t, first.ReservationID, again.ReservationID)
}

// Finding 2: overflow is a validation error, never an internal one.
func TestQuantityOverflowIsInvalidArgument(t *testing.T) {
	e := setup(t)
	s := e.sku("BIG", domain.KindUnit, "und", 0)
	caja, err := e.svc.AddPresentation(e.ctx, e.sc, application.AddPresentationCommand{ID: uuid.New(), SKUID: s.ID, Name: "Caja", Factor: "24"})
	require.NoError(t, err)

	_, err = e.post(application.StockOpening, e.whA, "", application.Line{SKUID: s.ID, PresentationID: &caja.ID, Quantity: "100000000000000000"})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = e.post(application.StockOpening, e.whA, "", line(s.ID, "1000000000000000000"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)

	_, err = e.post(application.StockOpening, e.whA, "", line(s.ID, "999999999999999999"))
	require.NoError(t, err)
	_, err = e.post(application.StockOpening, e.whA, "", line(s.ID, "1"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "balance overflow must be reported as invalid")
	assert.Equal(t, "999999999999999999", e.balance(e.whA, s.ID).Physical.String())
}

// Finding 3: equal quantities written differently are the same request.
func TestIdempotencyHashUsesCanonicalQuantities(t *testing.T) {
	e := setup(t)
	kg := e.sku("KG", domain.KindWeight, "kg", 3)
	_, err := e.post(application.StockOpening, e.whA, "", line(kg.ID, "10"))
	require.NoError(t, err)
	opID := uuid.New()
	_, err = e.svc.PostStock(e.ctx, e.sc, application.StockCommand{OperationID: opID, Kind: application.StockIssue,
		WarehouseID: e.whA, Lines: []application.Line{line(kg.ID, "2.50")}})
	require.NoError(t, err)
	r, err := e.svc.PostStock(e.ctx, e.sc, application.StockCommand{OperationID: opID, Kind: application.StockIssue,
		WarehouseID: e.whA, Lines: []application.Line{line(kg.ID, "2.5")}})
	require.NoError(t, err)
	assert.True(t, r.Replayed)
	assert.Equal(t, "7.5", e.balance(e.whA, kg.ID).Physical.String())
}

// Finding 4: an expired reservation that was not swept can be released,
// and a sweep over all companies frees the rest.
func TestExpiredReservationCanBeReleasedAndSweptForAllCompanies(t *testing.T) {
	e := setup(t)
	s := e.sku("R4", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)
	reserve := func() uuid.UUID {
		id := uuid.New()
		_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: uuid.New(), ReservationID: id,
			WarehouseID: e.whA, Line: line(s.ID, "5"), Purpose: "apartado", ExpiresAt: e.clock.Now().Add(time.Minute)})
		require.NoError(t, err)
		return id
	}
	r1 := reserve()
	e.clock.Advance(2 * time.Minute)
	_, err = e.svc.ReleaseReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: r1})
	require.NoError(t, err, "releasing an expired, unswept reservation frees its stock")
	assert.Equal(t, "10", e.balance(e.whA, s.ID).Available().String())
	_, err = e.svc.ReleaseReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: r1})
	assert.ErrorIs(t, err, domain.ErrReservationNotActive)

	reserve()
	e.clock.Advance(2 * time.Minute)
	n, err := e.svc.SweepExpiredReservations(e.ctx, []uuid.UUID{e.sc.CompanyID, e.other.CompanyID}, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "10", e.balance(e.whA, s.ID).Available().String())
	e.assertReservationsCoherent()
	e.assertKardexCoherent()
}

// Finding 5: Retries counts only retries actually performed.
func TestRetriesCountOnlyPerformedRetries(t *testing.T) {
	e := setup(t)
	attempts := 0
	err := e.store.InTx(context.Background(), func(application.Tx) error {
		attempts++
		return &pgconn.PgError{Code: "40P01"}
	})
	require.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, int64(2), e.store.Retries())
}

// Finding 6: SetTransferStatus fails instead of silently doing nothing.
func TestSetTransferStatusRequiresShippedTransfer(t *testing.T) {
	e := setup(t)
	err := e.store.InTx(e.ctx, func(tx application.Tx) error {
		return tx.SetTransferStatus(e.ctx, e.sc.CompanyID, uuid.New(), application.TransferReceived)
	})
	assert.ErrorIs(t, err, domain.ErrTransferNotShipped)
}

// Manual mutation M10: a received transfer cannot be marked again.
func TestSetTransferStatusRejectsAlreadyReceivedTransfer(t *testing.T) {
	e := setup(t)
	s := e.sku("M10", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "3"))
	require.NoError(t, err)
	tr := uuid.New()
	_, err = e.svc.ShipTransfer(e.ctx, e.sc, application.ShipTransferCommand{OperationID: uuid.New(), TransferID: tr,
		FromWarehouseID: e.whA, ToWarehouseID: e.whB, Lines: []application.Line{line(s.ID, "1")}})
	require.NoError(t, err)
	_, err = e.svc.ReceiveTransfer(e.ctx, e.sc, application.ReceiveTransferCommand{OperationID: uuid.New(), TransferID: tr})
	require.NoError(t, err)
	err = e.store.InTx(e.ctx, func(tx application.Tx) error {
		return tx.SetTransferStatus(e.ctx, e.sc.CompanyID, tr, application.TransferReceived)
	})
	assert.ErrorIs(t, err, domain.ErrTransferNotShipped)
}

// Manual mutation M11: catalog writes from another company never reach a
// SKU of this company, even where no foreign key would stop them.
func TestCrossCompanyCatalogWritesAreNotFound(t *testing.T) {
	e := setup(t)
	s := e.sku("M11", domain.KindUnit, "und", 0)
	_, err := e.svc.AddPresentation(e.ctx, e.other, application.AddPresentationCommand{ID: uuid.New(), SKUID: s.ID, Name: "Caja", Factor: "12"})
	assert.ErrorIs(t, err, domain.ErrNotFound)
	err = e.svc.AddBarcode(e.ctx, e.other, application.AddBarcodeCommand{Code: "M11-X", SKUID: s.ID})
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = e.svc.Reserve(e.ctx, e.other, application.ReserveCommand{OperationID: uuid.New(), ReservationID: uuid.New(),
		WarehouseID: e.otherW, Line: line(s.ID, "1"), Purpose: "x", ExpiresAt: e.clock.Now().Add(time.Hour)})
	assert.ErrorIs(t, err, domain.ErrNotFound)
	var n int
	require.NoError(t, e.pool.QueryRow(e.ctx, `SELECT count(*) FROM inventory_presentations WHERE sku_id = $1`, s.ID).Scan(&n))
	assert.Zero(t, n, "no presentation may be created on another company's SKU")
}
