package domain_test

import (
	"testing"
	"time"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptBudgetReservesBeforeVerification(t *testing.T) {
	p := domain.AttemptPolicy{MaxAttempts: 3, Window: time.Minute, Cooldown: 2 * time.Minute}
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	var s domain.AttemptState
	for i := 0; i < 3; i++ {
		next, allowed, err := p.Reserve(s, now.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
		require.True(t, allowed)
		s = next
	}
	assert.Equal(t, 3, s.Used)
	assert.Equal(t, now.Add(122*time.Second), s.BlockedUntil)
	next, allowed, err := p.Reserve(s, now.Add(121*time.Second))
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, s, next)
	next, allowed, err = p.Reserve(s, now.Add(122*time.Second))
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, next.Used)
	assert.True(t, next.BlockedUntil.IsZero())
}
func TestAttemptBudgetWindowAndInvalidState(t *testing.T) {
	p := domain.AttemptPolicy{MaxAttempts: 2, Window: time.Minute, Cooldown: time.Second}
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	s, allowed, err := p.Reserve(domain.AttemptState{}, now)
	require.NoError(t, err)
	require.True(t, allowed)
	_, allowed, err = p.Reserve(s, now.Add(-time.Second))
	require.Error(t, err)
	assert.False(t, allowed)
	s, allowed, err = p.Reserve(s, now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, allowed)
	assert.Equal(t, 1, s.Used)
	s, allowed, err = p.Reserve(s, now.Add(61*time.Second))
	require.NoError(t, err)
	require.True(t, allowed)
	assert.Equal(t, now.Add(2*time.Minute), s.BlockedUntil, "short cooldown must not defeat window")
	_, allowed, err = p.Reserve(domain.AttemptState{Used: -1}, now)
	require.Error(t, err)
	assert.False(t, allowed)
	_, allowed, err = p.Reserve(domain.AttemptState{Used: 1, WindowStart: now, LastReserved: now, BlockedUntil: now.Add(time.Minute)}, now)
	require.Error(t, err)
	assert.False(t, allowed)
	for _, bad := range []domain.AttemptPolicy{{}, {MaxAttempts: 1, Window: time.Second, Cooldown: 0}, {MaxAttempts: 101, Window: time.Minute, Cooldown: time.Minute}, {MaxAttempts: 1, Window: 25 * time.Hour, Cooldown: time.Minute}} {
		_, allowed, err = bad.Reserve(domain.AttemptState{}, now)
		require.Error(t, err)
		assert.False(t, allowed)
	}
	_, allowed, err = p.Reserve(domain.AttemptState{}, time.Time{})
	require.Error(t, err)
	assert.False(t, allowed)
}
