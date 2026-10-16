// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

var _ fmt.Formatter = Num{}

func TestFormatFixedPoint(t *testing.T) {
	tests := []struct {
		name, input, format, want string
	}{
		{"default precision", "1.2", "%f", "1.200000"},
		{"uppercase verb", "1.235", "%.2F", "1.24"},
		{"half even down", "1.245", "%.2f", "1.24"},
		{"half even up", "1.235", "%.2f", "1.24"},
		{"negative half even", "-1.245", "%.2f", "-1.24"},
		{"integer half even down", "2.5", "%.0f", "2"},
		{"integer half even up", "3.5", "%.0f", "4"},
		{"carry", "99.995", "%.2f", "100.00"},
		{"padding scale", "1.2", "%.4f", "1.2000"},
		{"right width", "12.345", "%10.2f", "     12.34"},
		{"left width", "12.345", "%-10.2f", "12.34     "},
		{"plus zero padding", "12.345", "%+010.2f", "+000012.34"},
		{"space zero padding", "12.345", "% 010.2f", " 000012.34"},
		{"negative zero padding", "-12.345", "%010.2f", "-000012.34"},
		{"left overrides zero", "12.345", "%-010.2f", "12.34     "},
		{"plus overrides space", "12.345", "%+ .2f", "+12.34"},
		{"zero sign normalized", "-0.004", "%.2f", "0.00"},
		{"zero plus sign", "-0.004", "%+.2f", "+0.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := FromString(tt.input)
			if !n.Ok() {
				t.Fatal(n.Err)
			}
			before := n.String()
			if got := fmt.Sprintf(tt.format, n); got != tt.want {
				t.Fatalf("format %q: got %q, want %q", tt.format, got, tt.want)
			}
			if got := n.String(); got != before || !n.Ok() {
				t.Fatalf("formatting changed %q to %q (error %v)", before, got, n.Err)
			}
		})
	}
}

func TestFormatDynamicWidthAndPrecision(t *testing.T) {
	n := FromString("12.345")
	if got := fmt.Sprintf("%*.*f", 10, 2, n); got != "     12.34" {
		t.Fatalf("dynamic formatting = %q", got)
	}
	if got := fmt.Sprintf("%*.*f", -10, 2, n); got != "12.34     " {
		t.Fatalf("negative dynamic width = %q", got)
	}
}

func TestFormatText(t *testing.T) {
	n := FromString("12.3400")
	for _, tt := range []struct{ format, want string }{
		{"%v", "12.3400"},
		{"%s", "12.3400"},
		{"%q", `"12.3400"`},
		{"%12v", "     12.3400"},
		{"%-12s", "12.3400     "},
		{"%12q", `   "12.3400"`},
	} {
		if got := fmt.Sprintf(tt.format, n); got != tt.want {
			t.Errorf("format %q: got %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestFormatInvalidStates(t *testing.T) {
	for _, n := range []Num{{}, FromString("invalid"), FromInt64(1).Div(Zero())} {
		if n.Ok() {
			t.Fatal("expected invalid test value")
		}
		marker := n.String()
		for _, format := range []string{"%v", "%s"} {
			if got := fmt.Sprintf(format, n); got != marker {
				t.Errorf("format %q: got %q, want marker %q", format, got, marker)
			}
		}
		if got := fmt.Sprintf("%q", n); got != strconv.Quote(marker) {
			t.Errorf("quoted invalid value = %q", got)
		}
		for _, format := range []string{"%f", "%.2F"} {
			got := fmt.Sprintf(format, n)
			if !strings.HasPrefix(got, "%!") || !strings.Contains(got, marker) {
				t.Errorf("numeric invalid value = %q, want diagnostic with %q", got, marker)
			}
		}
	}
}

func TestFormatRejectsUnsupportedRequests(t *testing.T) {
	n := FromString("12.345")
	for _, format := range []string{
		"%d", "%e", "%g", "%#f", "%+v", "%0s", "% q", "%#q",
		"%.2v", "%.2s", "%.2q",
	} {
		if got := fmt.Sprintf(format, n); !strings.HasPrefix(got, "%!") {
			t.Errorf("format %q = %q, want diagnostic", format, got)
		}
	}
}

func TestFormatPrecisionAndWidthBounds(t *testing.T) {
	n := FromString("1")
	if got := fmt.Sprintf("%.*f", 4096, n); got != "1."+strings.Repeat("0", 4096) {
		t.Fatalf("maximum precision: length %d, prefix %.40s", len(got), got)
	}
	if got := fmt.Sprintf("%.*f", 4097, n); !strings.Contains(got, "precision exceeds") {
		t.Fatalf("excessive precision = %q", got)
	}
	if got := fmt.Sprintf("%*v", 16384, n); len(got) != 16384 || !strings.HasSuffix(got, "1") {
		t.Fatalf("maximum width: length %d", len(got))
	}
	if got := fmt.Sprintf("%*v", 16385, n); !strings.Contains(got, "width exceeds") {
		t.Fatalf("excessive width = %q", got)
	}
}

func TestFormatDisplayPaddingBeyondStoredCapacity(t *testing.T) {
	integer := strings.Repeat("9", 4096)
	n := FromString(integer)
	if !n.Ok() {
		t.Fatal(n.Err)
	}
	want := integer + "." + strings.Repeat("0", 4096)
	if got := fmt.Sprintf("%.*f", 4096, n); got != want {
		t.Fatalf("large display: length %d, want %d; prefix %.80s", len(got), len(want), got)
	}
	if !n.Ok() || n.String() != integer {
		t.Fatal("display padding changed the stored value")
	}
}

func TestFormatConcurrentValueCopies(t *testing.T) {
	const integer = "1234567890123456789012345678901234567890"
	n := FromString(integer + ".125")
	if !n.Ok() {
		t.Fatal(n.Err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			copy := n
			for range 32 {
				if got := fmt.Sprintf("%+.2f", copy); got != "+"+integer+".12" {
					t.Errorf("concurrent rounding = %q", got)
					return
				}
				if got := fmt.Sprintf("%.*f", 40, copy); got != integer+".125"+strings.Repeat("0", 37) {
					t.Errorf("concurrent padding = %q", got)
					return
				}
			}
		})
	}
	workers.Wait()
	if got := n.String(); got != integer+".125" {
		t.Fatalf("concurrent formatting changed the input to %q", got)
	}
}

func TestFormatOutputBounds(t *testing.T) {
	// Width is measured in runes, but the separate output bound counts bytes.
	n := Num{Err: errors.New(strings.Repeat("界", 1000))}
	if got := fmt.Sprintf("%*s", 16384, n); !strings.Contains(got, "output exceeds") {
		t.Fatalf("excessive UTF-8 output: length %d, prefix %.80s", len(got), got)
	}
	n.Err = errors.New(strings.Repeat("x", 16384))
	if got := fmt.Sprintf("%q", n); !strings.Contains(got, "output exceeds") {
		t.Fatalf("excessive quoted output: length %d, prefix %.80s", len(got), got)
	}
	n.Err = errors.New(strings.Repeat("界", 16384))
	got := fmt.Sprintf("%f", n)
	if !strings.HasPrefix(got, "%!f(num.Num=") || len(got) > 16384 || !utf8.ValidString(got) {
		t.Fatalf("unbounded diagnostic: length %d, valid UTF-8 %v", len(got), utf8.ValidString(got))
	}
}
