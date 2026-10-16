// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func requireNum(t *testing.T, got Num, want string) {
	t.Helper()
	if !got.Ok() {
		t.Fatalf("invalid Num: %v; want %s", got.Err, want)
	}
	if got.String() != want {
		t.Fatalf("got %q, want %q", got.String(), want)
	}
}

func requireValue(t *testing.T, got Num, want string) {
	t.Helper()
	expected := FromString(want)
	if !got.Ok() || !expected.Ok() || !got.Equal(expected) {
		t.Fatalf("got %s (%v), want value %s", got, got.Err, want)
	}
}

func TestFromInt64(t *testing.T) {
	for _, value := range []int64{0, 1, -1, 1_000_000_000_000, math.MinInt64, math.MaxInt64} {
		t.Run(strconv.FormatInt(value, 10), func(t *testing.T) {
			requireNum(t, FromInt64(value), strconv.FormatInt(value, 10))
		})
	}
}

func TestFromFloat64(t *testing.T) {
	for _, value := range []float64{0, math.Copysign(0, -1), 123.45, -99.99, 0.1, 1e-20, 1e20, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		t.Run(strconv.FormatFloat(value, 'g', -1, 64), func(t *testing.T) {
			want := strconv.FormatFloat(value, 'f', -1, 64)
			if value == 0 {
				want = "0"
			}
			requireNum(t, FromFloat64(value), want)
		})
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		n := FromFloat64(value)
		if n.Ok() || !errors.Is(n.Err, ErrInvalidNumericString) {
			t.Errorf("FromFloat64(%v) = %v, want invalid numeric string", value, n)
		}
	}
}

func TestFromString(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"100", "100"}, {"123.4500", "123.4500"}, {"-50.5", "-50.5"},
		{"+1.25", "1.25"}, {".5", "0.5"}, {"1.", "1"}, {"00012.3400", "12.3400"},
		{"1e3", "1000"}, {"1.2300e1", "12.300"}, {"1E-8", "0.00000001"},
		{"-0.000", "0.000"}, {"0", "0"},
		{"10000000000000.000001", "10000000000000.000001"},
		{"1.2345665000000000001", "1.2345665000000000001"},
	} {
		t.Run(tc.input, func(t *testing.T) { requireNum(t, FromString(tc.input), tc.want) })
	}
}

func TestFromStringRejectsInvalidGrammar(t *testing.T) {
	for _, input := range []string{"", "+", "-", ".", "e1", "1e", "1e+", "1.2.3", " 1", "1 ", "1\n", "1_000", "NaN", "sNaN", "Inf", "Infinity", "-Infinity", "0x10", "1/2"} {
		t.Run(input, func(t *testing.T) {
			n := FromString(input)
			if n.Ok() || !errors.Is(n.Err, ErrInvalidNumericString) {
				t.Fatalf("FromString(%q) = %v, want invalid numeric string", input, n)
			}
		})
	}
}

func TestZeroAndUninitialized(t *testing.T) {
	requireNum(t, Zero(), "0")
	if !Zero().IsZero() {
		t.Fatal("Zero should report zero")
	}
	var n Num
	if n.Ok() || n.IsZero() {
		t.Fatal("the zero-value Num must be uninitialized")
	}
	for _, result := range []Num{n.Add(Zero()), Zero().Sub(n), n.Mul(FromInt64(1)), FromInt64(1).Div(n), n.Neg(), n.Abs(), n.Quantize(0)} {
		if result.Ok() || !errors.Is(result.Err, ErrUninitialized) {
			t.Fatalf("uninitialized arithmetic = %v, want ErrUninitialized", result)
		}
	}
	for _, marshal := range []func() error{
		func() error { _, err := n.MarshalJSON(); return err },
		func() error { _, err := n.MarshalText(); return err },
		func() error { _, err := n.Value(); return err },
	} {
		if err := marshal(); !errors.Is(err, ErrMarshalErrorState) || !errors.Is(err, ErrUninitialized) {
			t.Fatalf("marshal uninitialized Num = %v, want both sentinels", err)
		}
	}
}

func TestExactArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  Num
		want string
	}{
		{"decimal sum", FromString("0.1").Add(FromString("0.2")), "0.3"},
		{"mixed-scale sum", FromString("1.20").Add(FromString("0.003")), "1.203"},
		{"difference", FromString("10.5").Sub(FromString("3.2")), "7.3"},
		{"product", FromString("2.5").Mul(FromString("4.0")), "10"},
		{"tiny quantity", FromString("0.00000001").Mul(FromInt64(1_000_000)), "0.01"},
		{"large sum", FromString("10000000000000").Add(FromString("0.000001")), "10000000000000.000001"},
		{"large product", FromString("9999999999999").Mul(FromString("9999999999999")), "99999999999980000000000001"},
		{"negate", FromString("5.50").Neg(), "-5.5"},
		{"absolute", FromString("-5.50").Abs(), "5.5"},
	} {
		t.Run(tc.name, func(t *testing.T) { requireValue(t, tc.got, tc.want) })
	}
}

