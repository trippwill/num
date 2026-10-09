// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"strconv"

	"github.com/govalues/decimal"
)

// Num is an error-carrying decimal type for financial data.
// It contains a [decimal.Decimal] with a fixed per-binary scale and
// error propagation through chained arithmetic.
type Num struct {
	dec decimal.Decimal
	Err error
	_   [0]func()
}

// From creates a Num from an existing [decimal.Decimal], rescaled to the process scale.
func From(d decimal.Decimal) Num {
	return Num{dec: d.Rescale(processScale)}
}

// FromInt64 creates a Num from an int64 value.
func FromInt64(v int64) Num {
	d, err := decimal.New(v, 0)
	if err != nil {
		return Num{Err: fmt.Errorf("%w: %d: %w", ErrIntegerOverflow, v, err)}
	}
	return Num{dec: d.Rescale(processScale)}
}

// FromFloat64 creates a Num from a float64 value.
func FromFloat64(v float64) Num {
	d, err := decimal.NewFromFloat64(v)
	if err != nil {
		return Num{Err: fmt.Errorf("%w: %v: %w", ErrInvalidNumericString, v, err)}
	}
	return Num{dec: d.Rescale(processScale)}
}

// FromString creates a Num from a string representation.
// It rounds the original decimal once to the process scale using half-to-even.
// Invalid input or a rounded value that cannot fit at that scale sets [Num.Err].
func FromString(s string) Num {
	d, err := parseAtScale(s)
	if err != nil {
		return Num{Err: fmt.Errorf("%w: %q: %w", ErrInvalidNumericString, s, err)}
	}
	return Num{dec: d}
}

func parseAtScale(s string) (decimal.Decimal, error) {
	// Validate the backend's syntax and input limits before exact conversion.
	if _, err := decimal.Parse(s); err != nil {
		return decimal.Decimal{}, err
	}
	value, ok := new(big.Rat).SetString(s)
	if !ok {
		return decimal.Decimal{}, fmt.Errorf("parsing decimal: invalid numeric string")
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(processScale)), nil)
	numerator := new(big.Int).Abs(value.Num())
	numerator.Mul(numerator, factor)
	coef, remainder := new(big.Int), new(big.Int)
	coef.QuoRem(numerator, value.Denom(), remainder)
	half := remainder.Lsh(remainder, 1).Cmp(value.Denom())
	if half > 0 || (half == 0 && coef.Bit(0) != 0) {
		coef.Add(coef, big.NewInt(1))
	}
	if value.Sign() < 0 {
		coef.Neg(coef)
	}
	return decimal.ParseExact(coef.String()+"e-"+strconv.Itoa(processScale), processScale)
}

// Zero returns a Num with value zero at the process scale.
func Zero() Num {
	return Num{dec: decimal.MustNew(0, processScale)}
}

// Ok reports whether the Num has no error and its decimal is normalized to the process scale.
func (n Num) Ok() bool { return n.Err == nil && n.dec.Scale() == processScale }

// IsZero reports whether the Num is ok and its value is zero.
func (n Num) IsZero() bool { return n.Ok() && n.dec.IsZero() }

// Add returns the sum of n and other with error propagation.
func (n Num) Add(other Num) Num {
	if n.Err != nil {
		return n
	}
	if other.Err != nil {
		return other
	}
	d, err := n.dec.Add(other.dec)
	if err != nil {
		return Num{Err: err}
	}
	return Num{dec: d.Rescale(processScale)}
}

// Sub returns the difference of n and other with error propagation.
func (n Num) Sub(other Num) Num {
	if n.Err != nil {
		return n
	}
	if other.Err != nil {
		return other
	}
	d, err := n.dec.Sub(other.dec)
	if err != nil {
		return Num{Err: err}
	}
	return Num{dec: d.Rescale(processScale)}
}

// Mul returns the product of n and other with error propagation.
func (n Num) Mul(other Num) Num {
	if n.Err != nil {
		return n
	}
	if other.Err != nil {
		return other
	}
	d, err := n.dec.Mul(other.dec)
	if err != nil {
		return Num{Err: err}
	}
	return Num{dec: d.Rescale(processScale)}
}

