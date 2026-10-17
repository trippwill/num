// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"testing"
)

func FuzzNumParseRoundtrip(f *testing.F) {
	for _, input := range []string{"", "1.2300", "-.5", "+1.", "1e-20", "NaN", "1_000", "1.2345665000000000001", "1e-4096", "1e4096"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		n := FromString(input)
		if !n.Ok() {
			return
		}
		data, err := json.Marshal(n)
		if err != nil || !json.Valid(data) {
			t.Fatalf("valid number emitted invalid JSON: %s, %v", data, err)
		}
		var restored Num
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		requireNum(t, restored, n.String())
		text, err := n.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		if err := restored.UnmarshalText(text); err != nil {
			t.Fatal(err)
		}
		requireNum(t, restored, n.String())
	})
}

func FuzzNumIntegerArithmetic(f *testing.F) {
	for _, pair := range [][2]int64{{0, 0}, {1, -1}, {math.MinInt64, math.MaxInt64}, {math.MaxInt64, math.MaxInt64}, {-9999999999999, 9999999999999}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, a, b int64) {
		x, y := FromInt64(a), FromInt64(b)
		bx, by := big.NewInt(a), big.NewInt(b)
		requireNum(t, x.Add(y), new(big.Int).Add(bx, by).String())
		requireNum(t, x.Sub(y), new(big.Int).Sub(bx, by).String())
		requireNum(t, x.Mul(y), new(big.Int).Mul(bx, by).String())
		requireNum(t, x, bx.String())
		requireNum(t, y, by.String())
	})
}

func FuzzNumQuantizeHalfEven(f *testing.F) {
	for _, seed := range []struct {
		coefficient   int64
		input, output uint8
	}{{1225, 3, 2}, {1235, 3, 2}, {-1225, 3, 2}, {-1235, 3, 2}, {-5, 3, 2}, {math.MinInt64, 18, 0}, {1, 0, 18}} {
		f.Add(seed.coefficient, seed.input, seed.output)
	}
	f.Fuzz(func(t *testing.T, coefficient int64, input, output uint8) {
		inScale, outScale := int(input%19), int(output%19)
		n := FromString(scaledIntegerText(big.NewInt(coefficient), inScale))
		q := new(big.Int).Abs(big.NewInt(coefficient))
		if inScale > outScale {
			divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(inScale-outScale)), nil)
			var remainder big.Int
			q.QuoRem(q, divisor, &remainder)
			comparison := new(big.Int).Lsh(&remainder, 1).Cmp(divisor)
			if comparison > 0 || comparison == 0 && q.Bit(0) == 1 {
				q.Add(q, big.NewInt(1))
			}
		} else {
			factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(outScale-inScale)), nil)
			q.Mul(q, factor)
		}
		if coefficient < 0 {
			q.Neg(q)
		}
		got := n.Quantize(outScale)
		requireNum(t, got, scaledIntegerText(q, outScale))
		requireNum(t, got.Quantize(outScale), got.String())
	})
}

func scaledIntegerText(coefficient *big.Int, scale int) string {
	digits := new(big.Int).Abs(coefficient).String()
	if len(digits) <= scale {
		digits = strings.Repeat("0", scale-len(digits)+1) + digits
	}
	if scale > 0 {
		digits = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if coefficient.Sign() < 0 {
		digits = "-" + digits
	}
	return digits
}
