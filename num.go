// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"

	"github.com/cockroachdb/apd/v3"
)

// Num is an immutable, error-carrying finite decimal. Its zero value is
// uninitialized; use Zero or a constructor for a valid numeric zero.
type Num struct {
	_     [0]func()
	dec   *apd.Decimal
	coef  uint64
	scale int32
	Err   error
}

// FromString constructs an exact decimal, retaining fractional trailing zeros.
// Malformed, nonfinite, or out-of-capacity inputs produce an error-state Num.
func FromString(s string) Num {
	if n, ok := parseCompact(s); ok {
		return n
	}
	if err := validateInput(s); err != nil {
		return Num{Err: err}
	}
	ctx := decimalContext(0)
	d, _, err := ctx.NewFromString(s)
	if err != nil {
		return Num{Err: fmt.Errorf("%w: %w", ErrLimitExceeded, err)}
	}
	return stored(d)
}

// FromInt64 constructs an exact integer at scale zero.
func FromInt64(v int64) Num {
	coefficient := uint64(v)
	if v < 0 {
		coefficient = -coefficient
	}
	return compact(coefficient, 0, v < 0)
}

// FromFloat64 interprets v using its shortest round-trip decimal spelling,
// rather than the exact binary fraction. NaN and infinities are invalid.
func FromFloat64(v float64) Num {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return Num{Err: ErrInvalidNumericString}
	}
	var buffer [32]byte
	text := strconv.AppendFloat(buffer[:0], v, 'f', -1, 64)
	negative := text[0] == '-'
	digits := text
	if negative {
		digits = digits[1:]
	}
	if len(digits) <= 19 {
		// strconv already guarantees finite decimal syntax here. Nineteen
		// characters cannot overflow uint64, so no second validation is needed.
		var coefficient uint64
		var scale int32
		for i, c := range digits {
			if c == '.' {
				scale = int32(len(digits) - i - 1)
			} else {
				coefficient = coefficient*10 + uint64(c-'0')
			}
		}
		return compact(coefficient, scale, negative)
	}
	return FromString(string(text))
}

// Zero constructs an initialized zero at scale zero.
func Zero() Num { return FromInt64(0) }

// Ok reports whether n is initialized and has no carried error.
func (n Num) Ok() bool { return n.stateError() == nil }

func (n Num) stateError() error {
	if n.Err != nil {
		return n.Err
	}
	if n.dec == nil {
		return ErrUninitialized
	}
	return nil
}

// IsZero reports whether n is valid and numerically zero.
func (n Num) IsZero() bool {
	if !n.Ok() {
		return false
	}
	if n.dec == &compactMarker {
		return n.coef == 0
	}
	return n.dec.IsZero()
}

// Add returns the exact sum, retaining the greater operand scale.
func (n Num) Add(other Num) Num {
	if n.dec == &compactMarker && other.dec == &compactMarker && n.Err == nil && other.Err == nil {
		return addCompact(n.coef, n.scale, other.coef, other.scale, false)
	}
	return n.calculate(other, '+')
}

// Sub returns the exact difference, retaining the greater operand scale.
func (n Num) Sub(other Num) Num {
	if n.dec == &compactMarker && other.dec == &compactMarker && n.Err == nil && other.Err == nil {
		return addCompact(n.coef, n.scale, other.coef, other.scale, true)
	}
	return n.calculate(other, '-')
}

// Mul returns the exact product, retaining the sum of operand scales.
func (n Num) Mul(other Num) Num {
	if n.dec == &compactMarker && other.dec == &compactMarker && n.Err == nil && other.Err == nil {
		scale := n.compactScale() + other.compactScale()
		if scale > maxScale {
			return Num{Err: ErrLimitExceeded}
		}
		hi, lo := bits.Mul64(n.coef, other.coef)
		if hi == 0 {
			return compact(lo, scale, (n.scale < 0) != (other.scale < 0))
		}
	}
	return n.calculate(other, '*')
}

