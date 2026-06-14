// Package classify derives a job's seniority and role category deterministically
// from its title. It is a curated dictionary, not a model: it resolves known
// English and Russian title terms and emits nothing for what it cannot resolve
// (it never guesses). Canonical values are drawn from the same controlled
// vocabularies the enrichment contract defines (enrich.SeniorityValues /
// enrich.CategoryValues), so the parser, the enrichment payload, and the search
// facet all speak one set of values — the same doctrine as internal/location.
package classify

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Classification is the seniority and role category parsed from a job title.
// Each field is "" when the title states nothing the dictionary resolves.
type Classification struct {
	Seniority string // "" or one of enrich.SeniorityValues
	Category  string // "" or one of enrich.CategoryValues
}

// Parse resolves a job title to its seniority and category. It never guesses;
// an unresolved field is "".
func Parse(title string) Classification {
	lower := strings.ToLower(title)
	return Classification{
		Seniority: matchOrdered(lower, seniorityOrder, seniorityAliases),
		Category:  matchOrdered(lower, categoryOrder, categoryAliases),
	}
}

// matchOrdered returns the canonical value of the first alias (in priority order)
// that occurs as a whole word in title, or "" if none match. Ordering encodes
// precedence: the most specific / highest-rank alias is checked first, so a title
// carrying several grade words ("Lead Senior") resolves the stronger one.
func matchOrdered(title string, order []string, aliases map[string]string) string {
	for _, alias := range order {
		if containsWord(title, alias) {
			return aliases[alias]
		}
	}
	return ""
}

// containsWord reports whether needle occurs in haystack bounded by non-word
// runes on both sides, so "lead" does not match inside "leading" and "react"
// does not match inside "reactor". Word runes are Unicode letters/digits, so
// Cyrillic boundaries are handled the same as Latin.
func containsWord(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(haystack[from:], needle)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(needle)
		if boundaryOK(haystack, start, end) {
			return true
		}
		from = start + 1
	}
}

func boundaryOK(s string, start, end int) bool {
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(s[:start]); isWordRune(r) {
			return false
		}
	}
	if end < len(s) {
		if r, _ := utf8.DecodeRuneInString(s[end:]); isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
