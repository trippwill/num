// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

var (
	benchmarkText string
	benchmarkAPD  *apd.Decimal
)

// These cases keep the short exact path, rounded division, and promotion to
// arbitrary precision visible separately. The raw apd benchmark below retains
// its own normalization so it measures the unmodified backend policy.
func BenchmarkDivision(b *testing.B) {
	for _, tc := range []struct{ name, a, divisor, want string }{
		{"compact_exact", "231.00", "10", "23.1"},
		{"compact_fraction", "1", "8", "0.125"},
		{"repeating", "1", "3", "0." + strings.Repeat("3", 34)},
		{"terminating_rounded", "1", "1125899906842624", "0.0000000000000008881784197001252323389053344726562"},
		{"large_exact", strings.Repeat("9", 50), "1", strings.Repeat("9", 50)},
		{"padded_operand", "1.23" + strings.Repeat("0", 100), "1", "1.23"},
		{"tiny_scale", "1e-4096", "1", "0." + strings.Repeat("0", 4095) + "1"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			n, divisor := FromString(tc.a), FromString(tc.divisor)
			for b.Loop() {
				benchmarkNum = n.Div(divisor)
			}
			if !benchmarkNum.Ok() || benchmarkNum.String() != tc.want {
				b.Fatalf("unexpected quotient: %v", benchmarkNum)
			}
		})
	}
}

func BenchmarkQuantize(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		benchmarkNum = n.Quantize(2)
	}
	if !benchmarkNum.Ok() {
		b.Fatal(benchmarkNum.Err)
	}
}

func BenchmarkFormatFixed(b *testing.B) {
	n := FromString("123.456789")
	for b.Loop() {
		benchmarkText = fmt.Sprintf("%10.2f", n)
	}
}

// BenchmarkAPDArithmeticChain uses the same arithmetic policy and result as
// BenchmarkArithmeticChain for these inputs. It isolates raw backend work;
// it does not use Num's compact coefficient and exact-quotient fast paths.
func BenchmarkAPDArithmeticChain(b *testing.B) {
	a, _, err := apd.NewFromString("100.50")
	if err != nil {
		b.Fatal(err)
	}
	c := apd.New(10, 0)
	d := apd.New(3, 0)
	exact := apd.Context{
		MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent,
		Traps:       apd.DefaultTraps,
		Rounding:    apd.RoundHalfEven,
	}
	division := exact
	division.Precision = 34 // All three operands have fewer meaningful digits.
	for b.Loop() {
		var sum, product, difference apd.Decimal
		result := new(apd.Decimal)
		_, _ = exact.Add(&sum, a, c)
		_, _ = exact.Mul(&product, &sum, d)
		_, _ = exact.Sub(&difference, &product, a)
		_, benchmarkErr = division.Quo(result, &difference, c)
		result.Reduce(result)
		benchmarkAPD = result
	}
	if benchmarkErr != nil || benchmarkAPD.Text('f') != "23.1" {
		b.Fatalf("unexpected arithmetic result: %v, %v", benchmarkAPD, benchmarkErr)
	}
}
