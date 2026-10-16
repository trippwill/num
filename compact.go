// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"math"
	"math/bits"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// compactMarker is an identity token, never numeric storage and never mutated.
// A nil decimal pointer still means uninitialized. Small values keep their
// coefficient inline; negative scale encodes the complement of the actual
// fractional scale for a negative value. No pointers refer into a Num itself.
var compactMarker apd.Decimal

var powersOfTen = [...]uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000,
	100000000, 1000000000, 10000000000, 100000000000,
	1000000000000, 10000000000000, 100000000000000,
	1000000000000000, 10000000000000000, 100000000000000000,
	1000000000000000000, 10000000000000000000,
}

func compact(coefficient uint64, scale int32, negative bool) Num {
	if negative && coefficient != 0 {
		scale = ^scale
	}
	return Num{dec: &compactMarker, coef: coefficient, scale: scale}
}

func (n Num) compactScale() int32 {
	if n.scale < 0 {
		return ^n.scale
	}
	return n.scale
}

func (n Num) decimal(temporary *apd.Decimal) *apd.Decimal {
	if n.dec == &compactMarker {
		temporary.Coeff.SetUint64(n.coef)
		temporary.Exponent = -n.compactScale()
		temporary.Negative = n.scale < 0
		return temporary
	}
	if n.dec == nil {
		return temporary // Comparisons of uninitialized values retain zero ordering.
	}
	return n.dec
}

// The common finite spelling needs one scan and no allocation. Longer inputs
// and exponent notation use the bounded full parser, including its diagnostics.
func parseCompact(s string) (Num, bool) {
	if len(s) == 0 || len(s) > 20 {
		return Num{}, false
	}
	i, negative := 0, false
	if s[0] == '-' || s[0] == '+' {
		negative = s[0] == '-'
		i++
	}
	// At most nineteen coefficient digits always fit. Only an unsigned,
	// twenty-character integer can overflow, so keep that check out of the
	// digit loops used by ordinary prices, quantities, and float conversions.
	if len(s) == 20 && i == 0 && s > "18446744073709551615" && strings.IndexByte(s, '.') < 0 {
		return Num{}, false
	}
	var coefficient uint64
	start := i
	for i < len(s) && s[i]-'0' <= 9 {
		coefficient = coefficient*10 + uint64(s[i]-'0')
		i++
	}
	digits, scale := i-start, int32(0)
	if i < len(s) && s[i] == '.' {
		i++
		start = i
		for i < len(s) && s[i]-'0' <= 9 {
			coefficient = coefficient*10 + uint64(s[i]-'0')
			i++
		}
		scale = int32(i - start)
		digits += i - start
	}
	if i != len(s) || digits == 0 {
		return Num{}, false
	}
	return compact(coefficient, scale, negative), true
}

func multiplyPower(coefficient uint64, power int32) (uint64, bool) {
	if coefficient == 0 {
		return 0, true
	}
	if power >= int32(len(powersOfTen)) {
		return 0, false
	}
	hi, lo := bits.Mul64(coefficient, powersOfTen[power])
	return lo, hi == 0
}

func alignCompact(a, b uint64, leftScale, rightScale int32) (left, right uint64, scale int32, ok bool) {
	if leftScale > rightScale {
		right, ok = multiplyPower(b, leftScale-rightScale)
		return a, right, leftScale, ok
	}
	if rightScale > leftScale {
		left, ok = multiplyPower(a, rightScale-leftScale)
		return left, b, rightScale, ok
	}
	return a, b, leftScale, true
}

func addCompact(coefficient uint64, signedScale int32, otherCoefficient uint64, otherSignedScale int32, subtract bool) Num {
	leftScale, rightScale := signedScale, otherSignedScale
	if leftScale < 0 {
		leftScale = ^leftScale
	}
	if rightScale < 0 {
		rightScale = ^rightScale
	}
	a, b, scale, ok := alignCompact(coefficient, otherCoefficient, leftScale, rightScale)
	if !ok {
		operation := byte('+')
		if subtract {
			operation = '-'
		}
		return compactFallback(coefficient, signedScale, otherCoefficient, otherSignedScale, operation)
	}
	negative, otherNegative := signedScale < 0, otherSignedScale < 0
	if subtract {
		otherNegative = !otherNegative
	}
	if negative == otherNegative {
		sum, carry := bits.Add64(a, b, 0)
		if carry != 0 {
			operation := byte('+')
			if subtract {
				operation = '-'
			}
			return compactFallback(coefficient, signedScale, otherCoefficient, otherSignedScale, operation)
		}
		return compact(sum, scale, negative)
	}
	if a < b {
		return compact(b-a, scale, otherNegative)
	}
	return compact(a-b, scale, negative)
}

func compactFallback(coefficient uint64, scale int32, otherCoefficient uint64, otherScale int32, operation byte) Num {
	n := Num{dec: &compactMarker, coef: coefficient, scale: scale}
	other := Num{dec: &compactMarker, coef: otherCoefficient, scale: otherScale}
	return n.calculate(other, operation)
}

