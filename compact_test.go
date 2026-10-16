// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestArithmeticAcrossCompactBoundary(t *testing.T) {
	n := FromString("18446744073709551615")
	requireNum(t, n.Add(FromInt64(1)), "18446744073709551616")
	requireNum(t, n.Add(FromInt64(1)).Sub(FromInt64(1)), n.String())
	requireNum(t, n.Mul(FromInt64(2)), "36893488147419103230")
	requireNum(t, n.Mul(FromInt64(2)).Div(FromInt64(2)), n.String())
	requireNum(t, n.Neg(), "-18446744073709551615")
	requireNum(t, n.Neg().Abs(), n.String())
	requireNum(t, n.Quantize(1), "18446744073709551615.0")
	requireNum(t, n.Quantize(1).Quantize(0), n.String())
	requireNum(t, FromString("1844674407370955161.5").Quantize(0), "1844674407370955162")
	requireNum(t, FromString("1e-4096").Mul(FromInt64(10)).Div(FromInt64(10)), "0."+strings.Repeat("0", 4095)+"1")
	requireNum(t, n.Add(n.Neg()), "0")
	if !n.GreaterThan(FromString("18446744073709551614.9")) || !n.Equal(n.Quantize(1)) {
		t.Fatal("comparison across coefficient capacities lost precision")
	}
}

func TestDivisionRemovesPrecisionPadding(t *testing.T) {
	for _, input := range []string{
		"0.00000001", "1.2300000000000000000000000000000000000000",
		"12345678901234567890123456789012345678900.0000",
		strings.Repeat("9", 4000) + strings.Repeat("0", 96),
	} {
		n := FromString(input)
		got := n.Div(FromInt64(1))
		if !got.Ok() || !got.Equal(n) {
			t.Fatalf("division by one changed %s: %v", input, got)
		}
		if strings.Contains(got.String(), ".") && strings.HasSuffix(got.String(), "0") {
			t.Fatalf("division retained fractional padding: %s", got)
		}
		requireNum(t, n, input)
	}
	precise := FromString("1." + strings.Repeat("23456789", 6))
	padded := FromString(precise.String() + strings.Repeat("0", 100))
	requireNum(t, padded.Div(FromInt64(3)), precise.Div(FromInt64(3)).String())
}

func TestCarriedErrorsOnInitializedStorage(t *testing.T) {
	bad := FromString("123.4500")
	bad.Err = ErrUnsupportedType
	good := FromInt64(2)
	for _, got := range []Num{
		bad.Add(good), good.Add(bad), bad.Sub(good), good.Sub(bad),
		bad.Mul(good), good.Mul(bad), bad.Div(good), good.Div(bad),
		bad.Quantize(2), bad.Neg(), bad.Abs(),
	} {
		if got.Ok() || !errors.Is(got.Err, ErrUnsupportedType) {
			t.Fatalf("initialized storage bypassed carried error: %v", got)
		}
	}
	if _, err := bad.Value(); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("SQL output bypassed carried error: %v", err)
	}
	if _, err := bad.MarshalJSON(); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("JSON output bypassed carried error: %v", err)
	}
}

func TestJSONTokenGrammarOnFastPaths(t *testing.T) {
	for _, input := range []string{`+1`, `.5`, `01`, `00`, `-01`, `1.`, `1.e2`, `1_0`, `"1""`, `"1` + "\n" + `2"`, `"1"trailing"`} {
		n := FromInt64(42)
		if err := n.UnmarshalJSON([]byte(input)); err != nil || n.Ok() || !errors.Is(n.Err, ErrInvalidNumericString) {
			t.Fatalf("invalid JSON %q produced %v, %v", input, n, err)
		}
	}
	for _, input := range []string{`  -0.00 `, `"+1.2300"`, `".5"`, `"1."`, `0e3`, `1.20E+1`, `"1\u002e2300"`} {
		var n Num
		if err := n.UnmarshalJSON([]byte(input)); err != nil || !n.Ok() {
			t.Fatalf("valid numeric JSON %q rejected: %v, %v", input, n, err)
		}
	}
}

func TestConcurrentCompactArithmetic(t *testing.T) {
	n := FromString("123.4500")
	alias := n
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 30 {
				requireNum(t, n.Add(FromString("0.0001")).Sub(FromString("0.0001")), "123.4500")
				requireNum(t, n.Mul(FromInt64(2)).Div(FromInt64(2)), "123.45")
				requireNum(t, n.Neg().Abs(), "123.4500")
				requireNum(t, alias, "123.4500")
			}
		})
	}
	wg.Wait()
}

func TestSmallCalculationsDoNotAllocate(t *testing.T) {
	a, b, c := FromString("100.50"), FromInt64(10), FromInt64(3)
	var result Num
	for _, tc := range []struct {
		name string
		run  func()
	}{
		{"string", func() { result = FromString("123.456789") }},
		{"integer", func() { result = FromInt64(123456) }},
		{"float", func() { result = FromFloat64(123.456789) }},
		{"chain", func() { result = a.Add(b).Mul(c).Sub(a).Div(b) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, tc.run); allocations != 0 {
				t.Fatalf("small-value %s allocated %g times; want zero", tc.name, allocations)
			}
			if !result.Ok() {
				t.Fatal(result.Err)
			}
		})
	}
}

