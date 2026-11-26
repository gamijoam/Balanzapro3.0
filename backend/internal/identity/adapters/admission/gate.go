// Package admission bounds pre-authentication work before identity lookup.
// Its fixed-window budget and concurrency limit are local to one process.
package admission

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalid = errors.New("identity: invalid admission configuration")
	ErrBusy    = errors.New("identity: admission capacity exhausted")
	ErrRate    = errors.New("identity: admission frequency exhausted")
	ErrClock   = errors.New("identity: admission clock invalid")
)

// Gate has constant-sized state, independent of untrusted identifiers.
// Construct once per server and never copy after use.
type Gate struct {
	mu                          sync.Mutex
	capacity, max, active, used int
	window                      time.Duration
	clock                       func() time.Time
	start, last                 time.Time
}

func New(capacity, max int, window time.Duration, clock func() time.Time) (*Gate, error) {
	if capacity < 1 || capacity > 256 || max < 1 || max > 10000 || window < time.Millisecond || window > time.Minute || clock == nil {
		return nil, ErrInvalid
	}
	return &Gate{capacity: capacity, max: max, window: window, clock: clock}, nil
}

func (g *Gate) acquire() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active >= g.capacity {
		return ErrBusy
	}
	now := g.clock()
	if now.IsZero() || (!g.last.IsZero() && now.Before(g.last)) {
		return ErrClock
	}
	g.last = now
	if g.start.IsZero() || !now.Before(g.start.Add(g.window)) {
		g.start = now
		g.used = 0
	}
	if g.used >= g.max {
		return ErrRate
	}
	g.used++
	g.active++
	return nil
}

// Run does not queue requests for a work slot. A grant spends a frequency
// unit even on failure or cancellation, and retains its concurrent slot
// until the actual work returns.
func (g *Gate) Run(ctx context.Context, fn func() error) error {
	if g == nil || g.clock == nil || ctx == nil || fn == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.acquire(); err != nil {
		return err
	}
	defer func() { g.mu.Lock(); g.active--; g.mu.Unlock() }()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
