package sources

import "testing"

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		parts []string
		want  string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"  ", "b"}, "b"}, // whitespace-only counts as blank
		{[]string{"", ""}, ""},
		{nil, ""},
		{[]string{" trimmed "}, " trimmed "}, // first non-blank returned verbatim, not trimmed
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.parts...); got != c.want {
			t.Errorf("firstNonEmpty(%q) = %q, want %q", c.parts, got, c.want)
		}
	}
}
