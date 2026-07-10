package sources

import (
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	in := `<h2>Role</h2><p>Lead the <strong>backend</strong> team.</p>` +
		`<ul><li>Ship features</li></ul>` +
		`<a href="https://example.com" onclick="steal()">apply</a>` +
		`<img src="https://evil.example/track.gif">` +
		`<script>alert(1)</script>`

	got := sanitizeHTML(in)

	// Structural formatting is preserved.
	for _, want := range []string{"<h2>Role</h2>", "<strong>backend</strong>", "<li>Ship features</li>", `href="https://example.com"`} {
		if !strings.Contains(got, want) {
			t.Errorf("sanitizeHTML dropped expected markup %q\ngot: %s", want, got)
		}
	}

	// Active content and external request vectors are stripped.
	for _, bad := range []string{"<script", "onclick", "alert(1)", "<img", "track.gif"} {
		if strings.Contains(got, bad) {
			t.Errorf("sanitizeHTML kept unsafe content %q\ngot: %s", bad, got)
		}
	}

	// Links are defanged so untrusted postings cannot pass link authority.
	if !strings.Contains(got, `rel="nofollow"`) {
		t.Errorf("sanitizeHTML should mark links nofollow\ngot: %s", got)
	}
}

// Some ATS boards emit descriptions whose words are glued by non-breaking spaces
// (U+00A0, often as the &nbsp; entity) instead of regular spaces. Rendered with
// {@html} this becomes one unbreakable line that overflows the page horizontally,
// so the sanitizer normalizes no-break spaces to regular ones, restoring word-boundary
// wrapping. bluemonday decodes &nbsp; to the raw U+00A0 rune, so normalizing the
// sanitized output catches both the entity and raw-character forms.
func TestSanitizeHTMLNormalizesNoBreakSpaces(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"entity form":   {"<p>Java&nbsp;Spring&nbsp;Boot</p>", "<p>Java Spring Boot</p>"},
		"raw U+00A0":    {"<p>Java Spring Boot</p>", "<p>Java Spring Boot</p>"},
		"narrow U+202F": {"<p>5 years</p>", "<p>5 years</p>"},
	}
	for name, c := range cases {
		if got := sanitizeHTML(c.in); got != c.want {
			t.Errorf("%s: sanitizeHTML(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}

func TestLenientPercentUnescape(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"plain":              {"hello world", "hello world"},
		"valid escapes":      {"%3Cp%3Ehi%3C%2Fp%3E", "<p>hi</p>"},
		"literal percent":    {"line-height:115%;color", "line-height:115%;color"},
		"stat percent":       {"100% remote", "100% remote"},
		"mixed":              {"%3Cb%3E100%25 %3D%3E all%3C%2Fb%3E", "<b>100% => all</b>"},
		"plus preserved":     {"C%2B%2B and C++", "C++ and C++"},
		"trailing lone pct":  {"done 50%", "done 50%"},
		"lone pct then hex1": {"%3 only", "%3 only"},
		"utf8 bytes":         {"%D0%9F%D1%80%D0%B8%D0%B2%D0%B5%D1%82", "Привет"},
	}
	for name, c := range cases {
		if got := LenientPercentUnescape(c.in); got != c.want {
			t.Errorf("%s: LenientPercentUnescape(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}
