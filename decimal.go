// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"fmt"
	"github.com/cockroachdb/apd/v3"
)

const (
	maxDigits            = 4096
	maxScale             = 4096
	maxInputBytes        = 16384
	minDivisionPrecision = 34
)

// Contexts are local values: neither callers nor operations can change a policy
// observed by another calculation. Public limits are checked independently of
// apd's adjusted-exponent limits, leaving room for bounded exact intermediates.
func decimalContext(precision uint32) apd.Context {
	return apd.Context{Precision: precision, MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent, Traps: apd.DefaultTraps, Rounding: apd.RoundHalfEven}
}

func meaningfulDigits(d *apd.Decimal) int {
	var reduced apd.Decimal
	reduced.Set(d)
	reduceDecimal(&reduced)
	return int(reduced.NumDigits())
}

// apd.Reduce removes one decimal zero per big-integer division. Remove zeros
// in blocks instead, and skip odd coefficients entirely. This matters when
// Quo pads a short exact quotient to the context's full significant precision.
// d is always fresh storage; operand coefficients are never mutated.
func reduceDecimal(d *apd.Decimal) {
	if d.IsZero() {
		d.Negative = false
		d.Exponent = 0
		return
	}
	if d.Coeff.IsUint64() {
		coefficient := d.Coeff.Uint64()
		for coefficient%10000 == 0 {
			coefficient /= 10000
			d.Exponent += 4
		}
		for coefficient%10 == 0 {
			coefficient /= 10
			d.Exponent++
		}
		d.Coeff.SetUint64(coefficient)
		return
	}
	if d.Coeff.Bit(0) != 0 {
		return
	}
	var quotient, remainder, divisor apd.BigInt
	divisor.SetUint64(10)
	quotient.QuoRem(&d.Coeff, &divisor, &remainder)
	if remainder.Sign() != 0 {
		return
	}
	d.Coeff.Set(&quotient)
	d.Exponent++
	for _, digits := range [...]int32{16, 8, 4, 2, 1} {
		divisor.SetUint64(powersOfTen[digits])
		for {
			quotient.QuoRem(&d.Coeff, &divisor, &remainder)
			if remainder.Sign() != 0 {
				break
			}
			d.Coeff.Set(&quotient)
			d.Exponent += digits
		}
	}
}

// stored takes ownership of a fresh decimal. No operand is passed here directly.
func stored(d *apd.Decimal) Num {
	if err := normalizeStored(d); err != nil {
		return Num{Err: err}
	}
	if d.Coeff.IsUint64() {
		return compact(d.Coeff.Uint64(), -d.Exponent, d.Negative)
	}
	return Num{dec: d}
}

func normalizeStored(d *apd.Decimal) error {
	if d.Form != apd.Finite {
		return fmt.Errorf("%w: nonfinite result", ErrLimitExceeded)
	}
	digits := d.NumDigits()
	if digits > maxDigits || d.Exponent < -maxScale {
		return ErrLimitExceeded
	}
	if d.IsZero() {
		d.Negative = false
		if d.Exponent > 0 {
			d.Exponent = 0
		}
		return nil
	}
	if digits+int64(d.Exponent) > maxDigits {
		return ErrLimitExceeded
	}
	if d.Exponent > 0 {
		var power, exponent apd.BigInt
		power.SetInt64(10)
		exponent.SetInt64(int64(d.Exponent))
		power.Exp(&power, &exponent, nil)
		d.Coeff.Mul(&d.Coeff, &power)
		d.Exponent = 0
	}
	return nil
}

// quantizeDecimal rounds a temporary without enforcing stored coefficient
// capacity. Formatting may legitimately pad more digits than Num can store.
func quantizeDecimal(d *apd.Decimal, scale int) (*apd.Decimal, error) {
	if scale < 0 || scale > maxScale {
		return nil, ErrInvalidScale
	}
	precision := max(int64(1), d.NumDigits()+int64(d.Exponent)+int64(scale)+1)
	ctx := decimalContext(uint32(precision))
	result := new(apd.Decimal)
	if _, err := ctx.Quantize(result, d, -int32(scale)); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLimitExceeded, err)
	}
	if result.IsZero() {
		result.Negative = false
	}
	return result, nil
}

// validateInput checks syntax and capacity before apd allocates a coefficient or
// expands an exponent. Leading zeros do not count as coefficient digits.
func validateInput(s string) error {
	if len(s) > maxInputBytes {
		return ErrLimitExceeded
	}
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits, fraction, coefficient := 0, 0, 0
	point, nonzero := false, false
	for i < len(s) && s[i] != 'e' && s[i] != 'E' {
		c := s[i]
		if c == '.' && !point {
			point = true
		} else if c >= '0' && c <= '9' {
			digits++
			if point {
				fraction++
			}
			nonzero = nonzero || c != '0'
			if nonzero {
				coefficient++
			}
		} else {
			return ErrInvalidNumericString
		}
		i++
	}
	if digits == 0 {
		return ErrInvalidNumericString
	}
	exponent := 0
	if i < len(s) {
		i++
		negative := false
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			negative = s[i] == '-'
			i++
		}
		start := i
		tooLarge := false
		for ; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				return ErrInvalidNumericString
			}
			if exponent > (apd.MaxExponent-int(s[i]-'0'))/10 {
				tooLarge = true
			} else if !tooLarge {
				exponent = exponent*10 + int(s[i]-'0')
			}
		}
		if i == start {
			return ErrInvalidNumericString
		}
		if tooLarge {
			return ErrLimitExceeded
		}
		if negative {
			exponent = -exponent
		}
	}
	exponent -= fraction
	if coefficient > maxDigits || exponent < -maxScale ||
		(nonzero && coefficient+exponent > maxDigits) {
		return ErrLimitExceeded
	}
	return nil
}
