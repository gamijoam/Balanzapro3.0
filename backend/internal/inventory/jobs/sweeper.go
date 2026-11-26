// Package jobs holds background processes of the inventory module.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Expirer is the use case the sweeper drives (application.Service).
type Expirer interface {
	SweepExpiredReservations(ctx context.Context, companies []uuid.UUID, batch int) (int, error)
}

// ReservationSweeper releases expired reservations periodically.
type ReservationSweeper struct {
	Expirer   Expirer
	Companies func(ctx context.Context) ([]uuid.UUID, error)
	Interval  time.Duration
	Batch     int
	Log       *slog.Logger
}

// RunOnce sweeps every company once and returns how many reservations it
// released.
func (s *ReservationSweeper) RunOnce(ctx context.Context) (int, error) {
	if s.Batch <= 0 {
		return 0, errors.New("sweeper: batch must be positive")
	}
	companies, err := s.Companies(ctx)
	if err != nil {
		return 0, fmt.Errorf("sweeper: list companies: %w", err)
	}
	n, err := s.Expirer.SweepExpiredReservations(ctx, companies, s.Batch)
	if err != nil {
		return n, fmt.Errorf("sweeper: %w", err)
	}
	return n, nil
}

// Run sweeps once at start (reservations may have expired while the server
// was down) and then every Interval until ctx ends. Errors are logged and
// the next tick tries again, so a transient failure never stops it. A
// non-positive Interval is a configuration error: it is logged and Run
// returns.
func (s *ReservationSweeper) Run(ctx context.Context) {
	if s.Interval <= 0 {
		s.Log.Error("reservation sweeper not started: interval must be positive", "interval", s.Interval)
		return
	}
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		s.sweep(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *ReservationSweeper) sweep(ctx context.Context) {
	n, err := s.RunOnce(ctx)
	switch {
	case err != nil && ctx.Err() == nil:
		s.Log.Error("reservation sweep failed", "error", err, "released", n)
	case n > 0:
		s.Log.Info("reservation sweep", "released", n)
	}
}
