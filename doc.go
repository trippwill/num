// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

// Package num provides [Num] and [NullNum] types for exact decimal
// arithmetic on financial data. Num contains [github.com/govalues/decimal.Decimal]
// with error-carrying chainable arithmetic, a fixed per-binary scale, and
// serialization interfaces covering JSON, XML, SQL, and text.
//
// # Error-Carrying Arithmetic
//
// Arithmetic methods propagate errors through chains, allowing callers to
// check once at the end rather than after every operation:
//
//	pnl := num.From(exitPrice).Sub(num.From(entryPrice)).Mul(num.From(quantity))
//	if !pnl.Ok() {
//	    return fmt.Errorf("P&L calculation: %w", pnl.Err)
//	}
//
// # Scale
//
// A process-global scale (default 6) is enforced at construction time.
// Override at link time with:
//
//	go build -ldflags "-X github.com/trippwill/num.scaleOverride=4"
//
// Valid range: 0–19 (matching [github.com/govalues/decimal] constraints).
//
// # JSON Flexibility
//
// [Num.UnmarshalJSON] accepts both bare numeric (123.45) and quoted string
// ("123.45") inputs — handling the mixed formats common in IBKR REST API
// responses. Parse failures set [Num.Err] rather than returning an error,
// allowing partial struct unmarshal.
//
// # NullNum
//
// [NullNum] wraps Num with a Valid flag for nullable database columns and
// optional API response fields. JSON null, SQL NULL, and empty text all
// map to Valid=false.
package num
