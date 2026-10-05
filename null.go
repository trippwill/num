// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import "database/sql/driver"

// NullNum represents a [Num] that may be null.
type NullNum struct {
	Num   Num
	Valid bool
}

// String implements [fmt.Stringer].
func (nn NullNum) String() string {
	if !nn.Valid {
		return "<null>"
	}
	return nn.Num.String()
}

// MarshalJSON returns JSON null when !Valid, otherwise delegates to [Num.MarshalJSON].
func (nn NullNum) MarshalJSON() ([]byte, error) {
	if !nn.Valid {
		return []byte("null"), nil
	}
	return nn.Num.MarshalJSON()
}

// UnmarshalJSON handles JSON null → Valid=false, otherwise delegates to [Num.UnmarshalJSON].
func (nn *NullNum) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nn.Valid = false
		nn.Num = Num{}
		return nil
	}
	nn.Valid = true
	return nn.Num.UnmarshalJSON(data)
}

// MarshalText returns an empty byte slice when !Valid, otherwise delegates to [Num.MarshalText].
func (nn NullNum) MarshalText() ([]byte, error) {
	if !nn.Valid {
		return []byte{}, nil
	}
	return nn.Num.MarshalText()
}

// UnmarshalText handles empty input → Valid=false, otherwise delegates to [Num.UnmarshalText].
func (nn *NullNum) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		nn.Valid = false
		nn.Num = Num{}
		return nil
	}
	var parsed Num
	if err := parsed.UnmarshalText(data); err != nil {
		nn.Valid = false
		nn.Num = Num{}
		return err
	}
	nn.Valid = true
	nn.Num = parsed
	return nil
}

// Value implements [database/sql/driver.Valuer]. Returns nil when !Valid.
func (nn NullNum) Value() (driver.Value, error) {
	if !nn.Valid {
		return nil, nil
	}
	return nn.Num.Value()
}

// Scan implements [database/sql.Scanner]. Handles nil → Valid=false.
func (nn *NullNum) Scan(src any) error {
	if src == nil {
		nn.Valid = false
		nn.Num = Num{}
		return nil
	}
	var scanned Num
	if err := scanned.Scan(src); err != nil {
		nn.Valid = false
		nn.Num = Num{}
		return err
	}
	nn.Valid = true
	nn.Num = scanned
	return nil
}
