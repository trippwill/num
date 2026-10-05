// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import "testing"

func sqlRoundtripScaleOf(n Num) int {
	return n.dec.Scale()
}

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
	original := FromString("123.456789")
	restored := roundtripNum(t, original)
	if !restored.Ok() {
		t.Fatalf("scan error: %v", restored.Err)
	}
	if !original.Equal(restored) {
		t.Errorf("roundtrip: got %s, want %s", restored.dec.String(), original.dec.String())
	}
	if got := sqlRoundtripScaleOf(restored); got != Scale() {
		t.Errorf("scale = %d, want %d", got, Scale())
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
		t.Errorf("expected Valid=false after NULL roundtrip, got %s", restored.Num.dec.String())
	}
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
