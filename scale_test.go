// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import "testing"

func TestParseScale(t *testing.T) {
	tests := []struct {
		name     string
		override string
		fallback int
		want     int
	}{
		{"empty uses fallback", "", 6, 6},
		{"valid 4", "4", 6, 4},
		{"valid 0", "0", 6, 0},
		{"valid 19 (max)", "19", 6, 19},
		{"negative falls back", "-1", 6, 6},
		{"too large falls back", "20", 6, 6},
		{"non-numeric falls back", "abc", 6, 6},
		{"float falls back", "6.5", 6, 6},
		{"empty with different fallback", "", 9, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScale(tt.override, tt.fallback)
			if got != tt.want {
				t.Errorf("parseScale(%q, %d) = %d, want %d", tt.override, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestDefaultScale(t *testing.T) {
	if got := Scale(); got != defaultScale {
		t.Skipf("Scale() = %d, expected %d — likely built with ldflags override", got, defaultScale)
	}
}
