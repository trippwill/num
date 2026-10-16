// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

var (
	benchmarkNum   Num
	benchmarkBytes []byte
	benchmarkValue driver.Value
	benchmarkErr   error
)

func BenchmarkFromString(b *testing.B) {
	for b.Loop() {
		benchmarkNum = FromString("123.456789")
	}
}

func BenchmarkFromInt64(b *testing.B) {
	for b.Loop() {
		benchmarkNum = FromInt64(123456)
	}
}

func BenchmarkFromFloat64(b *testing.B) {
	for b.Loop() {
		benchmarkNum = FromFloat64(123.456789)
	}
}

func BenchmarkArithmeticChain(b *testing.B) {
	a := FromString("100.50")
	c := FromString("10")
	d := FromString("3")
	for b.Loop() {
		benchmarkNum = a.Add(c).Mul(d).Sub(a).Div(c)
	}
	if !benchmarkNum.Ok() || !benchmarkNum.Equal(FromString("23.1")) {
		b.Fatalf("unexpected arithmetic result: %v", benchmarkNum)
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		benchmarkBytes, benchmarkErr = json.Marshal(n)
	}
	if benchmarkErr != nil {
		b.Fatal(benchmarkErr)
	}
}

func BenchmarkUnmarshalJSON_Bare(b *testing.B) {
	data := []byte("123.456789")
	for b.Loop() {
		var n Num
		benchmarkErr = n.UnmarshalJSON(data)
		benchmarkNum = n
	}
	if benchmarkErr != nil || !benchmarkNum.Ok() {
		b.Fatalf("unmarshal: %v, value: %v", benchmarkErr, benchmarkNum)
	}
}

func BenchmarkUnmarshalJSON_Quoted(b *testing.B) {
	data := []byte(`"123.456789"`)
	for b.Loop() {
		var n Num
		benchmarkErr = n.UnmarshalJSON(data)
		benchmarkNum = n
	}
	if benchmarkErr != nil || !benchmarkNum.Ok() {
		b.Fatalf("unmarshal: %v, value: %v", benchmarkErr, benchmarkNum)
	}
}

func BenchmarkScan(b *testing.B) {
	for b.Loop() {
		var n Num
		benchmarkErr = n.Scan("123.456789")
		benchmarkNum = n
	}
	if benchmarkErr != nil || !benchmarkNum.Ok() {
		b.Fatalf("scan: %v, value: %v", benchmarkErr, benchmarkNum)
	}
}

func BenchmarkValue(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		benchmarkValue, benchmarkErr = n.Value()
	}
	if benchmarkErr != nil {
		b.Fatal(benchmarkErr)
	}
}
