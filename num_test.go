// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/govalues/decimal"
)

func scaleOf(n Num) int {
	return n.dec.Scale()
}

// --- Constructors ---

func TestFrom(t *testing.T) {
	d, err := decimal.Parse("123.45")
	if err != nil {
		t.Fatal(err)
	}
	n := From(d)
	if !n.Ok() {
		t.Fatalf("From() error: %v", n.Err)
	}
	if got := scaleOf(n); got != Scale() {
		t.Errorf("scale = %d, want %d", got, Scale())
	}
	want := FromString("123.45")
	if !n.Equal(want) {
		t.Errorf("From() = %s, want %s", n.dec.String(), want.dec.String())
	}
}

func TestFromInt64(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "0.000000"},
		{"positive", 100, "100.000000"},
		{"negative", -50, "-50.000000"},
		{"large", 1000000, "1000000.000000"},
		{"trillion", 1000000000000, "1000000000000.000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromInt64(tt.input)
			if !n.Ok() {
				t.Fatalf("FromInt64(%d) error: %v", tt.input, n.Err)
			}
			if got := n.dec.String(); got != tt.want {
				t.Errorf("FromInt64(%d) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"simple", 123.45, "123.450000"},
		{"zero", 0.0, "0.000000"},
		{"negative", -99.99, "-99.990000"},
		{"small", 0.001, "0.001000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromFloat64(tt.input)
			if !n.Ok() {
				t.Fatalf("FromFloat64(%v) error: %v", tt.input, n.Err)
			}
			if got := n.dec.String(); got != tt.want {
				t.Errorf("FromFloat64(%v) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromFloat64_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"NaN", math.NaN()},
		{"+Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromFloat64(tt.input)
			if n.Ok() {
				t.Errorf("FromFloat64(%v) expected error", tt.input)
			}
			if !errors.Is(n.Err, ErrInvalidNumericString) {
				t.Errorf("expected ErrInvalidNumericString, got %v", n.Err)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"integer", "100", "100.000000"},
		{"decimal", "123.45", "123.450000"},
		{"negative", "-50.5", "-50.500000"},
		{"zero", "0", "0.000000"},
		{"leading zero", "0.123", "0.123000"},
		{"exact scale", "123.456789", "123.456789"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromString(tt.input)
			if !n.Ok() {
				t.Fatalf("FromString(%q) error: %v", tt.input, n.Err)
			}
			if got := n.dec.String(); got != tt.want {
				t.Errorf("FromString(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromString_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"text", "not_a_number"},
		{"special", "NaN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromString(tt.input)
			if n.Ok() {
				t.Errorf("FromString(%q) expected error, got %s", tt.input, n.dec.String())
			}
			if !errors.Is(n.Err, ErrInvalidNumericString) {
				t.Errorf("expected ErrInvalidNumericString, got %v", n.Err)
			}
		})
	}
}

func TestZero(t *testing.T) {
	n := Zero()
	if !n.Ok() {
		t.Fatalf("Zero() error: %v", n.Err)
	}
	if !n.IsZero() {
		t.Error("Zero().IsZero() = false, want true")
	}
	if got := scaleOf(n); got != Scale() {
		t.Errorf("Zero() scale = %d, want %d", got, Scale())
	}
}

// --- Arithmetic ---

func TestArithmeticCorrectness(t *testing.T) {
	// Acceptance criterion 1: 0.1 + 0.2 == 0.3 (float64 fails this)
	result := FromString("0.1").Add(FromString("0.2"))
	if !result.Ok() {
		t.Fatalf("0.1 + 0.2 error: %v", result.Err)
	}
	expected := FromString("0.3")
	if !result.Equal(expected) {
		t.Errorf("0.1 + 0.2 = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{"positive", "10.5", "20.3", "30.800000"},
		{"with negative", "10.0", "-3.5", "6.500000"},
		{"with zero", "5.0", "0", "5.000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromString(tt.a).Add(FromString(tt.b))
			if !result.Ok() {
				t.Fatalf("Add error: %v", result.Err)
			}
			if got := result.dec.String(); got != tt.want {
				t.Errorf("%s + %s = %s, want %s", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSub(t *testing.T) {
	result := FromString("10.5").Sub(FromString("3.2"))
	if !result.Ok() {
		t.Fatalf("Sub error: %v", result.Err)
	}
	expected := FromString("7.3")
	if !result.Equal(expected) {
		t.Errorf("10.5 - 3.2 = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

func TestMul(t *testing.T) {
	result := FromString("2.5").Mul(FromString("4.0"))
	if !result.Ok() {
		t.Fatalf("Mul error: %v", result.Err)
	}
	expected := FromString("10.0")
	if !result.Equal(expected) {
		t.Errorf("2.5 * 4.0 = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

func TestDiv(t *testing.T) {
	result := FromString("10.0").Div(FromString("4.0"))
	if !result.Ok() {
		t.Fatalf("Div error: %v", result.Err)
	}
	expected := FromString("2.5")
	if !result.Equal(expected) {
		t.Errorf("10.0 / 4.0 = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

func TestNeg(t *testing.T) {
	result := FromString("5.5").Neg()
	if !result.Ok() {
		t.Fatalf("Neg error: %v", result.Err)
	}
	expected := FromString("-5.5")
	if !result.Equal(expected) {
		t.Errorf("Neg(5.5) = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

func TestAbs(t *testing.T) {
	result := FromString("-5.5").Abs()
	if !result.Ok() {
		t.Fatalf("Abs error: %v", result.Err)
	}
	expected := FromString("5.5")
	if !result.Equal(expected) {
		t.Errorf("Abs(-5.5) = %s, want %s", result.dec.String(), expected.dec.String())
	}
}

// --- Comparisons ---

func TestEqual(t *testing.T) {
	a := FromString("100.00")
	b := FromString("100.0")
	if !a.Equal(b) {
		t.Errorf("%s should equal %s (both rescaled to scale %d)", a.dec.String(), b.dec.String(), Scale())
	}
}

func TestCmp(t *testing.T) {
	a := FromString("10.0")
	b := FromString("20.0")
	if a.Cmp(b) >= 0 {
		t.Error("10.0 should be less than 20.0")
	}
	if b.Cmp(a) <= 0 {
		t.Error("20.0 should be greater than 10.0")
	}
	c := FromString("10.0")
	if a.Cmp(c) != 0 {
		t.Error("10.0 should equal 10.0")
	}
}

func TestLessThan(t *testing.T) {
	a := FromString("5.0")
	b := FromString("10.0")
	if !a.LessThan(b) {
		t.Error("5.0 should be less than 10.0")
	}
	if b.LessThan(a) {
		t.Error("10.0 should not be less than 5.0")
	}
}

func TestGreaterThan(t *testing.T) {
	a := FromString("10.0")
	b := FromString("5.0")
	if !a.GreaterThan(b) {
		t.Error("10.0 should be greater than 5.0")
	}
	if b.GreaterThan(a) {
		t.Error("5.0 should not be greater than 10.0")
	}
}

// --- Error Propagation ---

func TestErrorPropagation(t *testing.T) {
	bad := FromString("not_a_number")
	if bad.Ok() {
		t.Fatal("expected error from bad input")
	}

	// Error on operand propagates
	result := FromString("10.0").Add(bad)
	if result.Ok() {
		t.Error("Add with error operand should propagate error")
	}
	if !errors.Is(result.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", result.Err)
	}

	// Error on receiver propagates
	result = bad.Sub(FromString("5.0"))
	if result.Ok() {
		t.Error("Sub on error receiver should propagate error")
	}

	// Error through chain
	result = bad.Mul(FromString("2.0")).Add(FromString("1.0"))
	if result.Ok() {
		t.Error("chained ops with error should propagate")
	}
}

func TestErrorPropagation_AllOps(t *testing.T) {
	bad := FromString("bad")
	good := FromString("5.0")

	ops := []struct {
		name string
		fn   func() Num
	}{
		{"Add(bad)", func() Num { return good.Add(bad) }},
		{"bad.Add", func() Num { return bad.Add(good) }},
		{"Sub(bad)", func() Num { return good.Sub(bad) }},
		{"bad.Sub", func() Num { return bad.Sub(good) }},
		{"Mul(bad)", func() Num { return good.Mul(bad) }},
		{"bad.Mul", func() Num { return bad.Mul(good) }},
		{"Div(bad)", func() Num { return good.Div(bad) }},
		{"bad.Div", func() Num { return bad.Div(good) }},
		{"bad.Neg", bad.Neg},
		{"bad.Abs", bad.Abs},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			result := op.fn()
			if result.Ok() {
				t.Error("expected error propagation")
			}
		})
	}
}

// --- Scale Enforcement ---

func TestScaleEnforcement(t *testing.T) {
	tests := []struct {
		name string
		n    Num
	}{
		{"FromInt64", FromInt64(42)},
		{"FromFloat64", FromFloat64(3.14)},
		{"FromString", FromString("99.99")},
		{"Zero", Zero()},
		{"From", From(decimal.MustNew(12345, 2))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.n.Ok() {
				t.Fatalf("constructor error: %v", tt.n.Err)
			}
			if got := scaleOf(tt.n); got != Scale() {
				t.Errorf("scale = %d, want %d", got, Scale())
			}
		})
	}
}

// --- JSON Serialization ---

func TestJSONRoundtrip(t *testing.T) {
	original := FromString("123.456789")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored Num
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Ok() {
		t.Fatalf("unmarshal error: %v", restored.Err)
	}
	if !original.Equal(restored) {
		t.Errorf("roundtrip: got %s, want %s", restored.dec.String(), original.dec.String())
	}
}

func TestJSONMarshalErrorState(t *testing.T) {
	bad := FromString("not_a_number")
	_, err := json.Marshal(bad)
	if err == nil {
		t.Error("MarshalJSON on error-state Num should return error")
	}
}

func TestUnmarshalJSON_BareNumeric(t *testing.T) {
	var n Num
	if err := n.UnmarshalJSON([]byte("123.45")); err != nil {
		t.Fatal(err)
	}
	if !n.Ok() {
		t.Fatalf("unmarshal error: %v", n.Err)
	}
	expected := FromString("123.45")
	if !n.Equal(expected) {
		t.Errorf("bare numeric: got %s, want %s", n.dec.String(), expected.dec.String())
	}
}

func TestUnmarshalJSON_QuotedString(t *testing.T) {
	var n Num
	if err := n.UnmarshalJSON([]byte(`"123.45"`)); err != nil {
		t.Fatal(err)
	}
	if !n.Ok() {
		t.Fatalf("unmarshal error: %v", n.Err)
	}
	expected := FromString("123.45")
	if !n.Equal(expected) {
		t.Errorf("quoted string: got %s, want %s", n.dec.String(), expected.dec.String())
	}
}

func TestUnmarshalJSON_Invalid(t *testing.T) {
	var n Num
	_ = n.UnmarshalJSON([]byte(`"not_a_number"`))
	if n.Ok() {
		t.Error("expected error for invalid input")
	}
	if !errors.Is(n.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", n.Err)
	}
}

func TestUnmarshalJSON_Null(t *testing.T) {
	var n Num
	_ = n.UnmarshalJSON([]byte("null"))
	if n.Ok() {
		t.Error("expected error for null on non-nullable Num")
	}
	if !errors.Is(n.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", n.Err)
	}
}

func TestOk_ZeroValueIsNotOk(t *testing.T) {
	var n Num
	if n.Ok() {
		t.Fatal("zero-value Num should not be ok")
	}
	if n.IsZero() {
		t.Fatal("zero-value Num should not report zero")
	}
}

// --- Text Serialization ---

func TestTextRoundtrip(t *testing.T) {
	original := FromString("456.789")
	data, err := original.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var restored Num
	if err := restored.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if !original.Equal(restored) {
		t.Errorf("text roundtrip: got %s, want %s", restored.dec.String(), original.dec.String())
	}
}

func TestUnmarshalText_FailurePoisonedState(t *testing.T) {
	n := FromString("123.45")
	if err := n.UnmarshalText([]byte("not-a-number")); err == nil {
		t.Fatal("expected error from invalid text")
	}
	if n.Ok() {
		t.Fatal("Num should not be ok after failed UnmarshalText")
	}
	if n.dec != (decimal.Decimal{}) {
		t.Fatal("Num should clear decimal state after failed UnmarshalText")
	}
}

// --- SQL ---

func TestScan(t *testing.T) {
	tests := []struct {
		name string
		src  any
		want string
	}{
		{"string", "123.45", "123.450000"},
		{"[]byte", []byte("99.99"), "99.990000"},
		{"int64", int64(42), "42.000000"},
		{"float64", float64(3.14), "3.140000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Num
			if err := n.Scan(tt.src); err != nil {
				t.Fatalf("Scan(%v) error: %v", tt.src, err)
			}
			if got := n.dec.String(); got != tt.want {
				t.Errorf("Scan(%v) = %s, want %s", tt.src, got, tt.want)
			}
		})
	}
}

func TestScan_Nil(t *testing.T) {
	var n Num
	if err := n.Scan(nil); err == nil {
		t.Error("Scan(nil) should return error for non-nullable Num")
	}
	if n.Ok() {
		t.Fatal("Num should not be ok after Scan(nil)")
	}
	if !errors.Is(n.Err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", n.Err)
	}
}

func TestScan_FailurePoisonedState(t *testing.T) {
	n := FromString("123.45")
	if err := n.Scan(struct{}{}); err == nil {
		t.Fatal("expected error from unsupported scan type")
	}
	if n.Ok() {
		t.Fatal("Num should not be ok after failed Scan")
	}
	if n.dec != (decimal.Decimal{}) {
		t.Fatal("Num should clear decimal state after failed Scan")
	}
}

func TestValue(t *testing.T) {
	n := FromString("123.45")
	v, err := n.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value() type = %T, want string", v)
	}
	if s != "123.450000" {
		t.Errorf("Value() = %q, want %q", s, "123.450000")
	}
}

func TestValue_ErrorState(t *testing.T) {
	bad := FromString("not_a_number")
	_, err := bad.Value()
	if err == nil {
		t.Error("Value() on error-state Num should return error")
	}
}

// --- Sentinel Errors ---

func TestSentinelErrors_Distinct(t *testing.T) {
	errs := []error{ErrInvalidNumericString, ErrIntegerOverflow, ErrUnsupportedType, ErrMarshalErrorState}
	for i := range errs {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Errorf("sentinel errors %v and %v should be distinct", errs[i], errs[j])
			}
		}
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	// Wrapped sentinel is detectable via errors.Is
	wrapped := FromString("bad")
	if !errors.Is(wrapped.Err, ErrInvalidNumericString) {
		t.Errorf("errors.Is should detect ErrInvalidNumericString in %v", wrapped.Err)
	}
}

// --- P1: Edge Cases ---

func TestDivByZero(t *testing.T) {
	result := FromInt64(1).Div(Zero())
	if result.Ok() {
		t.Error("division by zero should produce error-state Num")
	}
}

func TestEdge_VeryLargeValue(t *testing.T) {
	n := FromInt64(999999999999) // 12-digit integer
	if !n.Ok() {
		t.Fatalf("error: %v", n.Err)
	}
	if got := scaleOf(n); got != Scale() {
		t.Errorf("scale = %d, want %d", got, Scale())
	}
}

func TestEdge_VerySmallValue(t *testing.T) {
	// 7 decimal places → rounded to 6 by Rescale
	n := FromString("0.0000001")
	if !n.Ok() {
		t.Fatalf("error: %v", n.Err)
	}
	expected := Zero()
	if !n.Equal(expected) {
		t.Errorf("0.0000001 at scale 6 = %s, want %s (rounded to zero)", n.dec.String(), expected.dec.String())
	}
}

func TestEdge_HighPrecisionRounding(t *testing.T) {
	// "1.2345678" has 7 decimals → Rescale(6) rounds the last digit
	n := FromString("1.2345678")
	if !n.Ok() {
		t.Fatalf("error: %v", n.Err)
	}
	// 8 > 5, so 7 rounds up to 8
	expected := FromString("1.234568")
	if !n.Equal(expected) {
		t.Errorf("1.2345678 rounded = %s, want %s", n.dec.String(), expected.dec.String())
	}
}

func TestEdge_ArithmeticOverflow(t *testing.T) {
	// Multiplying large numbers that might overflow decimal precision
	large := FromString("9999999999999")
	result := large.Mul(large)
	// govalues/decimal handles overflow by returning error
	// (MaxPrec is 19, 13*2 = 26 digits > 19)
	if result.Ok() {
		// If it succeeds, at least verify it has correct scale
		if got := scaleOf(result); got != Scale() {
			t.Errorf("scale = %d, want %d", got, Scale())
		}
	}
	// Either outcome (error or truncated) is acceptable
}

func TestEdge_NegativeZero(t *testing.T) {
	// Ensure negative zero normalizes to zero
	n := FromString("-0")
	if !n.Ok() {
		t.Fatalf("error: %v", n.Err)
	}
	if !n.IsZero() {
		t.Errorf("-0 should be zero, got %s", n.dec.String())
	}
}

func TestEdge_MaxScaleParsing(t *testing.T) {
	// parseScale with boundary values
	if got := parseScale("19", 6); got != 19 {
		t.Errorf("parseScale(19) = %d, want 19", got)
	}
	if got := parseScale("20", 6); got != 6 {
		t.Errorf("parseScale(20) = %d, want 6 (fallback)", got)
	}
}

func TestEdge_ZeroScaleParsing(t *testing.T) {
	if got := parseScale("0", 6); got != 0 {
		t.Errorf("parseScale(0) = %d, want 0", got)
	}
	if got := parseScale("-1", 6); got != 6 {
		t.Errorf("parseScale(-1) = %d, want 6 (fallback)", got)
	}
}

// --- P1: Error Propagation Chains ---

func TestErrorChain_FirstOperand(t *testing.T) {
	bad := FromString("bad")
	good := FromString("10.0")
	result := bad.Add(good).Mul(good)
	if result.Ok() {
		t.Error("error from first operand should propagate through chain")
	}
	if !errors.Is(result.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", result.Err)
	}
}

func TestErrorChain_MiddleOperand(t *testing.T) {
	bad := FromString("bad")
	good := FromString("10.0")
	result := good.Add(bad).Mul(good)
	if result.Ok() {
		t.Error("error from middle operand should propagate through chain")
	}
	if !errors.Is(result.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", result.Err)
	}
}

func TestErrorChain_LastOperand(t *testing.T) {
	bad := FromString("bad")
	good := FromString("10.0")
	result := good.Add(good).Mul(bad)
	if result.Ok() {
		t.Error("error from last operand should propagate through chain")
	}
	if !errors.Is(result.Err, ErrInvalidNumericString) {
		t.Errorf("expected ErrInvalidNumericString, got %v", result.Err)
	}
}

func TestErrorChain_DivByZeroInChain(t *testing.T) {
	good := FromString("100.0")
	result := good.Add(good).Div(Zero()).Sub(good)
	if result.Ok() {
		t.Error("division by zero in chain should propagate")
	}
}

// --- P1: IBKR Mixed-Format JSON ---

func TestIBKR_MixedFormatJSON(t *testing.T) {
	type ibkrResponse struct {
		Price  Num `json:"price"`
		Bid    Num `json:"bid"`
		Ask    Num `json:"ask"`
		Volume Num `json:"volume"`
	}

	// IBKR sends both bare numbers and quoted strings in the same response
	input := `{"price": 123.45, "bid": "67.89", "ask": 70.50, "volume": "1000"}`

	var resp ibkrResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatal(err)
	}

	// All fields should parse successfully
	for _, tc := range []struct {
		name string
		got  Num
		want string
	}{
		{"price (bare)", resp.Price, "123.45"},
		{"bid (quoted)", resp.Bid, "67.89"},
		{"ask (bare)", resp.Ask, "70.50"},
		{"volume (quoted)", resp.Volume, "1000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.got.Ok() {
				t.Fatalf("parse error: %v", tc.got.Err)
			}
			expected := FromString(tc.want)
			if !tc.got.Equal(expected) {
				t.Errorf("got %s, want %s", tc.got.dec.String(), expected.dec.String())
			}
		})
	}

	// Bare and quoted should produce identical values
	barePrice := FromString("123.45")
	if !resp.Price.Equal(barePrice) {
		t.Errorf("bare 123.45 != FromString(123.45): %s vs %s",
			resp.Price.dec.String(), barePrice.dec.String())
	}
}

func TestIBKR_MixedFormatWithNullNum(t *testing.T) {
	type ibkrPosition struct {
		Price     Num     `json:"price"`
		Strike    NullNum `json:"strike"`
		CostBasis NullNum `json:"costBasis"`
	}

	input := `{"price": 42.50, "strike": "65.00", "costBasis": null}`

	var pos ibkrPosition
	if err := json.Unmarshal([]byte(input), &pos); err != nil {
		t.Fatal(err)
	}

	if !pos.Price.Ok() {
		t.Fatalf("price error: %v", pos.Price.Err)
	}
	if !pos.Strike.Valid || !pos.Strike.Num.Ok() {
		t.Fatalf("strike should be valid: Valid=%v, Err=%v", pos.Strike.Valid, pos.Strike.Num.Err)
	}
	if pos.CostBasis.Valid {
		t.Error("costBasis should be null")
	}
}

// --- P1: Stringer ---

func TestNum_String(t *testing.T) {
	n := FromString("42.50")
	if got := n.String(); got != "42.500000" {
		t.Errorf("String() = %q, want %q", got, "42.500000")
	}
}

func TestNum_String_Error(t *testing.T) {
	n := FromString("bad")
	got := n.String()
	if got == "0" || got == "" {
		t.Errorf("error-state String() should not be %q", got)
	}
	if len(got) < 7 { // should contain "<error: ...>"
		t.Errorf("error-state String() too short: %q", got)
	}
}

func TestNum_PublicAPIGuardrails(t *testing.T) {
	numType := reflect.TypeFor[Num]()
	ptrType := reflect.TypeFor[*Num]()
	if numType.Comparable() {
		t.Fatal("Num should not be comparable")
	}

	for _, methodName := range []string{"Round", "Scale", "Float64", "Sign", "Quo", "Less"} {
		if _, ok := numType.MethodByName(methodName); ok {
			t.Fatalf("Num should not promote decimal method %q", methodName)
		}
	}

	for _, methodName := range []string{
		"Add", "Sub", "Mul", "Div", "Neg", "Abs",
		"Ok", "IsZero", "Cmp", "Equal", "LessThan", "GreaterThan",
		"String", "MarshalJSON", "MarshalText", "Value",
	} {
		if _, ok := numType.MethodByName(methodName); !ok {
			t.Fatalf("Num should expose method %q", methodName)
		}
	}

	for _, methodName := range []string{"UnmarshalJSON", "UnmarshalText", "Scan"} {
		if _, ok := ptrType.MethodByName(methodName); !ok {
			t.Fatalf("*Num should expose method %q", methodName)
		}
	}
}
