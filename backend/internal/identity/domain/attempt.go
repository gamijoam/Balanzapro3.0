package domain

import "time"

// AttemptPolicy describes a budget for expensive credential checks.
// Thresholds are supplied by trusted server configuration; this primitive
// does not choose production defaults. Each granted reservation counts,
// even if verification later succeeds, fails or is abandoned.
type AttemptPolicy struct {
	MaxAttempts int
	Window      time.Duration
	Cooldown    time.Duration
}

// AttemptState is persisted by the future atomic storage adapter.
// The zero value represents a new budget. Changing policy on a live budget
// requires an explicit transition; it must not silently reset attempts.
type AttemptState struct {
	Used         int
	WindowStart  time.Time
	LastReserved time.Time
	BlockedUntil time.Time
}

// Validate bounds configuration to a finite, nonzero budget. These are
// technical guardrails, not the final production login policy.
func (p AttemptPolicy) Validate() error {
	if p.MaxAttempts < 1 || p.MaxAttempts > 100 ||
		p.Window < time.Second || p.Window > 24*time.Hour ||
		p.Cooldown < time.Second || p.Cooldown > 24*time.Hour {
		return invalid("invalid attempt budget policy")
	}
	return nil
}

func (p AttemptPolicy) validState(s AttemptState) bool {
	if s == (AttemptState{}) {
		return true
	}
	if s.Used < 1 || s.Used > p.MaxAttempts || s.WindowStart.IsZero() ||
		s.LastReserved.IsZero() || s.LastReserved.Before(s.WindowStart) ||
		!s.LastReserved.Before(s.WindowStart.Add(p.Window)) {
		return false
	}
	if s.Used < p.MaxAttempts {
		return s.BlockedUntil.IsZero()
	}
	minEnd := s.WindowStart.Add(p.Window)
	cooldownEnd := s.LastReserved.Add(p.Cooldown)
	if cooldownEnd.After(minEnd) {
		minEnd = cooldownEnd
	}
	return s.BlockedUntil.Equal(minEnd)
}

// Reserve returns the next state and whether one verification is allowed.
// A caller must commit an allowed reservation atomically before executing
// a credential check. Denied requests never extend a lockout. The clock
// must be a trusted server clock, not a terminal-provided timestamp.
func (p AttemptPolicy) Reserve(s AttemptState, at time.Time) (AttemptState, bool, error) {
	if err := p.Validate(); err != nil {
		return s, false, err
	}
	if !p.validState(s) || at.IsZero() || (!s.LastReserved.IsZero() && at.Before(s.LastReserved)) {
		return s, false, invalid("invalid attempt state or clock")
	}
	at = at.UTC()
	if !s.BlockedUntil.IsZero() && at.Before(s.BlockedUntil) {
		return s, false, nil
	}
	if s.Used == 0 || !at.Before(s.WindowStart.Add(p.Window)) {
		s = AttemptState{WindowStart: at}
	}
	s.Used++
	s.LastReserved = at
	if s.Used == p.MaxAttempts {
		s.BlockedUntil = s.WindowStart.Add(p.Window)
		if end := at.Add(p.Cooldown); end.After(s.BlockedUntil) {
			s.BlockedUntil = end
		}
	}
	return s, true, nil
}
