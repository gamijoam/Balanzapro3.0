package jobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/jobs"
)

type fakeExpirer struct {
	mu    sync.Mutex
	calls [][]uuid.UUID
	err   error
	n     int
}

func (f *fakeExpirer) SweepExpiredReservations(_ context.Context, companies []uuid.UUID, batch int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, companies)
	return f.n, f.err
}

func (f *fakeExpirer) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.calls) }

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestRunOncePassesEveryCompany(t *testing.T) {
	c1, c2 := uuid.New(), uuid.New()
	f := &fakeExpirer{n: 3}
	s := &jobs.ReservationSweeper{Expirer: f, Batch: 10, Log: quiet,
		Companies: func(context.Context) ([]uuid.UUID, error) { return []uuid.UUID{c1, c2}, nil }}
	n, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	require.Equal(t, 1, f.count())
	assert.Equal(t, []uuid.UUID{c1, c2}, f.calls[0])
}

func TestRunOnceReportsErrors(t *testing.T) {
	f := &fakeExpirer{err: errors.New("db down")}
	s := &jobs.ReservationSweeper{Expirer: f, Batch: 10, Log: quiet,
		Companies: func(context.Context) ([]uuid.UUID, error) { return []uuid.UUID{uuid.New()}, nil }}
	_, err := s.RunOnce(context.Background())
	assert.Error(t, err)

	s.Companies = func(context.Context) ([]uuid.UUID, error) { return nil, errors.New("list failed") }
	_, err = s.RunOnce(context.Background())
	assert.Error(t, err)
}

func TestRunTicksUntilCancelledAndSurvivesErrors(t *testing.T) {
	f := &fakeExpirer{err: errors.New("transient")}
	s := &jobs.ReservationSweeper{Expirer: f, Batch: 10, Interval: 5 * time.Millisecond, Log: quiet,
		Companies: func(context.Context) ([]uuid.UUID, error) { return []uuid.UUID{uuid.New()}, nil }}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { s.Run(ctx); close(stopped) }()
	require.Eventually(t, func() bool { return f.count() >= 3 }, 2*time.Second, time.Millisecond,
		"keeps sweeping after errors")
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	s := &jobs.ReservationSweeper{Expirer: &fakeExpirer{}, Log: quiet,
		Companies: func(context.Context) ([]uuid.UUID, error) { return nil, nil }}
	_, err := s.RunOnce(context.Background())
	assert.Error(t, err, "batch must be positive")
}
