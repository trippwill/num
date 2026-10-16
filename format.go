// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/apd/v3"
)

const (
	formatMaxPrecision   = 4096
	formatMaxWidth       = 16384
	formatMaxOutputBytes = 16384
)

// Format implements [fmt.Formatter]. The f and F verbs round half-even to the
// requested number of fractional digits, defaulting to six, without changing n.
// They support width and the +, space, -, and 0 flags. The v and s verbs print
// the complete [Num.String] representation; q quotes it as a Go string. These
// text verbs support width and the - flag, and reject precision.
//
// Precision is limited to 4096, width to 16384, and each formatted value to
// 16384 bytes. Unsupported formatting and invalid numeric states produce fmt
// diagnostics. Text verbs retain the error or uninitialized String marker.
func (n Num) Format(state fmt.State, verb rune) {
	numeric := verb == 'f' || verb == 'F'
	if !numeric && verb != 'v' && verb != 's' && verb != 'q' {
		writeFormatDiagnostic(state, verb, "unsupported verb")
		return
	}

	width, hasWidth := state.Width()
	if hasWidth && (width < 0 || width > formatMaxWidth) {
		writeFormatDiagnostic(state, verb, "width exceeds 16384")
		return
	}
	precision, hasPrecision := state.Precision()
	if hasPrecision && !numeric {
		writeFormatDiagnostic(state, verb, "precision requires f or F")
		return
	}
	if hasPrecision && (precision < 0 || precision > formatMaxPrecision) {
		writeFormatDiagnostic(state, verb, "precision exceeds 4096")
		return
	}
	for _, flag := range []byte{'#', '+', ' ', '0'} {
		if state.Flag(int(flag)) && (!numeric || flag == '#') {
			writeFormatDiagnostic(state, verb, "unsupported flag "+strconv.QuoteRune(rune(flag)))
			return
		}
	}

	var result string
	if numeric {
		if n.stateError() != nil {
			writeFormatDiagnostic(state, verb, n.String())
			return
		}
		if !hasPrecision {
			precision = 6
		}
		// Rendering may require more coefficient digits than a stored Num allows.
		// Use the internal rounding helper, without public Quantize validation.
		var temporary apd.Decimal
		decimal, err := quantizeDecimal(n.decimal(&temporary), precision)
		if err != nil {
			writeFormatDiagnostic(state, verb, err.Error())
			return
		}
		result = decimal.Text('f')
		if decimal.IsZero() {
			result = strings.TrimPrefix(result, "-")
		}
		if !strings.HasPrefix(result, "-") {
			if state.Flag('+') {
				result = "+" + result
			} else if state.Flag(' ') {
				result = " " + result
			}
		}
	} else {
		result = n.String()
		if verb == 'q' {
			result = strconv.Quote(result)
		}
	}

	padding := max(0, width-utf8.RuneCountInString(result))
	if len(result)+padding > formatMaxOutputBytes {
		writeFormatDiagnostic(state, verb, "output exceeds 16384 bytes")
		return
	}
	if padding > 0 {
		switch {
		case state.Flag('-'):
			result += strings.Repeat(" ", padding)
		case numeric && state.Flag('0'):
			prefix := 0
			if result[0] == '-' || result[0] == '+' || result[0] == ' ' {
				prefix = 1
			}
			result = result[:prefix] + strings.Repeat("0", padding) + result[prefix:]
		default:
			result = strings.Repeat(" ", padding) + result
		}
	}
	_, _ = io.WriteString(state, result)
}

func writeFormatDiagnostic(state fmt.State, verb rune, reason string) {
	prefix := "%!" + string(verb) + "(num.Num="
	const suffix = ")"
	limit := formatMaxOutputBytes - len(prefix) - len(suffix)
	if len(reason) > limit {
		reason = reason[:limit-3]
		for len(reason) > 0 && !utf8.ValidString(reason) {
			reason = reason[:len(reason)-1]
		}
		reason += "..."
	}
	_, _ = io.WriteString(state, prefix+reason+suffix)
}
