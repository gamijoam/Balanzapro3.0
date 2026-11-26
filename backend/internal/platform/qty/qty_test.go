package qty_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

func TestParseAcceptsCanonicalExactDecimals(t *testing.T) {
	cases := map[string]string{
		"0":                         "0",
		"1":                         "1",
		"120":                       "120",
		"0.5":                       "0.5",
		"1.500":                     "1.5",
		"0.000001":                  "0.000001",
		"-2.25":                     "-2.25",
		"123456789012345678.123456": "123456789012345678.123456",
	}
	for in, want := range cases {
		q, err := qty.Parse(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, q.String(), in)
	}
}

func TestParseRejectsAmbiguousOrInexactInput(t *testing.T) {
	for _, in := range []string{"", " 1", "1 ", "1e3", "1E-2", "0x10", "+1", "01", ".5", "5.", "1,5", "NaN", "Inf", "0.0000001", "1.1234567"} {
		_, err := qty.Parse(in)
		require.Error(t, err, in)
		assert.True(t, errors.Is(err, qty.ErrInvalid), in)
	}
}

func TestArithmeticIsExact(t *testing.T) {
	a := qty.MustParse("0.1")
	b := qty.MustParse("0.2")
	assert.Equal(t, "0.3", a.Add(b).String())
	assert.Equal(t, "-0.1", a.Sub(b).String())
	assert.Equal(t, "120", qty.MustParse("5").Mul(qty.MustParse("24")).String())
	assert.Equal(t, 0, a.Add(b).Cmp(qty.MustParse("0.30")))
}

func TestScale(t *testing.T) {
	assert.Equal(t, int32(0), qty.MustParse("120").Scale())
	assert.Equal(t, int32(0), qty.MustParse("120.000").Scale())
}

func TestScaleNormalizes(t *testing.T) {
	assert.Equal(t, int32(2), qty.MustParse("0.250").Scale())
	assert.Equal(t, int32(6), qty.MustParse("0.000001").Scale())
}

func TestQuoRemGivesWholePresentationsAndRemainder(t *testing.T) {
	whole, rem := qty.MustParse("127").QuoRem(qty.MustParse("24"))
	assert.Equal(t, "5", whole.String())
	assert.Equal(t, "7", rem.String())

	whole, rem = qty.MustParse("10.5").QuoRem(qty.MustParse("2.5"))
	assert.Equal(t, "4", whole.String())
	assert.Equal(t, "0.5", rem.String())
}

func TestSignHelpers(t *testing.T) {
	assert.True(t, qty.Zero().IsZero())
	assert.True(t, qty.MustParse("0.000001").IsPositive())
	assert.True(t, qty.MustParse("-1").IsNegative())
	assert.Equal(t, "-3", qty.MustParse("3").Neg().String())
}

func TestJSONUsesStringsOnly(t *testing.T) {
	b, err := json.Marshal(struct {
		Q qty.Q `json:"q"`
	}{qty.MustParse("1.50")})
	require.NoError(t, err)
	assert.JSONEq(t, `{"q":"1.5"}`, string(b))

	var out struct {
		Q qty.Q `json:"q"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"q":"0.333333"}`), &out))
	assert.Equal(t, "0.333333", out.Q.String())

	// A JSON number would go through float64 in many clients: reject it.
	require.Error(t, json.Unmarshal([]byte(`{"q":0.1}`), &out))
	require.Error(t, json.Unmarshal([]byte(`{"q":"1e2"}`), &out))
}

func TestEqualValuesShareRepresentation(t *testing.T) {
	assert.Equal(t, qty.MustParse("7"), qty.MustParse("7.000000"))
	assert.Equal(t, qty.MustParse("12"), qty.MustParse("0.5").Mul(qty.MustParse("24")))
	assert.Equal(t, qty.Zero(), qty.MustParse("0.000"))
}
