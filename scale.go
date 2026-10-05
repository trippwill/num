// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import "strconv"

const defaultScale = 6

var (
	processScale  int
	scaleOverride string // set via -ldflags "-X github.com/trippwill/num.scaleOverride=N"
)

func init() {
	processScale = parseScale(scaleOverride, defaultScale)
}

// parseScale converts a string override to a valid scale, falling back to
// the provided default on empty, non-numeric, or out-of-range input.
func parseScale(override string, fallback int) int {
	if override == "" {
		return fallback
	}
	s, err := strconv.Atoi(override)
	if err != nil || s < 0 || s > 19 {
		return fallback
	}
	return s
}

// Scale returns the process-global decimal scale.
func Scale() int { return processScale }
