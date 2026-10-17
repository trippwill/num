// SPDX-FileCopyrightText: 2026 github.com/trippwill
// SPDX-License-Identifier: MPL-2.0

package num

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"testing"
)

// Issue #1: parsing must preserve digits beyond the former backend's precision
// so explicit quantization and presentation each round the original value once.
func TestIssue1NoDoubleRounding(t *testing.T) {
	for _, tc := range []struct {
		input string
		scale int
		want  string
	}{
		{"1.00000050000000000001", 6, "1.000001"},
		{"-1.00000050000000000001", 6, "-1.000001"},
		{"1.00500000000000000001", 2, "1.01"},
		{"-1.00500000000000000001", 2, "-1.01"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			for _, path := range []struct {
				name   string
				decode func(*Num) error
			}{
				{"constructor", func(n *Num) error { *n = FromString(tc.input); return n.Err }},
				{"text", func(n *Num) error { return n.UnmarshalText([]byte(tc.input)) }},
				{"json-number", func(n *Num) error { return json.Unmarshal([]byte(tc.input), n) }},
				{"json-string", func(n *Num) error { return json.Unmarshal([]byte(strconv.Quote(tc.input)), n) }},
				{"sql-string", func(n *Num) error { return n.Scan(tc.input) }},
				{"sql-bytes", func(n *Num) error { return n.Scan([]byte(tc.input)) }},
				{"xml", func(n *Num) error { return xml.Unmarshal([]byte("<n>"+tc.input+"</n>"), n) }},
			} {
				t.Run(path.name, func(t *testing.T) {
					var n Num
					if err := path.decode(&n); err != nil {
						t.Fatal(err)
					}
					requireNum(t, n, tc.input)
					requireNum(t, n.Quantize(tc.scale), tc.want)
					if got := fmt.Sprintf("%.*f", tc.scale, n); got != tc.want {
						t.Fatalf("formatted = %q, want %q", got, tc.want)
					}
					requireNum(t, n, tc.input)
				})
			}
		})
	}
}