func TestArithmeticDoesNotDoubleRound(t *testing.T) {
	product := FromString("10000000.000001").Mul(FromString("2.500001"))
	requireNum(t, product, "25000010.000002500001")
	requireNum(t, product.Quantize(6), "25000010.000003")
	requireNum(t, FromString("1.2345665000000000001").Quantize(6), "1.234567")
}

func TestArithmeticRetainsNaturalScale(t *testing.T) {
	requireNum(t, FromString("1.20").Add(FromString("3.4")), "4.60")
	requireNum(t, FromString("5.000").Sub(FromString("1.2")), "3.800")
	requireNum(t, FromString("1.20").Mul(FromString("3.0")), "3.600")
	requireNum(t, FromString("5.50").Neg(), "-5.50")
	requireNum(t, FromString("-5.50").Abs(), "5.50")
}

func TestDivision(t *testing.T) {
	for _, tc := range []struct{ a, b, want string }{
		{"10.0", "4.0", "2.5"}, {"1", "2", "0.5"}, {"1000", "1", "1000"},
		{"1", "3", "0." + strings.Repeat("3", 34)},
		{"2", "3", "0." + strings.Repeat("6", 33) + "7"},
		{"-1", "3", "-0." + strings.Repeat("3", 34)},
		{"1", "-3", "-0." + strings.Repeat("3", 34)},
		{"1.0000000000000000000000000000000000000000", "3.0000", "0." + strings.Repeat("3", 34)},
		{"1", "1125899906842624", "0." + strings.Repeat("0", 15) + "8881784197001252323389053344726562"},
		{"3", "562949953421312", "0." + strings.Repeat("0", 14) + "5329070518200751394033432006835938"},
	} {
		t.Run(tc.a+"/"+tc.b, func(t *testing.T) { requireNum(t, FromString(tc.a).Div(FromString(tc.b)), tc.want) })
	}
	precise := "1." + strings.Repeat("23456789", 6)
	requireNum(t, FromString(precise).Div(FromInt64(1)), precise)
	requireNum(t, FromString(strings.Repeat("9", 50)).Div(FromInt64(1)), strings.Repeat("9", 50))
	requireNum(t, FromString("1e-4096").Div(FromInt64(1)), "0."+strings.Repeat("0", 4095)+"1")
	for _, zero := range []Num{Zero(), FromString("-0.00")} {
		if result := FromInt64(1).Div(zero); !errors.Is(result.Err, ErrDivisionByZero) {
			t.Fatalf("division by zero = %v, want ErrDivisionByZero", result)
		}
	}
}

func TestQuantizeHalfEven(t *testing.T) {
	for _, tc := range []struct {
		input string
		scale int
		want  string
	}{
		{"1.225", 2, "1.22"}, {"1.235", 2, "1.24"}, {"-1.225", 2, "-1.22"}, {"-1.235", 2, "-1.24"},
		{"2.5", 0, "2"}, {"3.5", 0, "4"}, {"-2.5", 0, "-2"}, {"-3.5", 0, "-4"},
		{"0.005", 2, "0.00"}, {"-0.005", 2, "0.00"}, {"0.015", 2, "0.02"}, {"-0.015", 2, "-0.02"},
		{"1.2", 4, "1.2000"}, {"1.2349", 2, "1.23"}, {"9.995", 2, "10.00"},
	} {
		t.Run(tc.input+"/"+strconv.Itoa(tc.scale), func(t *testing.T) { requireNum(t, FromString(tc.input).Quantize(tc.scale), tc.want) })
	}
	for _, scale := range []int{-1, 4097, math.MaxInt} {
		if result := FromInt64(1).Quantize(scale); !errors.Is(result.Err, ErrInvalidScale) {
			t.Errorf("Quantize(%d) = %v, want ErrInvalidScale", scale, result)
		}
	}
	requireNum(t, FromInt64(1).Quantize(4095), "1."+strings.Repeat("0", 4095))
	if result := FromInt64(1).Quantize(4096); !errors.Is(result.Err, ErrLimitExceeded) {
		t.Fatalf("Quantize needing 4097 coefficient digits = %v, want limit error", result)
	}
}

