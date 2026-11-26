package pgstore_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/jobs"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
)

// The automatic sweep, as wired in bpserver (organization.ListCompanyIDs +
// application.Service), frees expired reservations of every company with
// no manual call, and records them as expired events.
func TestAutomaticSweepReleasesExpiredReservations(t *testing.T) {
	e := setup(t)
	s := e.sku("AUTO", domain.KindUnit, "und", 0)
	_, err := e.post(application.StockOpening, e.whA, "", line(s.ID, "10"))
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err := e.svc.Reserve(e.ctx, e.sc, application.ReserveCommand{OperationID: uuid.New(), ReservationID: uuid.New(),
			WarehouseID: e.whA, Line: line(s.ID, "2"), Purpose: "apartado", ExpiresAt: e.clock.Now().Add(time.Minute)})
		require.NoError(t, err)
	}
	require.Equal(t, "4", e.balance(e.whA, s.ID).Available().String())
	e.clock.Advance(2 * time.Minute)

	orgs := organization.NewStore(e.pool)
	sw := &jobs.ReservationSweeper{Expirer: e.svc, Companies: orgs.ListCompanyIDs, Batch: 2,
		Interval: 10 * time.Millisecond, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	require.Eventually(t, func() bool {
		return e.balance(e.whA, s.ID).Available().String() == "10"
	}, 5*time.Second, 20*time.Millisecond, "sweeper must release all expired reservations (batch smaller than total)")
	e.assertReservationsCoherent()
}