// Div divides n by other using half-even rounding. Its precision is the greater
// of 34 and each operand's coefficient digits after removing trailing zeros.
// Repeating expansions succeed; a zero divisor or exceeded capacity does not.
func (n Num) Div(other Num) Num {
	if n.dec == &compactMarker && other.dec == &compactMarker && n.Err == nil && other.Err == nil {
		return divideCompact(n.coef, n.scale, other.coef, other.scale)
	}
	return n.calculate(other, '/')
}

func (n Num) calculate(other Num, operation byte) Num {
	if err := n.stateError(); err != nil {
		return Num{Err: err}
	}
	if err := other.stateError(); err != nil {
		return Num{Err: err}
	}
	var left, right apd.Decimal
	a, b := n.decimal(&left), other.decimal(&right)
	ctx := decimalContext(0)
	d := new(apd.Decimal)
	var err error
	switch operation {
	case '+':
		_, err = ctx.Add(d, a, b)
	case '-':
		_, err = ctx.Sub(d, a, b)
	case '*':
		_, err = ctx.Mul(d, a, b)
	case '/':
		if b.IsZero() {
			return Num{Err: ErrDivisionByZero}
		}
		ctx.Precision = uint32(max(minDivisionPrecision, meaningfulDigits(a), meaningfulDigits(b)))
		_, err = ctx.Quo(d, a, b)
		if err == nil {
			reduceDecimal(d)
		}
	}
	if err != nil {
		return Num{Err: fmt.Errorf("%w: %w", ErrLimitExceeded, err)}
	}
	return stored(d)
}

// Quantize rounds half-even or pads zeros to exactly scale fractional places.
// Scale must be between 0 and 4096 and the result must fit Num's capacity.
func (n Num) Quantize(scale int) Num {
	if err := n.stateError(); err != nil {
		return Num{Err: err}
	}
	if n.dec == &compactMarker {
		if result, ok := quantizeCompact(n, scale); ok {
			return result
		}
	}
	var temporary apd.Decimal
	d, err := quantizeDecimal(n.decimal(&temporary), scale)
	if err != nil {
		return Num{Err: err}
	}
	return stored(d)
}

// Neg returns the negation, preserving scale and propagating errors.
func (n Num) Neg() Num {
	if err := n.stateError(); err != nil {
		return Num{Err: err}
	}
	if n.dec == &compactMarker {
		if n.coef != 0 {
			n.scale = ^n.scale
		}
		return n
	}
	return stored(new(apd.Decimal).Neg(n.dec))
}

// Abs returns the absolute value, preserving scale and propagating errors.
func (n Num) Abs() Num {
	if err := n.stateError(); err != nil {
		return Num{Err: err}
	}
	if n.dec == &compactMarker {
		if n.scale < 0 {
			n.scale = ^n.scale
		}
		return n
	}
	return stored(new(apd.Decimal).Abs(n.dec))
}

// String returns exact plain decimal text, or an explicit error marker.
func (n Num) String() string {
	if err := n.stateError(); err != nil {
		return "<error: " + err.Error() + ">"
	}
	if n.dec == &compactMarker {
		return compactText(n.coef, n.scale)
	}
	return n.dec.Text('f')
}

// Cmp compares numerical values, independently of scale. Check both Ok first.
func (n Num) Cmp(other Num) int {
	if n.dec == &compactMarker && other.dec == &compactMarker {
		if comparison, ok := compareCompact(n, other); ok {
			return comparison
		}
	}
	var left, right apd.Decimal
	a, b := n.decimal(&left), other.decimal(&right)
	return a.Cmp(b)
}

// Equal reports numeric equality. Check both Ok first.
func (n Num) Equal(other Num) bool { return n.Cmp(other) == 0 }

// LessThan reports numeric ordering. Check both Ok first.
func (n Num) LessThan(other Num) bool { return n.Cmp(other) < 0 }

// GreaterThan reports numeric ordering. Check both Ok first.
func (n Num) GreaterThan(other Num) bool { return n.Cmp(other) > 0 }
