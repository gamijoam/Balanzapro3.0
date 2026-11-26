package pgstore_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
)

// assertReservationsCoherent: the reserved bucket equals the active
// reservations, every reservation has exactly one "created" event and one
// terminal event iff it is closed (matching its status), and the signed
// sum of events equals the reserved balance (ADR-016/068).
func (e *env) assertReservationsCoherent() {
	e.t.Helper()
	var bad int
	require.NoError(e.t, e.pool.QueryRow(e.ctx, `
		WITH active AS (
			SELECT warehouse_id, sku_id, SUM(quantity) q FROM inventory_reservations
			WHERE status = 'active' GROUP BY 1, 2),
		ev AS (
			SELECT warehouse_id, sku_id,
			       SUM(CASE WHEN event_type = 'created' THEN quantity ELSE -quantity END) q
			FROM inventory_reservation_events GROUP BY 1, 2)
		SELECT count(*) FROM inventory_balances b
		LEFT JOIN active a USING (warehouse_id, sku_id)
		LEFT JOIN ev USING (warehouse_id, sku_id)
		WHERE b.reserved <> COALESCE(a.q, 0) OR b.reserved <> COALESCE(ev.q, 0)`).Scan(&bad))
	assert.Zero(e.t, bad, "reserved balance, active reservations and events diverge")

	require.NoError(e.t, e.pool.QueryRow(e.ctx, `
		SELECT count(*) FROM inventory_reservations r
		WHERE (SELECT count(*) FROM inventory_reservation_events v
		       WHERE v.reservation_id = r.id AND v.event_type = 'created') <> 1
		   OR (SELECT count(*) FROM inventory_reservation_events v
		       WHERE v.reservation_id = r.id AND v.event_type <> 'created')
		      <> CASE WHEN r.status = 'active' THEN 0 ELSE 1 END
		   OR (r.status <> 'active' AND NOT EXISTS (
		       SELECT 1 FROM inventory_reservation_events v
		       WHERE v.reservation_id = r.id AND v.event_type = r.status))`).Scan(&bad))
	assert.Zero(e.t, bad, "reservation lifecycle events are incomplete")
}

func TestReservationHistoryIsTraceable(t *testing.T) {
	e := setup(t)
	s := e.sku("TRAZA", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "20"))
	require.NoError(t, err)

	newRes := func(q string, ttl time.Duration) (uuid.UUID, uuid.UUID) {
		id, op := uuid.New(), uuid.New()
		_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: op, ReservationID: id,
			WarehouseID: e.whA, Line: line(s.ID, q), Purpose: "apartado", OwnerRef: "cliente-sint-7", ExpiresAt: e.clock.Now().Add(ttl)})
		require.NoError(t, err)
		return id, op
	}
	released, createOp := newRes("2", time.Hour)
	consumed, _ := newRes("3", time.Hour)
	expired, _ := newRes("4", time.Minute)
	lateRelease, _ := newRes("1", time.Minute)

	relOp := uuid.New()
	_, err = e.svc.ReleaseReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: relOp, ReservationID: released})
	require.NoError(t, err)
	_, err = e.svc.ConsumeReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: consumed})
	require.NoError(t, err)
	e.clock.Advance(2 * time.Minute)
	_, err = e.svc.ReleaseReservation(e.ctx, e.sc, application.ReservationCommand{OperationID: uuid.New(), ReservationID: lateRelease})
	require.NoError(t, err)
	n, err := e.svc.ExpireReservations(e.ctx, e.sc, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	hist, err := e.svc.ReservationHistory(e.ctx, e.sc, released)
	require.NoError(t, err)
	require.Len(t, hist, 2)
	assert.Equal(t, application.ReservationEventCreated, hist[0].Type)
	assert.Equal(t, createOp, hist[0].OperationID)
	assert.Equal(t, application.OpReserve, hist[0].OperationType)
	assert.Equal(t, "2", hist[0].Quantity.String())
	assert.Equal(t, "2", hist[0].ReservedAfter.String(), "first reservation of the balance")
	assert.Equal(t, application.ReservationEventReleased, hist[1].Type)
	assert.Equal(t, relOp, hist[1].OperationID)

	last := func(id uuid.UUID) application.ReservationEvent {
		h, err := e.svc.ReservationHistory(e.ctx, e.sc, id)
		require.NoError(t, err)
		require.Len(t, h, 2)
		return h[1]
	}
	assert.Equal(t, application.ReservationEventConsumed, last(consumed).Type)
	assert.Equal(t, application.ReservationEventExpired, last(lateRelease).Type, "late release closes as expired")
	ex := last(expired)
	assert.Equal(t, application.ReservationEventExpired, ex.Type)
	assert.Equal(t, application.OpReservationExpire, ex.OperationType, "sweep batches are identified")
	assert.NotEqual(t, uuid.Nil, ex.OperationID)

	_, err = e.svc.ReservationHistory(e.ctx, e.other, released)
	assert.ErrorIs(t, err, domain.ErrNotFound, "history is company-scoped")

	_, err = e.pool.Exec(e.ctx, `UPDATE inventory_reservation_events SET quantity = 0`)
	assert.Error(t, err, "reservation events are append-only")
	_, err = e.pool.Exec(e.ctx, `DELETE FROM inventory_reservation_events`)
	assert.Error(t, err)

	e.assertReservationsCoherent()
	e.assertKardexCoherent()
}

func TestReservationEventsCoherentUnderConcurrency(t *testing.T) {
	e := setup(t)
	s := e.sku("TRAZA2", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "30"))
	require.NoError(t, err)
	done := make(chan struct{})
	for i := 0; i < 30; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			id := uuid.New()
			_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: uuid.New(), ReservationID: id,
				WarehouseID: e.whA, Line: line(s.ID, "1"), Purpose: "pedido", ExpiresAt: e.clock.Now().Add(time.Hour)})
			if !assert.NoError(t, err) {
				return
			}
			cmd := application.ReservationCommand{OperationID: uuid.New(), ReservationID: id}
			if i%2 == 0 {
				_, err = e.svc.ConsumeReservation(e.ctx, e.sc, cmd)
			} else {
				_, err = e.svc.ReleaseReservation(e.ctx, e.sc, cmd)
			}
			assert.NoError(t, err)
		}(i)
	}
	for i := 0; i < 30; i++ {
		<-done
	}
	assert.Equal(t, "15", e.balance(e.whA, s.ID).Physical.String())
	e.assertReservationsCoherent()
	e.assertKardexCoherent()
}
