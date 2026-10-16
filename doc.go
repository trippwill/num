// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

// Package num provides error-carrying decimal arithmetic with fixed policies.
// [Num] keeps small coefficients inline and uses private apd storage for larger
// values and rounded division. It preserves input precision and supports JSON,
// XML, SQL, text, and Go's fmt package. Values are
// immutable: assignment, arithmetic, and formatting never mutate another copy.
// Num is deliberately not comparable; use [Num.Equal] or [Num.Cmp].
//
// # Calculations and rounding
//
// Constructors preserve decimal values and fractional trailing zeros. Addition
// and subtraction are exact and retain the greater operand scale;
// multiplication is exact and retains the sum of the scales. Capacity errors
// are reported instead of rounding an exact result to fit.
//
// Division is a rounding boundary. [Num.Div] rounds half-even (banker's rounding)
// to max(34, meaningfulDigits(left), meaningfulDigits(right)) significant digits,
// then removes unnecessary trailing zeros. Meaningful digits are the coefficient
// digits after removing trailing zeros from a temporary copy; zero counts as one
// digit. Equivalent values such as 1.23 and 1.2300 select the same precision.
// The 34-digit floor follows the decimal128 precision convention; it is a package
// policy rather than a claim that every later calculation will be correctly
// rounded. Repeating quotients succeed. Terminating quotients exceeding the
// selected precision also round.
//
// [Num.Quantize] is an explicit rounding boundary: Quantize(2) rounds half-even
// or pads zeros to exactly two fractional places. Half-even is the only rounding
// rule: exact ties choose the even final digit, for either sign. There are no
// public contexts, rounding-mode parameters, or process-global scale settings.
// Negative zero becomes positive zero, retaining any intentional fractional
// scale. Comparisons ignore scale; callers must check [Num.Ok] before comparing.
//
// Arithmetic carries errors through chains so callers can check the final value:
//
//	pnl := exitPrice.Sub(entryPrice).Mul(quantity)
//	settlement := pnl.Quantize(2)
//	if !settlement.Ok() {
//		return fmt.Errorf("settlement: %w", settlement.Err)
//	}
//
// # Formatting
//
// Formatting selects presentation without changing the value. [Num.String],
// %v, and %s produce the complete stored representation in plain decimal
// notation. %q quotes that representation. %f and %F select fixed-point display
// with half-even rounding; their default precision is six fractional places.
// Width and precision may be supplied dynamically:
//
//	n := num.FromString("123.456789")
//	fmt.Sprintf("%.2f", n)         // "123.46"
//	fmt.Sprintf("%10.2f", n)       // "    123.46"
//	fmt.Sprintf("%.*f", 4, n)      // "123.4568"
//	n.String()                     // "123.456789", unchanged
//
// Numeric formatting supports +, space, -, and 0 flags. Plus takes precedence
// over space; left alignment takes precedence over zero-padding; zeros follow
// the sign. Textual forms support width and alignment, but reject precision
// rather than truncate numeric text. Unsupported verbs (including %e and %g),
// unsupported flags, and excessive rendering requests produce bounded fmt-style
// diagnostics. Numeric formatting of invalid values includes an error or
// uninitialized marker; textual formatting preserves that marker. Formatting
// never converts through float64.
//
// # Capacity and input
//
// Stored values support at most 4,096 coefficient digits (including retained
// trailing zeros), 4,096 integer digits, and a fractional scale from 0 through
// 4,096. Positive exponents are normalized to scale zero before validation.
// Construction and exact arithmetic never silently round to satisfy capacity.
// Division results must also fit after unnecessary trailing zeros are removed.
//
// Numeric input is limited to 16,384 decoded bytes. Accepted forms are finite
// decimal literals with an optional sign, fractional part, and decimal exponent;
// forms such as .5 and 1. are accepted. Embedded or surrounding numeric-text
// whitespace, underscores, nondecimal syntax, NaN, and infinities are rejected.
// JSON permits its own surrounding whitespace and standard string escapes.
// Float64 construction uses the shortest round-trip decimal representation of
// the supplied float, not its exact binary expansion.
//
// Rendering separately permits up to 4,096 fractional places, field width
// 16,384, and 16,384 output bytes. Formatting padding may exceed stored-value
// coefficient capacity because it is presentation only.
//
// # Errors and initialization
//
// Num{} is uninitialized, not numeric zero: its Err field is nil but Ok is false.
// Use [Zero] for an initialized scale-zero zero. Arithmetic and serialization
// reject uninitialized operands rather than treating them as zero. Errors from
// invalid input, capacity limits, invalid quantization scale, and division by
// zero propagate through arithmetic. Ordinary division or quantization rounding
// succeeds. Existing carried errors take precedence over later calculation.
// Use errors.Is to identify [ErrInvalidNumericString], [ErrUninitialized],
// [ErrInvalidScale], [ErrLimitExceeded], or [ErrDivisionByZero]. Unsupported SQL
// scan sources report [ErrUnsupportedType]. Serialization failures wrap
// [ErrMarshalErrorState] together with the underlying numeric error.
//
// # Serialization and nullable values
//
// JSON output is a bare exact decimal number, including stored trailing zeros.
// [Num.UnmarshalJSON] accepts quoted and bare input; parse failures set Num.Err
// and return nil so unrelated struct fields can still decode. JSON null is
// invalid for Num. Text/XML and SQL encode exact decimal text without rounding;
// SQL [Num.Value] returns a string. Text and SQL decoding return their errors.
// Failed decoding clears stale numeric state.
//
// Direct JSON decoding into Num preserves precision. Generic Go decoding into
// any uses float64 by default, and JavaScript numbers use binary floating point;
// these consumers can lose decimal precision. Go consumers handling generic
// JSON can call encoding/json.Decoder.UseNumber and pass json.Number.String()
// to [FromString]. Output does not switch between strings and numbers by size.
//
// [NullNum] represents presence separately from numeric validity. JSON null,
// SQL NULL, and empty text set Valid=false; absent JSON output is null. A present
// malformed JSON number has Valid=true and a carried Num error. Failed text or
// SQL decoding returns an error and leaves Valid=false. A missing nullable struct
// field remains absent. SQL scanning accepts string, []byte, int64, uint64,
// float64, and float32; float32 is first converted to float64.
package num
