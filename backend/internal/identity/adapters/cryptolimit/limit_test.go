package cryptolimit_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/cryptolimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiterConcurrency(t *testing.T) {
	l, err := cryptolimit.New(2)
	require.NoError(t, err)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var running, peak atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			err := l.Run(context.Background(), func() error {
				n := running.Add(1)
				for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
				}
				entered <- struct{}{}
				<-release
				running.Add(-1)
				return nil
			})
			assert.NoError(t, err)
		})
	}
	<-entered
	<-entered
	var rejected sync.WaitGroup
	for i := 0; i < 20; i++ {
		rejected.Go(func() {
			assert.ErrorIs(t, l.Run(context.Background(), func() error { t.Error("unexpected execution"); return nil }), cryptolimit.ErrBusy)
		})
	}
	rejected.Wait()
	close(release)
	wg.Wait()
	assert.Equal(t, int32(2), peak.Load())
	require.NoError(t, l.Run(context.Background(), func() error { return nil }))
}
func TestLimiterReleasesAndRejectsInvalidInputs(t *testing.T) {
	for _, n := range []int{0, -1, 65} {
		_, err := cryptolimit.New(n)
		assert.Error(t, err)
	}
	l, err := cryptolimit.New(1)
	require.NoError(t, err)
	sentinel := errors.New("synthetic failure")
	assert.ErrorIs(t, l.Run(context.Background(), func() error { return sentinel }), sentinel)
	assert.Panics(t, func() { _ = l.Run(context.Background(), func() error { panic("synthetic") }) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, l.Run(ctx, func() error { t.Error("unexpected execution"); return nil }), context.Canceled)
	assert.Error(t, l.Run(context.Background(), nil))
	assert.NoError(t, l.Run(context.Background(), func() error { return nil }))
}
