package admission_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/admission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateConcurrentAndRate(t *testing.T) {
	now := time.Unix(1000, 0)
	clock := func() time.Time { return now }
	g, err := admission.New(2, 4, time.Second, clock)
	require.NoError(t, err)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			assert.NoError(t, g.Run(context.Background(), func() error { entered <- struct{}{}; <-release; return nil }))
		})
	}
	<-entered
	<-entered
	var ran atomic.Int32
	var rejects sync.WaitGroup
	for i := 0; i < 20; i++ {
		rejects.Go(func() {
			assert.ErrorIs(t, g.Run(context.Background(), func() error { ran.Add(1); return nil }), admission.ErrBusy)
		})
	}
	rejects.Wait()
	assert.Zero(t, ran.Load())
	close(release)
	wg.Wait()
	for i := 0; i < 2; i++ {
		require.NoError(t, g.Run(context.Background(), func() error { return nil }))
	}
	assert.ErrorIs(t, g.Run(context.Background(), func() error { return nil }), admission.ErrRate)
	now = now.Add(time.Second)
	require.NoError(t, g.Run(context.Background(), func() error { return nil }))
	now = now.Add(-time.Millisecond)
	assert.ErrorIs(t, g.Run(context.Background(), func() error { return nil }), admission.ErrClock)
}
func TestGateReleasesAfterPanicAndCancellation(t *testing.T) {
	g, err := admission.New(1, 10, time.Second, time.Now)
	require.NoError(t, err)
	assert.Panics(t, func() { _ = g.Run(context.Background(), func() error { panic("fixture") }) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, g.Run(ctx, func() error { t.Error("unexpected run"); return nil }), context.Canceled)
	require.NoError(t, g.Run(context.Background(), func() error { return nil }))
	_, err = admission.New(0, 1, time.Second, time.Now)
	require.Error(t, err)
	_, err = admission.New(1, 0, time.Second, time.Now)
	require.Error(t, err)
	_, err = admission.New(1, 1, 0, time.Now)
	require.Error(t, err)
	_, err = admission.New(1, 1, time.Second, nil)
	require.Error(t, err)
}