func FuzzNumScaledArithmetic(f *testing.F) {
	for _, seed := range []struct {
		a, b   int64
		as, bs uint8
	}{{1, 3, 0, 0}, {1, 2, 20, 30}, {-1225, 49, 3, 2}, {9223372036854775807, -9223372036854775808, 18, 1}} {
		f.Add(seed.a, seed.b, seed.as, seed.bs)
	}
	f.Fuzz(func(t *testing.T, a, b int64, leftScale, rightScale uint8) {
		as, bs := int(leftScale%64), int(rightScale%64)
		x, y := FromString(scaledIntegerText(big.NewInt(a), as)), FromString(scaledIntegerText(big.NewInt(b), bs))
		bx, by := big.NewInt(a), big.NewInt(b)
		scale := max(as, bs)
		alignedA := new(big.Int).Mul(bx, decimalPower(scale-as))
		alignedB := new(big.Int).Mul(by, decimalPower(scale-bs))
		requireNum(t, x.Add(y), scaledIntegerText(new(big.Int).Add(alignedA, alignedB), scale))
		requireNum(t, x.Sub(y), scaledIntegerText(new(big.Int).Sub(alignedA, alignedB), scale))
		requireNum(t, x.Mul(y), scaledIntegerText(new(big.Int).Mul(bx, by), as+bs))
		if got, want := x.Cmp(y), alignedA.Cmp(alignedB); got != want {
			t.Fatalf("comparison = %d, want %d", got, want)
		}
		if b != 0 {
			requireNum(t, x.Div(y), rationalDivisionText(a, as, b, bs))
		}
	})
}

func decimalPower(power int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(power)), nil)
}

// An independent rational oracle for the 34-digit precision selected by int64
// coefficients. This does not use apd, Num's backend selection, or its reducer.
func rationalDivisionText(a int64, as int, b int64, bs int) string {
	if a == 0 {
		return "0"
	}
	numerator := new(big.Int).Mul(new(big.Int).Abs(big.NewInt(a)), decimalPower(bs))
	denominator := new(big.Int).Mul(new(big.Int).Abs(big.NewInt(b)), decimalPower(as))
	exponent := len(numerator.String()) - len(denominator.String())
	comparison := 0
	if exponent >= 0 {
		comparison = numerator.Cmp(new(big.Int).Mul(denominator, decimalPower(exponent)))
	} else {
		comparison = new(big.Int).Mul(numerator, decimalPower(-exponent)).Cmp(denominator)
	}
	if comparison < 0 {
		exponent--
	}
	scale := 33 - exponent
	if scale >= 0 {
		numerator.Mul(numerator, decimalPower(scale))
	} else {
		denominator.Mul(denominator, decimalPower(-scale))
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	comparison = remainder.Lsh(remainder, 1).Cmp(denominator)
	if comparison > 0 || comparison == 0 && quotient.Bit(0) != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if scale < 0 {
		quotient.Mul(quotient, decimalPower(-scale))
		scale = 0
	}
	if (a < 0) != (b < 0) {
		quotient.Neg(quotient)
	}
	text := scaledIntegerText(quotient, scale)
	if scale > 0 {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	return text
}

func FuzzNumJSONGrammar(f *testing.F) {
	for _, input := range []string{`123.4500`, `"123.4500"`, `"123\u002e4500"`, `01`, `1.`, `1.e2`, `null`, `"1` + "\n" + `2"`, `"-0.00"`} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > maxInputBytes*6+2 {
			return
		}
		got := FromInt64(42)
		if err := got.UnmarshalJSON([]byte(input)); err != nil {
			t.Fatal(err)
		}
		if !json.Valid([]byte(input)) {
			if got.Ok() {
				t.Fatalf("accepted malformed JSON %q", input)
			}
			return
		}
		text := strings.Trim(input, " \t\r\n")
		if strings.HasPrefix(text, `"`) {
			if err := json.Unmarshal([]byte(text), &text); err != nil {
				t.Fatal(err)
			}
		}
		want := FromString(text)
		if got.Ok() != want.Ok() {
			t.Fatalf("JSON %q produced %v, direct decoded text produced %v", input, got, want)
		}
		if want.Ok() {
			requireNum(t, got, want.String())
		}
	})
}

func FuzzNumFloatRoundtrip(f *testing.F) {
	for _, value := range []float64{0, -0.1, 123.456789, math.SmallestNonzeroFloat64, math.MaxFloat64, math.Inf(1), math.NaN()} {
		f.Add(math.Float64bits(value))
	}
	f.Fuzz(func(t *testing.T, bits uint64) {
		value := math.Float64frombits(bits)
		got := FromFloat64(value)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			if got.Ok() || !errors.Is(got.Err, ErrInvalidNumericString) {
				t.Fatalf("accepted nonfinite float: %v", got)
			}
			return
		}
		want := strconv.FormatFloat(value, 'f', -1, 64)
		if value == 0 {
			want = "0"
		}
		requireNum(t, got, want)
	})
}
