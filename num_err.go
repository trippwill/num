// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

// Error is a sentinel error type for the num package.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidNumericString Error = "num: invalid numeric string"
	ErrUnsupportedType      Error = "num: unsupported type"
	ErrMarshalErrorState    Error = "num: cannot marshal error-state value"
	ErrUninitialized        Error = "num: uninitialized value"
	ErrInvalidScale         Error = "num: invalid scale"
	ErrLimitExceeded        Error = "num: numeric limit exceeded"
	ErrDivisionByZero       Error = "num: division by zero"
)
