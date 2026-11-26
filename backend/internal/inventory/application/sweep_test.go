package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
)

// fakeTx implements only what ExpireReservations needs; any other call
// panics through the nil embedded interface.
type fakeTx struct {
	application.Tx
	failFor uuid.UUID
	swept   *[]uuid.UUID
}

func (f fakeTx) ExpireDueReservations(_ context.Context, company uuid.UUID, _ time.Time, _ int) ([]application.Reservation, error) {
	*f.swept = append(*f.swept, company)
	if company == f.failFor {
		return nil, errors.New("boom")
	}
	return nil, nil
}

type fakeStore struct{ tx fakeTx }

func (s fakeStore) InTx(_ context.Context, fn func(application.Tx) error) error { return fn(s.tx) }

// Second independent review: one failing company must not stop the sweep
// of the others.
func TestSweepContinuesAfterCompanyError(t *testing.T) {
	bad, good1, good2 := uuid.New(), uuid.New(), uuid.New()
	var swept []uuid.UUID
	svc := application.NewService(fakeStore{tx: fakeTx{failFor: bad, swept: &swept}}, nil)
	_, err := svc.SweepExpiredReservations(context.Background(), []uuid.UUID{bad, good1, good2}, 10)
	assert.Error(t, err, "the failure is still reported")
	assert.Equal(t, []uuid.UUID{bad, good1, good2}, swept)
}