func TestNumericLimits(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"coefficient boundary", strings.Repeat("9", 4096), true},
		{"coefficient overflow", "1." + strings.Repeat("1", 4096), false},
		{"integer boundary", "1e4095", true}, {"integer overflow", "1e4096", false},
		{"scale boundary", "1e-4096", true}, {"scale overflow", "1e-4097", false},
		{"input boundary", strings.Repeat("0", 16383) + "1", true},
		{"input overflow", strings.Repeat("0", 16384) + "1", false},
		{"huge positive exponent", "1e" + strings.Repeat("9", 100), false},
		{"huge negative exponent", "1e-" + strings.Repeat("9", 100), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := FromString(tc.input)
			if tc.valid {
				if !n.Ok() {
					t.Fatalf("boundary rejected: %v", n.Err)
				}
			} else if n.Ok() || !errors.Is(n.Err, ErrLimitExceeded) {
				t.Fatalf("out-of-range value = %v, want ErrLimitExceeded", n)
			}
		})
	}
	for _, tc := range []struct {
		name string
		got  Num
	}{
		{"sum integer overflow", FromString(strings.Repeat("9", 4096)).Add(FromInt64(1))},
		{"sum coefficient overflow", FromString("1e4095").Add(FromString("0.1"))},
		{"product coefficient overflow", FromString("9." + strings.Repeat("9", 4095)).Mul(FromInt64(9))},
		{"product scale overflow", FromString("1e-2048").Mul(FromString("1e-2049"))},
		{"division scale overflow", FromString("1e-4096").Div(FromInt64(3))},
		{"division integer overflow", FromString("1e4095").Div(FromString("0.1"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.got.Err, ErrLimitExceeded) {
				t.Fatalf("got %v, want ErrLimitExceeded", tc.got)
			}
		})
	}
	// An alignment temporary may exceed stored precision if cancellation fits.
	requireNum(t, FromString("1e4095").Sub(FromString(strings.Repeat("9", 4095)+".9")), "0.1")
}

func TestComparisons(t *testing.T) {
	a, same, larger := FromString("100.00"), FromString("100.0"), FromString("100.001")
	if !a.Equal(same) || a.Cmp(same) != 0 || a.LessThan(same) || a.GreaterThan(same) {
		t.Fatal("comparison should ignore trailing fractional zeros")
	}
	if a.Cmp(larger) != -1 || larger.Cmp(a) != 1 || !a.LessThan(larger) || !larger.GreaterThan(a) {
		t.Fatal("mixed-scale comparison has wrong ordering")
	}
}

func TestErrorPropagation(t *testing.T) {
	bad, good := FromString("bad"), FromInt64(5)
	for _, tc := range []struct {
		name string
		fn   func() Num
	}{
		{"Add left", func() Num { return bad.Add(good) }}, {"Add right", func() Num { return good.Add(bad) }},
		{"Sub left", func() Num { return bad.Sub(good) }}, {"Sub right", func() Num { return good.Sub(bad) }},
		{"Mul left", func() Num { return bad.Mul(good) }}, {"Mul right", func() Num { return good.Mul(bad) }},
		{"Div left", func() Num { return bad.Div(good) }}, {"Div right", func() Num { return good.Div(bad) }},
		{"Neg", bad.Neg}, {"Abs", bad.Abs}, {"Quantize", func() Num { return bad.Quantize(2) }},
		{"invalid scale after error", func() Num { return bad.Quantize(-1) }},
		{"chain", func() Num { return good.Add(bad).Mul(good).Sub(good) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(); got.Ok() || !errors.Is(got.Err, bad.Err) {
				t.Fatalf("got error %v, want original error %v", got.Err, bad.Err)
			}
		})
	}
	if result := good.Add(good).Div(Zero()).Sub(good); !errors.Is(result.Err, ErrDivisionByZero) {
		t.Fatalf("division error lost through chain: %v", result.Err)
	}
	if got := bad.Add(Num{Err: ErrUnsupportedType}); !errors.Is(got.Err, bad.Err) {
		t.Fatal("receiver error must precede operand error")
	}
}

func TestArithmeticDoesNotMutateAliases(t *testing.T) {
	original := FromString("1234567890." + strings.Repeat("123456789", 6))
	alias, before := original, original.String()
	for _, result := range []Num{
		original.Add(FromInt64(1)), original.Sub(FromInt64(1)), original.Mul(FromInt64(2)),
		original.Div(FromInt64(3)), original.Neg(), original.Abs(), original.Quantize(2),
	} {
		if !result.Ok() {
			t.Fatal(result.Err)
		}
	}
	if original.String() != before || alias.String() != before {
		t.Fatal("arithmetic mutated a copied Num's decimal storage")
	}
	if err := alias.UnmarshalText([]byte("5.00")); err != nil {
		t.Fatal(err)
	}
	requireNum(t, original, before)
	if err := alias.Scan(struct{}{}); err == nil {
		t.Fatal("expected unsupported scan input")
	}
	requireNum(t, original, before)
}

