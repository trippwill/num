// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num_test

import (
	"encoding/json"
	"fmt"

	"github.com/trippwill/num"
)

func ExampleFromString() {
	n := num.FromString("123.45")
	fmt.Println(n)
	// Output: 123.450000
}

func ExampleNum_Add() {
	price := num.FromString("0.1")
	tax := num.FromString("0.2")
	total := price.Add(tax)
	fmt.Println(total)
	// Output: 0.300000
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
	// Output: 500.000000
}

func ExampleNum_MarshalJSON() {
	n := num.FromString("42.50")
	data, _ := json.Marshal(n)
	fmt.Println(string(data))
	// Output: "42.500000"
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
	// Output: price=42.500000 strike.valid=false
}
