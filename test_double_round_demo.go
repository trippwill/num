package num

import "github.com/govalues/decimal"

// Demonstration of double rounding concern
func demonstrateDoubleRounding() {
	// Input has 19 decimal places
	s := "1.2345678901234567890" // 19 decimal places

	// Current approach: Parse then Rescale
	d1, _ := decimal.Parse(s)
	println("After Parse: scale =", d1.Scale())
	rescaled := d1.Rescale(6)
	println("After Rescale(6):", rescaled.String())

	// Better approach: ParseExact to target scale
	d2, err := decimal.ParseExact(s, 6)
	if err != nil {
		println("ParseExact error:", err)
	} else {
		println("After ParseExact(s, 6):", d2.String())
	}
}