func TestConcurrentArithmeticOnSharedNum(t *testing.T) {
	original := FromString("1234567890." + strings.Repeat("123456789", 6))
	before := original.String()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 30 {
				result := original.Add(FromInt64(1)).Sub(FromInt64(1))
				if !result.Ok() || !result.Equal(original) {
					t.Errorf("shared arithmetic result = %v", result)
					return
				}
				if !original.Div(FromInt64(1)).Equal(original) || original.String() != before {
					t.Error("concurrent arithmetic changed the shared value")
					return
				}
			}
		})
	}
	wg.Wait()
}

func TestJSONCodecs(t *testing.T) {
	for _, input := range []string{`123.4500`, `"123.4500"`, `"123\u002e4500"`} {
		t.Run(input, func(t *testing.T) {
			var n Num
			if err := json.Unmarshal([]byte(input), &n); err != nil {
				t.Fatal(err)
			}
			requireNum(t, n, "123.4500")
			data, err := json.Marshal(n)
			if err != nil || string(data) != `123.4500` {
				t.Fatalf("Marshal = %s, %v", data, err)
			}
		})
	}
	var response struct {
		Price Num `json:"price"`
		Bad   Num `json:"bad"`
		After Num `json:"after"`
	}
	if err := json.Unmarshal([]byte(`{"price":42.50,"bad":"bad","after":"65.00"}`), &response); err != nil {
		t.Fatal(err)
	}
	requireNum(t, response.Price, "42.50")
	requireNum(t, response.After, "65.00")
	if !errors.Is(response.Bad.Err, ErrInvalidNumericString) {
		t.Fatalf("partial JSON parse lost numeric error: %v", response.Bad.Err)
	}
}

func TestJSONFailureClearsStateAndCanRecover(t *testing.T) {
	for _, input := range []string{`"bad"`, `null`, `true`, `[]`, `{}`, `"NaN"`, `"Infinity"`, `"1_000"`, `"1\u0020"`, `"1.2`} {
		t.Run(input, func(t *testing.T) {
			n := FromString("123.45")
			if err := n.UnmarshalJSON([]byte(input)); err != nil {
				t.Fatalf("numeric JSON failure must stay in Num.Err: %v", err)
			}
			if n.Ok() || n.dec != nil || !errors.Is(n.Err, ErrInvalidNumericString) {
				t.Fatalf("invalid JSON retained state: %v", n)
			}
			if _, err := n.MarshalJSON(); !errors.Is(err, ErrMarshalErrorState) {
				t.Fatalf("error-state marshal = %v", err)
			}
			if err := n.UnmarshalJSON([]byte(`"1.2300"`)); err != nil {
				t.Fatal(err)
			}
			requireNum(t, n, "1.2300")
		})
	}
}

func TestJSONDecodedInputLimits(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        error
	}{
		{"escaped coefficient boundary", `"` + strings.Repeat(`\u0039`, 4096) + `"`, nil},
		{"escaped coefficient overflow", `"` + strings.Repeat(`\u0039`, 4097) + `"`, ErrLimitExceeded},
		{"escaped input boundary", `"` + strings.Repeat(`\u0030`, 16383) + `\u0031"`, nil},
		{"escaped input overflow", `"` + strings.Repeat(`\u0030`, 16384) + `\u0031"`, ErrLimitExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var n Num
			if err := n.UnmarshalJSON([]byte(tc.input)); err != nil {
				t.Fatal(err)
			}
			if tc.want == nil {
				if !n.Ok() {
					t.Fatal(n.Err)
				}
			} else if n.Ok() || !errors.Is(n.Err, tc.want) {
				t.Fatalf("got %v, want %v", n.Err, tc.want)
			}
		})
	}
}

func TestPresentationDoesNotChangeCodecs(t *testing.T) {
	n := FromString("123.4567890000123456789000")
	if got := fmt.Sprintf("%.2f", n); got != "123.46" {
		t.Fatal(got)
	}
	want := "123.4567890000123456789000"
	jsonBytes, err := json.Marshal(n)
	if err != nil || string(jsonBytes) != want {
		t.Fatalf("JSON after display = %s, %v", jsonBytes, err)
	}
	text, err := n.MarshalText()
	if err != nil || string(text) != want {
		t.Fatalf("text after display = %s, %v", text, err)
	}
	if value, err := n.Value(); err != nil || value != want {
		t.Fatalf("SQL value after display = %v, %v", value, err)
	}
	requireNum(t, n, want)
}

