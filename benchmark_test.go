// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"testing"

	"github.com/govalues/decimal"
)

func BenchmarkFromString(b *testing.B) {
	for b.Loop() {
		FromString("123.456789")
	}
}

func BenchmarkFromInt64(b *testing.B) {
	for b.Loop() {
		FromInt64(123456)
	}
}

func BenchmarkFromFloat64(b *testing.B) {
	for b.Loop() {
		FromFloat64(123.456789)
	}
}

func BenchmarkArithmeticChain(b *testing.B) {
	a := FromString("100.50")
	c := FromString("10")
	d := FromString("3")
	for b.Loop() {
		_ = a.Add(c).Mul(d).Sub(a).Div(c)
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		_, _ = json.Marshal(n)
	}
}

func BenchmarkUnmarshalJSON_Bare(b *testing.B) {
	data := []byte("123.456789")
	for b.Loop() {
		var n Num
		_ = n.UnmarshalJSON(data)
	}
}

func BenchmarkUnmarshalJSON_Quoted(b *testing.B) {
	data := []byte(`"123.456789"`)
	for b.Loop() {
		var n Num
		_ = n.UnmarshalJSON(data)
	}
}

func BenchmarkScan(b *testing.B) {
	for b.Loop() {
		var n Num
		_ = n.Scan("123.456789")
	}
}

func BenchmarkValue(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		_, _ = n.Value()
	}
}

func BenchmarkDecimalParseRescale(b *testing.B) {
	for b.Loop() {
		d, _ := decimal.Parse("123.456789")
		_ = d.Rescale(Scale())
	}
}

func BenchmarkDecimalNew(b *testing.B) {
	for b.Loop() {
		d, _ := decimal.New(123456, 0)
		_ = d.Rescale(Scale())
	}
}

func BenchmarkDecimalNewFromFloat64(b *testing.B) {
	for b.Loop() {
		d, _ := decimal.NewFromFloat64(123.456789)
		_ = d.Rescale(Scale())
	}
}

func BenchmarkDecimalArithmeticChain(b *testing.B) {
	a := decimal.MustParse("100.50").Rescale(Scale())
	c := decimal.MustParse("10").Rescale(Scale())
	d := decimal.MustParse("3").Rescale(Scale())
	for b.Loop() {
		sum, _ := a.Add(c)
		product, _ := sum.Mul(d)
		diff, _ := product.Sub(a)
		_, _ = diff.Quo(c)
	}
}

func BenchmarkDecimalMarshalJSON(b *testing.B) {
	d := decimal.MustParse("123.456789").Rescale(Scale())
	for b.Loop() {
		_, _ = json.Marshal(d)
	}
}

func BenchmarkDecimalUnmarshalJSON_Bare(b *testing.B) {
	data := []byte("123.456789")
	for b.Loop() {
		var d decimal.Decimal
		_ = d.UnmarshalJSON(data)
		d = d.Rescale(Scale())
	}
}

func BenchmarkDecimalUnmarshalJSON_Quoted(b *testing.B) {
	data := []byte(`"123.456789"`)
	for b.Loop() {
		var d decimal.Decimal
		_ = d.UnmarshalJSON(data)
		d = d.Rescale(Scale())
	}
}

func BenchmarkDecimalScan(b *testing.B) {
	for b.Loop() {
		var d decimal.Decimal
		_ = d.Scan("123.456789")
		d = d.Rescale(Scale())
	}
}

func BenchmarkDecimalValue(b *testing.B) {
	d := decimal.MustParse("123.456789").Rescale(Scale())
	for b.Loop() {
		_, _ = d.Value()
	}
}