// Div returns the quotient of n divided by other with error propagation.
// Division by zero produces an error-state Num.
func (n Num) Div(other Num) Num {
	if n.Err != nil {
		return n
	}
	if other.Err != nil {
		return other
	}
	d, err := n.dec.Quo(other.dec)
	if err != nil {
		return Num{Err: err}
	}
	return Num{dec: d.Rescale(processScale)}
}

// Neg returns the negation of n with error propagation.
func (n Num) Neg() Num {
	if n.Err != nil {
		return n
	}
	return Num{dec: n.dec.Neg()}
}

// Abs returns the absolute value of n with error propagation.
func (n Num) Abs() Num {
	if n.Err != nil {
		return n
	}
	return Num{dec: n.dec.Abs()}
}

// String implements [fmt.Stringer]. Returns the decimal string representation,
// or an error indicator if the Num is in an error state.
func (n Num) String() string {
	if n.Err != nil {
		return "<error: " + n.Err.Error() + ">"
	}
	return n.dec.String()
}

// Cmp compares n and other, returning -1, 0, or +1.
// Caller should check [Num.Ok] before comparing.
func (n Num) Cmp(other Num) int { return n.dec.Cmp(other.dec) }

// Equal reports whether n and other have the same value.
// Caller should check [Num.Ok] before comparing.
func (n Num) Equal(other Num) bool { return n.dec.Equal(other.dec) }

// LessThan reports whether n is less than other.
// Caller should check [Num.Ok] before comparing.
func (n Num) LessThan(other Num) bool { return n.dec.Less(other.dec) }

// GreaterThan reports whether n is greater than other.
// Caller should check [Num.Ok] before comparing.
func (n Num) GreaterThan(other Num) bool { return other.dec.Less(n.dec) }

// MarshalJSON returns the JSON encoding of n as a quoted decimal string.
// Returns an error if n is in an error state.
func (n Num) MarshalJSON() ([]byte, error) {
	if n.Err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMarshalErrorState, n.Err)
	}
	return n.dec.MarshalJSON()
}

// UnmarshalJSON parses JSON input, accepting both bare numeric (123.45)
// and quoted string ("123.45") formats. Parse failures set [Num.Err]
// rather than returning an error, allowing partial struct unmarshal.
// Numeric input uses the single-round conversion described by [FromString].
func (n *Num) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		n.Err = fmt.Errorf("%w: null", ErrInvalidNumericString)
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	d, err := parseAtScale(s)
	if err != nil {
		n.Err = fmt.Errorf("%w: %s: %w", ErrInvalidNumericString, string(data), err)
		return nil
	}
	n.dec = d
	n.Err = nil
	return nil
}

// MarshalText returns the text encoding of n.
// Delegates to [decimal.Decimal] for XML/Flex report compatibility.
func (n Num) MarshalText() ([]byte, error) {
	if n.Err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMarshalErrorState, n.Err)
	}
	return n.dec.MarshalText()
}

// UnmarshalText parses text input.
// It uses the single-round conversion described by [FromString].
func (n *Num) UnmarshalText(data []byte) error {
	d, err := parseAtScale(string(data))
	if err != nil {
		n.dec = decimal.Decimal{}
		n.Err = err
		return err
	}
	n.dec = d
	n.Err = nil
	return nil
}

// Scan implements [database/sql.Scanner].
// Accepts string, []byte, int64, and float64 inputs.
// String and []byte inputs use the single-round conversion described by [FromString].
func (n *Num) Scan(src any) error {
	if src == nil {
		n.dec = decimal.Decimal{}
		n.Err = fmt.Errorf("%w: nil", ErrUnsupportedType)
		return n.Err
	}
	var d decimal.Decimal
	var err error
	switch src := src.(type) {
	case string:
		d, err = parseAtScale(src)
	case []byte:
		d, err = parseAtScale(string(src))
	default:
		err = d.Scan(src)
		d = d.Rescale(processScale)
	}
	if err != nil {
		n.dec = decimal.Decimal{}
		n.Err = err
		return err
	}
	n.dec = d
	n.Err = nil
	return nil
}

// Value implements [database/sql/driver.Valuer].
// Returns the decimal as a string for exact database roundtrips.
func (n Num) Value() (driver.Value, error) {
	if n.Err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMarshalErrorState, n.Err)
	}
	return n.dec.String(), nil
}
