package skilltag

import "testing"

// TestDictionaryInvariants guards the two properties the engine relies on:
// every canonical is a stable slug (lowercase, no spaces), and the vocabulary is
// at least the launch floor so an accidental truncation is caught.
func TestDictionaryInvariants(t *testing.T) {
	for alias, c := range wordAliases {
		assertSlug(t, "wordAliases["+alias+"]", c)
	}
	for _, p := range phraseAliases {
		assertSlug(t, "phraseAliases "+p.alias, p.canonical)
	}
	if got := len(wordAliases) + len(phraseAliases); got < 200 {
		t.Errorf("vocabulary size = %d, want >= 200 (launch floor)", got)
	}
	// Ambiguous English words must never be bare word aliases (they resolve only
	// via an unambiguous alias or a phrase).
	for _, w := range []string{"go", "c", "r"} {
		if _, ok := wordAliases[w]; ok {
			t.Errorf("ambiguous word %q must not be a wordAliases key", w)
		}
	}
}

func assertSlug(t *testing.T, what, s string) {
	t.Helper()
	if s == "" || s != trimLower(s) {
		t.Errorf("%s: canonical %q is not a lowercase no-space slug", what, s)
	}
}
