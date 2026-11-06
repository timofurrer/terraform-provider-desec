// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package txtrdata

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{`"hello"`, []string{"hello"}},
		{`"a" "b"`, []string{"a", "b"}},
		{`"with \"quote\""`, []string{`with "quote"`}},
		{`"back\\slash"`, []string{`back\slash`}},
		{`"\195\164"`, []string{"ä"}},
		{`unquoted "quoted"`, []string{"unquoted", "quoted"}},
		{`""`, []string{""}},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if len(got) != len(tt.want) {
			t.Fatalf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
		}
		for i := range got {
			if string(got[i]) != tt.want[i] {
				t.Errorf("Parse(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{`"unterminated`, `"dangling\`, `"\999"`} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): expected error", in)
		}
	}
}

func TestCanonicalLongString(t *testing.T) {
	long := strings.Repeat("a", 300) + strings.Repeat("b", 120)
	user := `"` + long + `"`
	desec := `"` + long[:255] + `" "` + long[255:] + `"`

	if Canonical(user) != Canonical(desec) {
		t.Errorf("expected long string and its split form to be equivalent:\n%s\n%s", Canonical(user), Canonical(desec))
	}
	if Canonical(desec) != desec {
		t.Errorf("Canonical(%q) = %q, want unchanged", desec, Canonical(desec))
	}
}

func TestCanonicalDistinguishesShortStrings(t *testing.T) {
	if Canonical(`"ab"`) == Canonical(`"a" "b"`) {
		t.Error(`"ab" and "a" "b" must not be considered equivalent`)
	}
}

func TestFormatRoundTrip(t *testing.T) {
	in := `"x\"y\\z" "\195\164" "plain"`
	strs, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := Format(strs); got != in {
		t.Errorf("Format(Parse(%q)) = %q", in, got)
	}
}
