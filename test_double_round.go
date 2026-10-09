package num

import (
	"fmt"
	"testing"

	"github.com/govalues/decimal"
)

// This demonstrates the double rounding problem when parsing a string
// with more precision than the destination scale.
func TestDoubleRounding(t *testing.T) {
	// Input has 19 decimal places, but we want to rescale to 6 (our process scale)
	// This tests the concern about double rounding

	// Using current approach: Parse then Rescale
	s := "1.2345678901234567890" // 19 decimal places
	d1, _ := decimal.Parse(s)
	fmt.Println("After Parse:", d1.String(), "Scale:", d1.Scale())
	// This might have already rounded if scale > 19

	rescaled := d1.Rescale(6)
	fmt.Println("After Rescale(6):", rescaled.String(), "Scale:", rescaled.Scale())

	// Using ParseExact: Parse to exact scale in one operation
	d2, err := decimal.ParseExact(s, 6)
	if err != nil {
		fmt.Println("ParseExact error:", err)
	} else {
		fmt.Println("After ParseExact(s, 6):", d2.String(), "Scale:", d2.Scale())
	}

	if d1.String() != d2.String() {
		t.Errorf("Double rounding detected!\n  Parse+Rescale: %s\n  ParseExact: %s", rescaled.String(), d2.String())
	}
}
