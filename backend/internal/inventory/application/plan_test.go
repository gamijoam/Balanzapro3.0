package application_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

func TestPlanDeltasMergesAndOrdersByWarehouseThenSKU(t *testing.T) {
	w1 := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	w2 := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	s1 := uuid.MustParse("10000000-0000-4000-8000-000000000000")
	s2 := uuid.MustParse("20000000-0000-4000-8000-000000000000")
	in := []application.BalanceDelta{
		{Key: application.BalanceKey{WarehouseID: w2, SKUID: s1}, Physical: qty.MustParse("-1")},
		{Key: application.BalanceKey{WarehouseID: w1, SKUID: s2}, Physical: qty.MustParse("-2")},
		{Key: application.BalanceKey{WarehouseID: w1, SKUID: s1}, Physical: qty.MustParse("-3")},
		{Key: application.BalanceKey{WarehouseID: w1, SKUID: s2}, Physical: qty.MustParse("-0.5"), Reserved: qty.MustParse("1")},
	}
	out := application.PlanDeltas(in)
	assert.Len(t, out, 3)
	assert.Equal(t, application.BalanceKey{WarehouseID: w1, SKUID: s1}, out[0].Key)
	assert.Equal(t, application.BalanceKey{WarehouseID: w1, SKUID: s2}, out[1].Key)
	assert.Equal(t, application.BalanceKey{WarehouseID: w2, SKUID: s1}, out[2].Key)
	assert.Equal(t, "-2.5", out[1].Physical.String())
	assert.Equal(t, "1", out[1].Reserved.String())

	reversed := []application.BalanceDelta{in[3], in[2], in[1], in[0]}
	assert.Equal(t, out, application.PlanDeltas(reversed), "order independent of input order")
}
