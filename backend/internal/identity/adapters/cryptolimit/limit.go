// Package cryptolimit bounds simultaneous expensive cryptographic work in
// one server process. Share a single limiter across credential verifiers.
package cryptolimit

import (
	"context"
	"errors"
)

var ErrBusy = errors.New("identity: verification capacity exhausted")
var ErrInvalid = errors.New("identity: invalid verification limiter")

// Limiter must be constructed once per server, not once per request.
// It cannot be copied after construction to obtain an independent budget.
type Limiter struct{ slots chan struct{} }

// New bounds capacity as a technical guardrail. Deployment sizing is
// deliberately not a default: measure memory and CPU on target hardware.
func New(capacity int) (*Limiter, error) {
	if capacity < 1 || capacity > 64 {
		return nil, ErrInvalid
	}
	return &Limiter{slots: make(chan struct{}, capacity)}, nil
}

// Run admits immediately or returns ErrBusy, with no unbounded queue.
// Cancellation never releases a slot before the callback actually returns.
func (l *Limiter) Run(ctx context.Context, fn func() error) error {
	if l == nil || l.slots == nil || ctx == nil || fn == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case l.slots <- struct{}{}:
		defer func() { <-l.slots }()
	default:
		return ErrBusy
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
