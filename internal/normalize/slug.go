// Package normalize derives normalized keys from raw source data. It is the
// home for the pipeline's name-to-slug normalization.
package normalize

import (
	"strings"
	"unicode"
)

// Slug turns a company name into its natural key: lowercased, with each run of
// non-alphanumeric characters collapsed to a single hyphen and leading/trailing
// hyphens trimmed. Unicode letters and digits are preserved, so non-Latin names
// (e.g. "Яндекс") keep their script. An empty or punctuation-only name yields an
// empty slug, which the write path treats as "no company".
//
// It deliberately does not strip legal suffixes (LLC, Inc, ООО); that is a noted
// future refinement.
func Slug(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case b.Len() > 0 && !prevHyphen:
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
