// Package skilltag derives a job's technology tags deterministically from its
// free-text (HTML) description.
//
// Like internal/location, it is a curated dictionary, not an extractor: it
// resolves a known vocabulary of languages, frameworks, datastores, and infra by
// alias, and emits nothing for anything it cannot resolve (it never guesses).
// Tokens are lowercase slugs (go, postgresql, react, kubernetes), the same shape
// the enrichment contract's skills field uses, so the parser and the LLM payload
// speak one vocabulary and union cleanly at read time.
package skilltag

import (
	"regexp"
	"sort"
	"strings"
)

// htmlTagRE matches an HTML tag; descriptions are raw ATS HTML, so tags are
// replaced with a space before matching to keep markup tokens (div, href) out of
// the result and to avoid gluing words across a tag boundary.
var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

// wordTokenRE splits normalized text into bare alphanumeric tokens for the word
// pass. Punctuated terms (c++, node.js) are handled separately by the phrase pass.
var wordTokenRE = regexp.MustCompile(`[a-z0-9]+`)

// normalize strips HTML tags, lowercases the text, and trims leading/trailing
// whitespace. Tags are replaced with a space (not empty) to preserve word
// boundaries so "<b>Go</b>Engineer" cannot fuse.
func normalize(text string) string {
	return strings.TrimSpace(strings.ToLower(htmlTagRE.ReplaceAllString(text, " ")))
}

// wordTokens returns the alphanumeric tokens of already-normalized text, in order.
func wordTokens(norm string) []string {
	return wordTokenRE.FindAllString(norm, -1)
}

// Parse scans free text and returns the curated canonical skill slugs it contains,
// sorted and deduplicated. Returns nil when nothing resolves. It strips HTML, runs a
// phrase pass for punctuated/multi-word terms, then a word pass over the bare tokens.
func Parse(text string) []string {
	norm := normalize(text)
	set := map[string]struct{}{}

	for _, p := range phraseAliases {
		if containsTerm(norm, p.alias) {
			set[p.canonical] = struct{}{}
		}
	}
	for _, tok := range wordTokens(norm) {
		if c, ok := wordAliases[tok]; ok {
			set[c] = struct{}{}
		}
	}
	return sortedKeys(set)
}

// containsTerm reports whether term occurs in norm bounded by a non-alphanumeric
// character (or string edge) on each side, so "c++" does not match inside "abc++x"
// and "react native" matches only as a whole phrase. term's own internal
// punctuation/spaces are matched literally.
func containsTerm(norm, term string) bool {
	from := 0
	for {
		i := strings.Index(norm[from:], term)
		if i < 0 {
			return false
		}
		i += from
		if isBounded(norm, i, i+len(term)) {
			return true
		}
		from = i + 1
	}
}

// isBounded reports whether the [start,end) span in s is a standalone term: each
// side is a string edge or a separating (non-alphanumeric) character, with one
// asymmetry — a LEADING '.' is not a valid left boundary. A leading dot means the
// term is the suffix of a larger dotted token (a domain like "foo.asp.net" or
// "use.c#"), which must not match; a TRAILING dot, by contrast, is a sentence
// period ("We use C#.") and is a valid right boundary.
func isBounded(s string, start, end int) bool {
	if alnumAt(s, start-1) || byteAt(s, start-1) == '.' {
		return false
	}
	return !alnumAt(s, end)
}

// byteAt returns s[i], or 0 when i is out of range.
func byteAt(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func alnumAt(s string, i int) bool {
	c := byteAt(s, i)
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

// sortedKeys returns the set's keys ascending, or nil when empty so an absent tag
// list omits cleanly and matches the text[] column default '{}'.
func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
