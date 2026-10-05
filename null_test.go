// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"testing"
)

func TestNullNumJSON_Null(t *testing.T) {
	// Marshal
	nn := NullNum{Valid: false}
	data, err := json.Marshal(nn)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Errorf("Marshal(!Valid) = %s, want null", data)
	}

	// Unmarshal
	var restored NullNum
	if err := json.Unmarshal([]byte("null"), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Valid {
		t.Error("Unmarshal(null) should set Valid=false")
	}
}

func TestNullNumJSON_Valid(t *testing.T) {
	nn := NullNum{Num: FromString("123.45"), Valid: true}
	data, err := json.Marshal(nn)
	if err != nil {
		t.Fatal(err)
	}

	var restored NullNum
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Valid {
		t.Error("Unmarshal should set Valid=true")
	}
	if !restored.Num.Ok() {
		t.Fatalf("unmarshal error: %v", restored.Num.Err)
	}
	if !nn.Num.Equal(restored.Num) {
		t.Errorf("roundtrip: got %s, want %s",
			restored.Num.dec.String(), nn.Num.dec.String())
	}
}

func TestNullNumJSON_Roundtrip(t *testing.T) {
	type record struct {
		Price  NullNum `json:"price"`
		Strike NullNum `json:"strike"`
	}

	orig := record{
		Price:  NullNum{Num: FromString("42.50"), Valid: true},
		Strike: NullNum{Valid: false},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}

	var restored record
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Price.Valid || !restored.Price.Num.Equal(orig.Price.Num) {
		t.Errorf("Price roundtrip failed: %+v", restored.Price)
	}
	if restored.Strike.Valid {
		t.Errorf("Strike should be null: %+v", restored.Strike)
	}
}

func TestNullNumText_Null(t *testing.T) {
	nn := NullNum{Valid: false}
	data, err := nn.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("MarshalText(!Valid) = %q, want empty", data)
	}

	var restored NullNum
	if err := restored.UnmarshalText([]byte{}); err != nil {
		t.Fatal(err)
	}
	if restored.Valid {
		t.Error("UnmarshalText(empty) should set Valid=false")
	}
}

func TestNullNumText_Valid(t *testing.T) {
	nn := NullNum{Num: FromString("99.99"), Valid: true}
	data, err := nn.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var restored NullNum
	if err := restored.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if !restored.Valid {
		t.Error("UnmarshalText should set Valid=true")
	}
	if !nn.Num.Equal(restored.Num) {
		t.Errorf("text roundtrip: got %s, want %s",
			restored.Num.dec.String(), nn.Num.dec.String())
	}
}

func TestNullNumUnmarshalText_FailureKeepsInvalid(t *testing.T) {
	nn := NullNum{Num: FromString("99.99"), Valid: true}
	if err := nn.UnmarshalText([]byte("not-a-number")); err == nil {
		t.Fatal("expected error from invalid text")
	}
	if nn.Valid {
		t.Fatal("NullNum should be invalid after failed UnmarshalText")
	}
	if nn.Num.Ok() {
		t.Fatal("NullNum.Num should not be ok after failed UnmarshalText")
	}
}

func TestNullNumScan_Nil(t *testing.T) {
	var nn NullNum
	if err := nn.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if nn.Valid {
		t.Error("Scan(nil) should set Valid=false")
	}
}

func TestNullNumScan_Valid(t *testing.T) {
	var nn NullNum
	if err := nn.Scan("42.5"); err != nil {
		t.Fatal(err)
	}
	if !nn.Valid {
		t.Error("Scan should set Valid=true")
	}
	if !nn.Num.Ok() {
		t.Fatalf("scan error: %v", nn.Num.Err)
	}
	expected := FromString("42.5")
	if !nn.Num.Equal(expected) {
		t.Errorf("Scan(42.5) = %s, want %s",
			nn.Num.dec.String(), expected.dec.String())
	}
}

func TestNullNumScan_FailureKeepsInvalid(t *testing.T) {
	nn := NullNum{Num: FromString("99.99"), Valid: true}
	if err := nn.Scan(struct{}{}); err == nil {
		t.Fatal("expected error from unsupported scan type")
	}
	if nn.Valid {
		t.Fatal("NullNum should be invalid after failed Scan")
	}
	if nn.Num.Ok() {
		t.Fatal("NullNum.Num should not be ok after failed Scan")
	}
}

func TestNullNumValue_Null(t *testing.T) {
	nn := NullNum{Valid: false}
	v, err := nn.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.Errorf("Value(!Valid) = %v, want nil", v)
	}
}

func TestNullNumValue_Valid(t *testing.T) {
	nn := NullNum{Num: FromString("123.45"), Valid: true}
	v, err := nn.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value type = %T, want string", v)
	}
	if s != "123.450000" {
		t.Errorf("Value() = %q, want %q", s, "123.450000")
	}
}

func TestNullNum_String(t *testing.T) {
	valid := NullNum{Num: FromString("42.50"), Valid: true}
	if got := valid.String(); got != "42.500000" {
		t.Errorf("valid String() = %q, want %q", got, "42.500000")
	}

	null := NullNum{Valid: false}
	if got := null.String(); got != "<null>" {
		t.Errorf("null String() = %q, want %q", got, "<null>")
	}
}
