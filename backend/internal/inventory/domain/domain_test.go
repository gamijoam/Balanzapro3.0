package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

var company = uuid.MustParse("00000000-0000-4000-8000-000000000001")

func unitSKU(t *testing.T) domain.SKU {
	t.Helper()
	s, err := domain.NewSKU(uuid.New(), company, "HARINA-1KG", "Harina 1 kg", domain.KindUnit, "und", 0)
	require.NoError(t, err)
	return s
}

func TestNewSKUValidates(t *testing.T) {
	_, err := domain.NewSKU(uuid.New(), company, "", "x", domain.KindUnit, "und", 0)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewSKU(uuid.New(), company, "A", "x", "liquid", "und", 0)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewSKU(uuid.New(), company, "A", "x", domain.KindWeight, "kg", 7)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewSKU(uuid.Nil, company, "A", "x", domain.KindWeight, "kg", 3)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewSKU(uuid.New(), uuid.Nil, "A", "x", domain.KindWeight, "kg", 3)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	s, err := domain.NewSKU(uuid.New(), company, " A ", "Arroz", domain.KindWeight, "kg", 3)
	require.NoError(t, err)
	assert.Equal(t, "A", s.Code)
}

func TestSKUQuantityRespectsPrecision(t *testing.T) {
	und := unitSKU(t)
	assert.NoError(t, und.ValidateQuantity(qty.MustParse("3")))
	assert.ErrorIs(t, und.ValidateQuantity(qty.MustParse("0.5")), domain.ErrPrecision)
	assert.ErrorIs(t, und.ValidateQuantity(qty.Zero()), domain.ErrInvalidArgument)
	assert.ErrorIs(t, und.ValidateQuantity(qty.MustParse("-1")), domain.ErrInvalidArgument)

	kg, err := domain.NewSKU(uuid.New(), company, "QUESO", "Queso", domain.KindWeight, "kg", 3)
	require.NoError(t, err)
	assert.NoError(t, kg.ValidateQuantity(qty.MustParse("0.250")))
	assert.ErrorIs(t, kg.ValidateQuantity(qty.MustParse("0.2505")), domain.ErrPrecision)
}

func TestPresentationFactorMustBePositiveAndRepresentable(t *testing.T) {
	und := unitSKU(t)
	_, err := domain.NewPresentation(uuid.New(), und, "Caja", qty.Zero())
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewPresentation(uuid.New(), und, "Media", qty.MustParse("0.5"))
	assert.ErrorIs(t, err, domain.ErrPrecision, "a unit SKU cannot have fractional presentations")
	_, err = domain.NewPresentation(uuid.New(), und, "", qty.MustParse("24"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

// INV-03: Caja24, paquete6 y unidad descuentan un único stock.
func TestINV03PresentationsShareOneBaseStock(t *testing.T) {
	und := unitSKU(t)
	caja, err := domain.NewPresentation(uuid.New(), und, "Caja x24", qty.MustParse("24"))
	require.NoError(t, err)
	paquete, err := domain.NewPresentation(uuid.New(), und, "Paquete x6", qty.MustParse("6"))
	require.NoError(t, err)

	base := qty.FromInt(120)
	assert.Equal(t, "120", caja.ToBase(qty.MustParse("5")).String())
	assert.Equal(t, "120", paquete.ToBase(qty.MustParse("20")).String())

	whole, rem := domain.Breakdown(base, caja)
	assert.Equal(t, "5", whole.String())
	assert.True(t, rem.IsZero())
	whole, rem = domain.Breakdown(base, paquete)
	assert.Equal(t, "20", whole.String())
	assert.True(t, rem.IsZero())

	whole, rem = domain.Breakdown(qty.FromInt(31), caja)
	assert.Equal(t, "1", whole.String())
	assert.Equal(t, "7", rem.String())
}

func TestFractionalPresentationOnWeightSKU(t *testing.T) {
	kg, err := domain.NewSKU(uuid.New(), company, "AZUCAR", "Azúcar", domain.KindWeight, "kg", 3)
	require.NoError(t, err)
	saco, err := domain.NewPresentation(uuid.New(), kg, "Saco 45.5 kg", qty.MustParse("45.5"))
	require.NoError(t, err)
	assert.Equal(t, "91", saco.ToBase(qty.MustParse("2")).String())
	assert.Equal(t, "22.75", saco.ToBase(qty.MustParse("0.5")).String())
}

func TestNormalizeBarcodeKeepsTextSemantics(t *testing.T) {
	c, err := domain.NormalizeBarcode("  0012345678905 ")
	require.NoError(t, err)
	assert.Equal(t, "0012345678905", c, "leading zeros are significant")
	c, err = domain.NormalizeBarcode("Ab-12")
	require.NoError(t, err)
	assert.Equal(t, "Ab-12", c, "Code128 is case-sensitive")
	for _, bad := range []string{"", "   ", "12 34", "a\tb", string(make([]byte, 65))} {
		_, err := domain.NormalizeBarcode(bad)
		assert.True(t, errors.Is(err, domain.ErrInvalidArgument), "%q", bad)
	}
}
