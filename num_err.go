// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

// Error is a sentinel error type for the num package.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidNumericString Error = "num: invalid numeric string"
	ErrIntegerOverflow      Error = "num: integer overflow"
	ErrUnsupportedType      Error = "num: unsupported type"
	ErrMarshalErrorState    Error = "num: cannot marshal error-state value"
)
