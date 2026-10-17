// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num_test

import (
	"encoding/json"
	"fmt"

	"go.trippwill.dev/num"
)

func ExampleFromString() {
	n := num.FromString("123.45")
	fmt.Println(n)
	// Output: 123.45
}

func ExampleNum_Add() {
	price := num.FromString("0.1")
	tax := num.FromString("0.2")
	total := price.Add(tax)
	fmt.Println(total)
	// Output: 0.3
}

func ExampleNum_errorChain() {
	exit := num.FromString("150.00")
	entry := num.FromString("100.00")
	qty := num.FromInt64(10)

	pnl := exit.Sub(entry).Mul(qty)
	if !pnl.Ok() {
		fmt.Println("error:", pnl.Err)
		return
	}
	fmt.Println(pnl)
	// Output: 500.00
}

func ExampleNum_MarshalJSON() {
	n := num.FromString("42.50")
	data, _ := json.Marshal(n)
	fmt.Println(string(data))
	// Output: 42.50
}

func ExampleNum_Quantize() {
	price := num.FromString("1.235")
	fmt.Println(price.Quantize(2))
	fmt.Printf("display=%.2f exact=%s\n", price, price)
	// Output:
	// 1.24
	// display=1.24 exact=1.235
}

func ExampleNullNum() {
	type Position struct {
		Price  num.Num     `json:"price"`
		Strike num.NullNum `json:"strike"`
	}

	data := []byte(`{"price": 42.50, "strike": null}`)
	var pos Position
	_ = json.Unmarshal(data, &pos)

	fmt.Printf("price=%s strike.valid=%v\n", pos.Price, pos.Strike.Valid)
	// Output: price=42.50 strike.valid=false
}
