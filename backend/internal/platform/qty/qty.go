// Package qty implements exact decimal quantities and amounts (ADR-040).
//
// Values are never built from floating point. The only textual form accepted
// is a plain decimal literal ("-12.345"); exponents, signs other than a leading
// minus, separators and more than MaxScale decimals are rejected. JSON uses
// strings only, so clients never round-trip through float64.
package qty

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// MaxScale is the maximum number of decimal places stored (NUMERIC(24,6)).
const MaxScale = 6

// ErrInvalid reports a value that is not an accepted exact decimal literal.
var ErrInvalid = errors.New("qty: invalid exact decimal")

var literal = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// Q is an exact decimal value. The zero value is 0.
type Q struct{ d decimal.Decimal }

// Parse reads a plain decimal literal with at most MaxScale decimals.
func Parse(s string) (Q, error) {
	if !literal.MatchString(s) {
		return Q{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	if i := strings.IndexByte(s, '.'); i >= 0 && len(s)-i-1 > MaxScale {
		return Q{}, fmt.Errorf("%w: more than %d decimals: %q", ErrInvalid, MaxScale, s)
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Q{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return norm(d), nil
}

// norm drops trailing fractional zeros so equal values share one
// representation (7 and 7.000000 compare equal as Go values too).
func norm(d decimal.Decimal) Q {
	return Q{d: decimal.RequireFromString(d.String())}
}

// MustParse is Parse for constants and tests; it panics on invalid input.
func MustParse(s string) Q {
	q, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return q
}

// FromInt builds an exact integer value.
func FromInt(n int64) Q { return norm(decimal.NewFromInt(n)) }

// Zero returns 0.
func Zero() Q { return norm(decimal.Decimal{}) }

// String returns the canonical form: no exponent, no trailing zeros.
func (q Q) String() string { return q.d.String() }

func (q Q) Add(o Q) Q        { return norm(q.d.Add(o.d)) }
func (q Q) Sub(o Q) Q        { return norm(q.d.Sub(o.d)) }
func (q Q) Mul(o Q) Q        { return norm(q.d.Mul(o.d)) }
func (q Q) Neg() Q           { return norm(q.d.Neg()) }
func (q Q) Cmp(o Q) int      { return q.d.Cmp(o.d) }
func (q Q) Equal(o Q) bool   { return q.d.Equal(o.d) }
func (q Q) IsZero() bool     { return q.d.IsZero() }
func (q Q) IsPositive() bool { return q.d.IsPositive() }
func (q Q) IsNegative() bool { return q.d.IsNegative() }

// Scale is the number of significant decimal places (trailing zeros ignored).
func (q Q) Scale() int32 {
	s := q.String()
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return int32(len(s) - i - 1)
	}
	return 0
}

// QuoRem returns the integer quotient and the exact remainder of q / d.
// It panics if d is zero, like integer division.
func (q Q) QuoRem(d Q) (whole Q, rem Q) {
	w, r := q.d.QuoRem(d.d, 0)
	return norm(w), norm(r)
}

// MarshalJSON encodes the value as a JSON string.
func (q Q) MarshalJSON() ([]byte, error) { return json.Marshal(q.String()) }

// UnmarshalJSON accepts only JSON strings holding a valid literal.
func (q *Q) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("%w: expected a JSON string, got %s", ErrInvalid, string(b))
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*q = v
	return nil
}
