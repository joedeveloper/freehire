// Package remoteregion maps the free-text "where a company hires remotely" label
// carried by the remote-companies dataset to the macro-region codes of the
// enrichment contract (enrich.RegionValues). It is a curated, best-effort
// dictionary — a sibling of internal/location — over a closed set of source
// strings: clean labels map directly, composite labels split and map
// component-wise, and timezone or narrow-geography labels resolve to the nearest
// macro region. A label the dictionary cannot place resolves to no regions
// (never a guess). Output is always sorted and de-duplicated.
package remoteregion

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Macro-region codes, aliased for readability of the rule tables below. Each is a
// member of enrich.RegionValues (pinned by TestMapOutputIsConfinedAndCanonical).
const (
	global = "global"
	na     = "north_america"
	latam  = "latam"
	eu     = "eu"
	uk     = "uk"
	mena   = "mena"
	africa = "africa"
	apac   = "apac"
	cis    = "cis"
)

// exactCodes maps a whole token that is a short country/region code to its macro
// region. These are matched by exact token equality (not substring) because two-
// letter codes like "us" or "it" occur as substrings of unrelated words.
var exactCodes = map[string]string{
	"us": na, "usa": na, "ca": na,
	"uk": uk,
	"eu": eu, "latam": latam,
	"de": eu, "fr": eu, "nl": eu, "it": eu, "bg": eu, "es": eu, "pl": eu, "pt": eu,
	"br": latam, "co": latam,
	"au": apac, "jp": apac, "jpn": apac, "chn": apac, "cn": apac,
	// SK here is South Korea (it appears paired with SG/Singapore in the dataset),
	// not Slovakia — Slovakia arrives as the full word and maps to eu via substrRules.
	"sk": apac, "sg": apac, "ph": apac, "in": apac, "apac": apac,
}

// substrRule is a phrase/name matched as a substring of a token. Rules are applied
// most-specific first and the matched text is consumed, so a specific phrase
// ("western asia" → mena) blocks a generic substring it contains ("asia" → apac).
type substrRule struct {
	kw      string
	regions []string
}

// substrRules is ordered longest/most-specific first.
var substrRules = []substrRule{
	{"north and latin america", []string{na, latam}},
	{"western north america", []string{na}},
	{"western asia", []string{mena}},
	{"western europe", []string{eu}},
	{"east american", []string{na}},
	{"latin america", []string{latam}},
	{"south america", []string{latam}},
	{"central america", []string{latam}},
	{"north america", []string{na}},
	{"south africa", []string{africa}},
	{"republic of korea", []string{apac}},
	{"united states", []string{na}},
	{"united kingdom", []string{uk}},
	{"netherlands", []string{eu}},
	{"switzerland", []string{eu}},
	{"australia", []string{apac}},
	{"singapore", []string{apac}},
	{"indonesia", []string{apac}},
	{"philippines", []string{apac}},
	{"colombia", []string{latam}},
	{"portugal", []string{eu}},
	{"slovakia", []string{eu}},
	{"bulgaria", []string{eu}},
	{"germany", []string{eu}},
	{"ukraine", []string{cis}},
	{"greece", []string{eu}},
	{"ireland", []string{eu}},
	{"dublin", []string{eu}},
	{"austria", []string{eu}},
	{"france", []string{eu}},
	{"poland", []string{eu}},
	{"mexico", []string{latam}},
	{"canada", []string{na}},
	{"brazil", []string{latam}},
	{"chile", []string{latam}},
	{"israel", []string{mena}},
	{"pacific time", []string{na}},
	{"americas", []string{na, latam}},
	{"european", []string{eu}},
	{"europe", []string{eu}},
	{"african", []string{africa}},
	{"africa", []string{africa}},
	{"emea", []string{eu, mena, africa}},
	{"asia", []string{apac}},
	{"india", []string{apac}},
	{"japan", []string{apac}},
	{"korea", []string{apac}},
	{"china", []string{apac}},
	{"oceania", []string{apac}},
	{"spain", []string{eu}},
	{"italy", []string{eu}},
	{"usa", []string{na}},
	{"cet", []string{eu}},
}

// boundaryRule matches a short timezone code on a word boundary, so "est" does not
// fire inside "western". Applied after the substring rules, additively.
type boundaryRule struct {
	re      *regexp.Regexp
	regions []string
}

var boundaryRules = []boundaryRule{
	{regexp.MustCompile(`\best\b`), []string{na}},
	{regexp.MustCompile(`\bpst\b`), []string{na}},
	{regexp.MustCompile(`\bpdt\b`), []string{na}},
	{regexp.MustCompile(`\bgmt\b`), []string{uk}},
}

var offsetRe = regexp.MustCompile(`([+-]?\d+)`)

// Map resolves a free-text region label to sorted, de-duplicated macro-region
// codes. A "worldwide" label is absorbing (resolves to exactly ["global"]); an
// unplaceable label resolves to nil.
func Map(raw string) []string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return nil
	}
	if strings.Contains(s, "worldwide") {
		return []string{global}
	}

	var out []string
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '/' || r == ';'
	}) {
		out = append(out, mapToken(strings.TrimSpace(tok))...)
	}
	return dedupSort(out)
}

// mapToken resolves a single token (one comma/slash/semicolon-separated segment).
func mapToken(tok string) []string {
	if tok == "" {
		return nil
	}
	if r, ok := exactCodes[tok]; ok {
		return []string{r}
	}

	var out []string
	work := tok
	for _, rule := range substrRules {
		if strings.Contains(work, rule.kw) {
			out = append(out, rule.regions...)
			work = strings.ReplaceAll(work, rule.kw, " ")
		}
	}
	if len(out) > 0 {
		return out
	}
	// Fallbacks for a token no geographic keyword placed: a bare timezone code on a
	// word boundary, then a UTC offset/span. Gated on no geo match so a geo token
	// that merely contains a standalone tz word is not double-tagged.
	for _, rule := range boundaryRules {
		if rule.re.MatchString(tok) {
			out = append(out, rule.regions...)
		}
	}
	if len(out) == 0 {
		out = mapTimezone(tok)
	}
	return out
}

// mapTimezone resolves a UTC-offset token that no geographic rule matched. It
// reads both edges of a span: all-negative → the Americas {na, latam}; crossing
// zero (Americas into Europe) → {na, eu}; all-positive within Europe → {eu};
// reaching further east → {eu, apac}. A single offset maps to its one region; a
// token with no UTC offset resolves to nil.
func mapTimezone(tok string) []string {
	if !strings.Contains(tok, "utc") {
		return nil
	}
	nums := offsetRe.FindAllString(tok, -1)
	if len(nums) == 0 {
		return nil
	}
	if strings.Contains(tok, " to ") && len(nums) >= 2 {
		lo, hi := atoi(nums[0]), atoi(nums[1])
		if lo > hi {
			lo, hi = hi, lo
		}
		switch {
		case hi <= 0:
			return []string{na, latam}
		case lo < 0:
			return []string{na, eu}
		case hi <= 3:
			return []string{eu}
		default:
			return []string{eu, apac}
		}
	}
	switch o := atoi(nums[0]); {
	case o < 0:
		return []string{na}
	case o <= 3:
		return []string{eu}
	default:
		return []string{apac}
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// dedupSort returns the codes sorted and de-duplicated, or nil for an empty input.
func dedupSort(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}
	slices.Sort(codes)
	return slices.Compact(codes)
}