func TestTextAndXMLCodecs(t *testing.T) {
	original := FromString("456.78900001")
	data, err := original.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var restored Num
	if err := restored.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	requireNum(t, restored, original.String())
	var record struct {
		Price Num `xml:"price"`
	}
	if err := xml.Unmarshal([]byte(`<position><price>42.50000001</price></position>`), &record); err != nil {
		t.Fatal(err)
	}
	requireNum(t, record.Price, "42.50000001")
	if err := restored.UnmarshalText([]byte("bad")); !errors.Is(err, ErrInvalidNumericString) || restored.dec != nil || restored.Ok() {
		t.Fatalf("invalid text retained state or lost error: %v", err)
	}
	if err := restored.UnmarshalText([]byte("1.00")); err != nil {
		t.Fatal(err)
	}
	requireNum(t, restored, "1.00")
}

func TestScanAndValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  any
		want string
	}{
		{"string", "123.4500", "123.4500"}, {"bytes", []byte("99.9901"), "99.9901"},
		{"int64", int64(42), "42"}, {"minimum int64", int64(math.MinInt64), "-9223372036854775808"},
		{"uint64", uint64(math.MaxUint64), "18446744073709551615"},
		{"float64", float64(3.14), "3.14"}, {"float32", float32(0.1), "0.10000000149011612"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var n Num
			if err := n.Scan(tc.src); err != nil {
				t.Fatal(err)
			}
			requireNum(t, n, tc.want)
			if value, err := n.Value(); err != nil || value != tc.want {
				t.Fatalf("Value = %v, %v; want %q", value, err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		src  any
		want error
	}{{nil, ErrUnsupportedType}, {struct{}{}, ErrUnsupportedType}, {"bad", ErrInvalidNumericString}, {math.NaN(), ErrInvalidNumericString}} {
		n := FromString("123.45")
		if err := n.Scan(tc.src); !errors.Is(err, tc.want) || n.dec != nil || n.Ok() {
			t.Fatalf("Scan(%v) retained state or lost error: %v", tc.src, err)
		}
		if _, err := n.Value(); !errors.Is(err, ErrMarshalErrorState) || !errors.Is(err, tc.want) {
			t.Fatalf("error-state Value = %v", err)
		}
		if err := n.Scan("42.5000"); err != nil {
			t.Fatal(err)
		}
		requireNum(t, n, "42.5000")
	}
}

func TestSentinelErrorsDistinct(t *testing.T) {
	errs := []error{ErrInvalidNumericString, ErrUnsupportedType, ErrMarshalErrorState, ErrUninitialized, ErrInvalidScale, ErrLimitExceeded, ErrDivisionByZero}
	for i := range errs {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Fatalf("sentinel errors %v and %v must be distinct", errs[i], errs[j])
			}
		}
	}
}

func TestNumPublicAPIGuardrails(t *testing.T) {
	numType, ptrType := reflect.TypeFor[Num](), reflect.TypeFor[*Num]()
	if numType.Comparable() {
		t.Fatal("Num should not be comparable")
	}
	for _, name := range []string{"Round", "Scale", "Float64", "Sign", "Quo", "Less", "Set", "SetString"} {
		if _, ok := numType.MethodByName(name); ok {
			t.Fatalf("Num should not expose backend method %q", name)
		}
	}
	for _, name := range []string{"Add", "Sub", "Mul", "Div", "Neg", "Abs", "Quantize", "Ok", "IsZero", "Cmp", "Equal", "LessThan", "GreaterThan", "String", "MarshalJSON", "MarshalText", "Value"} {
		if _, ok := numType.MethodByName(name); !ok {
			t.Fatalf("Num should expose method %q", name)
		}
	}
	for _, name := range []string{"UnmarshalJSON", "UnmarshalText", "Scan"} {
		if _, ok := ptrType.MethodByName(name); !ok {
			t.Fatalf("*Num should expose method %q", name)
		}
	}
	for i := range numType.NumField() {
		field := numType.Field(i)
		if field.Type.Kind() == reflect.String {
			t.Fatalf("Num must not store a string field: %s", field.Name)
		}
		if field.IsExported() && field.Name != "Err" {
			t.Fatalf("Num must not expose mutable backend storage: %s", field.Name)
		}
	}
}
