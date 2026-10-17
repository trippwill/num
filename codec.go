// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// MarshalJSON encodes n as an exact bare JSON number, retaining stored scale.
// No presentation formatting or binary floating-point conversion is involved.
func (n Num) MarshalJSON() ([]byte, error) {
	if err := n.stateError(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMarshalErrorState, err)
	}
	if n.dec == &compactMarker {
		return n.appendCompact(make([]byte, 0, 32)), nil
	}
	return n.dec.Append(nil, 'f'), nil
}

// UnmarshalJSON accepts numeric tokens and quoted decimal strings. Invalid
// values clear the previous number and carry Err, allowing partial struct decode.
func (n *Num) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, " \t\r\n")
	// One decoded byte can occupy at most six bytes as a JSON Unicode escape.
	if len(data) > maxInputBytes*6+2 {
		*n = Num{Err: ErrLimitExceeded}
		return nil
	}
	// Plain quoted decimals need no JSON unescaping. The decimal grammar also
	// rejects embedded quotes, control bytes, and every nonnumeric character.
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' && !bytes.ContainsRune(data[1:len(data)-1], '\\') {
		if parsed, ok := parseCompact(string(data[1 : len(data)-1])); ok {
			*n = parsed
			return nil
		}
		*n = FromString(string(data[1 : len(data)-1]))
		return nil
	}
	// A short finite token can be parsed directly once its JSON-only grammar
	// restrictions have been checked. Escapes and exponent forms fall through.
	if len(data) > 0 && data[0] != '"' && compactJSONToken(data) {
		if parsed, ok := parseCompact(string(data)); ok {
			*n = parsed
			return nil
		}
	}
	var text string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			*n = Num{Err: fmt.Errorf("%w: %w", ErrInvalidNumericString, err)}
			return nil
		}
	} else {
		if !json.Valid(data) {
			*n = Num{Err: ErrInvalidNumericString}
			return nil
		}
		text = string(data)
	}
	*n = FromString(text)
	return nil
}

func compactJSONToken(data []byte) bool {
	i := 0
	if data[0] == '-' {
		i++
	}
	if i == len(data) || data[i] < '0' || data[i] > '9' || data[len(data)-1] == '.' {
		return false
	}
	return data[i] != '0' || i+1 == len(data) || data[i+1] == '.'
}

// MarshalText returns exact decimal text for text and XML encoders.
func (n Num) MarshalText() ([]byte, error) { return n.MarshalJSON() }

// UnmarshalText parses exact decimal text, clearing stale state on failure.
func (n *Num) UnmarshalText(data []byte) error {
	if len(data) > maxInputBytes {
		*n = Num{Err: ErrLimitExceeded}
		return n.Err
	}
	*n = FromString(string(data))
	return n.stateError()
}

// Scan implements database/sql.Scanner. Accepted types are string, []byte,
// int64, uint64, float64, and float32. Floats use shortest decimal conversion;
// float32 first converts to float64, preserving the earlier driver contract.
func (n *Num) Scan(src any) error {
	var parsed Num
	switch value := src.(type) {
	case string:
		parsed = FromString(value)
	case []byte:
		return n.UnmarshalText(value)
	case int64:
		parsed = FromInt64(value)
	case uint64:
		parsed = compact(value, 0, false)
	case float64:
		parsed = FromFloat64(value)
	case float32:
		parsed = FromFloat64(float64(value))
	default:
		parsed.Err = fmt.Errorf("%w: %T", ErrUnsupportedType, src)
	}
	*n = parsed
	return n.stateError()
}

// Value implements database/sql/driver.Valuer using exact string-based output.
func (n Num) Value() (driver.Value, error) {
	if n.dec == &compactMarker && n.Err == nil {
		return compactText(n.coef, n.scale), nil
	}
	if err := n.stateError(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMarshalErrorState, err)
	}
	return n.dec.Text('f'), nil
}
