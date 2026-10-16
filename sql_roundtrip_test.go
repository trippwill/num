// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import "testing"

func roundtripNum(t *testing.T, original Num) Num {
	t.Helper()

	value, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored Num
	if err := restored.Scan(value); err != nil {
		t.Fatal(err)
	}
	return restored
}

func roundtripNullNum(t *testing.T, original NullNum) NullNum {
	t.Helper()

	value, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored NullNum
	if err := restored.Scan(value); err != nil {
		t.Fatal(err)
	}
	return restored
}

func TestNumSQLRoundtrip(t *testing.T) {
	for _, input := range []string{
		"0", "0.0000", "123.456789", "123.4500000000",
		"123456789012345678901234567890123456789.1234567890123456789",
		"0.00000000000000000000000000000000000000000000000001",
	} {
		t.Run(input, func(t *testing.T) {
			original := FromString(input)
			restored := roundtripNum(t, original)
			requireNum(t, restored, input)
			if !original.Equal(restored) {
				t.Fatalf("SQL roundtrip changed value %s", original)
			}
		})
	}
}

func TestNullNumSQLRoundtrip_Valid(t *testing.T) {
	original := NullNum{Num: FromString("99.99"), Valid: true}
	restored := roundtripNullNum(t, original)
	if !restored.Valid {
		t.Fatal("expected Valid=true after roundtrip")
	}
	if !restored.Num.Ok() {
		t.Fatalf("scan error: %v", restored.Num.Err)
	}
	if !original.Num.Equal(restored.Num) {
		t.Errorf("roundtrip: got %s, want %s",
			restored.Num.dec.String(), original.Num.dec.String())
	}
}

func TestNullNumSQLRoundtrip_Null(t *testing.T) {
	null := NullNum{Valid: false}
	restored := roundtripNullNum(t, null)
	if restored.Valid {
		t.Errorf("expected Valid=false after NULL roundtrip, got %s", restored.Num.String())
	}
}

func TestNullNumSQLRoundtripZeroIsNotNull(t *testing.T) {
	original := NullNum{Num: Zero().Quantize(4), Valid: true}
	restored := roundtripNullNum(t, original)
	if !restored.Valid || !restored.Num.IsZero() {
		t.Fatalf("numeric zero became null: %v", restored)
	}
	requireNum(t, restored.Num, "0.0000")
}

func TestNullNumSQLRoundtrip_MixedRow(t *testing.T) {
	price := FromString("42.50")
	strike := NullNum{Valid: false}
	rPrice := roundtripNum(t, price)
	rStrike := roundtripNullNum(t, strike)

	if !rPrice.Ok() || !rPrice.Equal(price) {
		t.Errorf("price roundtrip: got %s, want %s", rPrice.dec.String(), price.dec.String())
	}
	if rStrike.Valid {
		t.Error("strike should be NULL")
	}
}