func divideCompact(coefficient uint64, signedScale int32, otherCoefficient uint64, otherSignedScale int32) Num {
	if otherCoefficient == 0 {
		return Num{Err: ErrDivisionByZero}
	}
	if coefficient == 0 {
		return compact(0, 0, false)
	}
	quotient, remainder := coefficient/otherCoefficient, coefficient%otherCoefficient
	leftScale, rightScale := signedScale, otherSignedScale
	if leftScale < 0 {
		leftScale = ^leftScale
	}
	if rightScale < 0 {
		rightScale = ^rightScale
	}
	scale := leftScale - rightScale
	if remainder != 0 {
		// Only terminating, exactly representable quotients take this path.
		// Repeating results go directly to apd's 34-digit half-even division.
		a, b := coefficient, otherCoefficient
		for b != 0 {
			a, b = b, a%b
		}
		denominator := otherCoefficient / a
		denominator >>= bits.TrailingZeros64(denominator)
		for denominator%5 == 0 {
			denominator /= 5
		}
		if denominator != 1 {
			return compactFallback(coefficient, signedScale, otherCoefficient, otherSignedScale, '/')
		}
		for remainder != 0 {
			hi, lo := bits.Mul64(remainder, 10)
			digit, next := bits.Div64(hi, lo, otherCoefficient)
			if quotient > (math.MaxUint64-digit)/10 {
				return compactFallback(coefficient, signedScale, otherCoefficient, otherSignedScale, '/')
			}
			quotient, remainder = quotient*10+digit, next
			scale++
		}
	}
	for scale > 0 && quotient%10 == 0 {
		quotient /= 10
		scale--
	}
	if scale > maxScale {
		return Num{Err: ErrLimitExceeded}
	}
	if scale < 0 {
		var ok bool
		quotient, ok = multiplyPower(quotient, -scale)
		if !ok {
			return compactFallback(coefficient, signedScale, otherCoefficient, otherSignedScale, '/')
		}
		scale = 0
	}
	return compact(quotient, scale, (signedScale < 0) != (otherSignedScale < 0))
}

func quantizeCompact(n Num, scale int) (Num, bool) {
	if scale < 0 || scale > maxScale {
		return Num{Err: ErrInvalidScale}, true
	}
	difference := int32(scale) - n.compactScale()
	coefficient := n.coef
	if difference >= 0 {
		var ok bool
		coefficient, ok = multiplyPower(coefficient, difference)
		if !ok {
			return Num{}, false
		}
	} else if -difference >= int32(len(powersOfTen)) {
		coefficient = 0
	} else {
		divisor := powersOfTen[-difference]
		quotient, remainder := coefficient/divisor, coefficient%divisor
		if remainder > divisor/2 || remainder == divisor/2 && quotient%2 != 0 {
			quotient++
		}
		coefficient = quotient
	}
	return compact(coefficient, int32(scale), n.scale < 0), true
}

func compareCompact(n, other Num) (int, bool) {
	if (n.scale < 0) != (other.scale < 0) {
		if n.scale < 0 {
			return -1, true
		}
		return 1, true
	}
	a, b, _, ok := alignCompact(n.coef, other.coef, n.compactScale(), other.compactScale())
	if !ok {
		return 0, false
	}
	comparison := 0
	if a < b {
		comparison = -1
	} else if a > b {
		comparison = 1
	}
	if n.scale < 0 {
		comparison = -comparison
	}
	return comparison, true
}

func (n Num) appendCompact(dst []byte) []byte {
	if n.compactScale() <= 20 {
		var buffer [24]byte
		return append(dst, compactDigits(buffer[:], n.coef, n.scale)...)
	}
	if n.scale < 0 {
		dst = append(dst, '-')
	}
	start := len(dst)
	dst = strconv.AppendUint(dst, n.coef, 10)
	scale := int(n.compactScale())
	if scale == 0 {
		return dst
	}
	digits := len(dst) - start
	if scale >= digits {
		padding := scale - digits + 2 // Leading zero and decimal point.
		dst = append(dst, make([]byte, padding)...)
		copy(dst[start+padding:], dst[start:start+digits])
		dst[start], dst[start+1] = '0', '.'
		for i := start + 2; i < start+padding; i++ {
			dst[i] = '0'
		}
	} else {
		point := len(dst) - scale
		dst = append(dst, 0)
		copy(dst[point+1:], dst[point:])
		dst[point] = '.'
	}
	return dst
}

func compactText(coefficient uint64, scale int32) string {
	var buffer [24]byte
	actualScale := scale
	if actualScale < 0 {
		actualScale = ^actualScale
	}
	if actualScale > 20 {
		return string(compactDigits(make([]byte, actualScale+3), coefficient, scale))
	}
	return string(compactDigits(buffer[:], coefficient, scale))
}

// Fill backwards so placing the decimal point needs no shifting or second
// coefficient conversion. The caller supplies enough room for the fixed scale.
func compactDigits(buffer []byte, coefficient uint64, signedScale int32) []byte {
	negative, scale := signedScale < 0, signedScale
	if negative {
		scale = ^scale
	}
	position := len(buffer)
	if scale > 0 {
		for ; scale > 0; scale-- {
			position--
			buffer[position] = byte(coefficient%10) + '0'
			coefficient /= 10
		}
		position--
		buffer[position] = '.'
	}
	for {
		position--
		buffer[position] = byte(coefficient%10) + '0'
		coefficient /= 10
		if coefficient == 0 {
			break
		}
	}
	if negative {
		position--
		buffer[position] = '-'
	}
	return buffer[position:]
}
