package remoteregion

import (
	"slices"
	"testing"

	"github.com/strelov1/freehire/internal/enrich"
)

// TestMap pins the curated best-effort dictionary against a representative slice
// of the closed set of region strings the remote-companies dataset carries. Each
// case is one source string → the macro-region codes it should resolve to (sorted,
// de-duplicated). The dictionary never guesses: a string it cannot place resolves
// to no regions.
func TestMap(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		// Clean single labels.
		{"Worldwide", []string{"global"}},
		{"USA", []string{"north_america"}},
		{"North America", []string{"north_america"}},
		{"Canada", []string{"north_america"}},
		{"Europe", []string{"eu"}},
		{"UK", []string{"uk"}},
		{"Brazil", []string{"latam"}},
		{"Latin America", []string{"latam"}},
		{"India", []string{"apac"}},
		{"Asia", []string{"apac"}},
		{"Japan", []string{"apac"}},
		{"Australia", []string{"apac"}},
		{"Singapore", []string{"apac"}},
		{"Indonesia", []string{"apac"}},
		{"Republic of Korea", []string{"apac"}},
		{"South Africa", []string{"africa"}},
		{"Germany", []string{"eu"}},
		{"France", []string{"eu"}},
		{"Spain", []string{"eu"}},
		{"Poland", []string{"eu"}},
		{"Italy", []string{"eu"}},
		{"Ireland", []string{"eu"}},
		{"Switzerland", []string{"eu"}},
		{"EU", []string{"eu"}},

		// Composite labels split on separators and map component-wise.
		{"Americas", []string{"latam", "north_america"}},
		{"Europe, Americas", []string{"eu", "latam", "north_america"}},
		{"USA, Canada", []string{"north_america"}},
		{"North and Latin America", []string{"latam", "north_america"}},
		{"USA, Europe", []string{"eu", "north_america"}},
		{"USA, UK", []string{"north_america", "uk"}},
		{"UK, Europe", []string{"eu", "uk"}},
		{"US, Asia, Europe", []string{"apac", "eu", "north_america"}},
		{"USA, Europe, APAC", []string{"apac", "eu", "north_america"}},
		{"USA, LATAM", []string{"latam", "north_america"}},
		{"USA, CA, UK, DE, FR, NL, AU, JPN", []string{"apac", "eu", "north_america", "uk"}},
		{"USA, CA, UK, BG, PH, CO", []string{"apac", "eu", "latam", "north_america", "uk"}},
		{"Asia, Africa, Europe, South America, USA", []string{"africa", "apac", "eu", "latam", "north_america"}},
		{"Germany, The Netherlands, Spain, Chile", []string{"eu", "latam"}},

		// Resolved judgment call: Ukraine → cis (post-Soviet), not eu.
		{"Ukraine, Poland", []string{"cis", "eu"}},

		// global is absorbing: any worldwide component collapses to [global].
		{"USA, Worldwide", []string{"global"}},
		{"USA / Worldwide", []string{"global"}},
		{"Worldwide, Primarily USA", []string{"global"}},

		// Timezone handling: narrow offset → one region, wide Americas→Europe span
		// → the two nearest macro regions, all-negative span → the Americas.
		{"UTC+2", []string{"eu"}},
		{"UTC+2 +- 2", []string{"eu"}},
		{"UTC-8 to UTC+2", []string{"eu", "north_america"}},
		{"UTC-10 to UTC+2", []string{"eu", "north_america"}},
		{"UTC-8 to UTC+1", []string{"eu", "north_america"}},
		{"UTC -3 to -5", []string{"latam", "north_america"}},
		{"UTC+1 to UTC+2", []string{"eu"}},          // all-positive span stays in Europe
		{"UTC+2 to UTC+9", []string{"apac", "eu"}},  // span reaching into Asia
		{"Asia-Pacific", []string{"apac"}},          // "pacific" must not add north_america
		{"Pacific Time Zone (PT)", []string{"north_america"}},
		{"PST Timezone", []string{"north_america"}},
		{"Central Europe Time (CET) +/1h", []string{"eu"}},
		{"Europe (CET -3 / CET +3)", []string{"eu"}},
		{"Europe UTC-1 to UTC+2", []string{"eu"}},
		{"Europe or comparable timezone", []string{"eu"}},
		{"East American / European / African timezones", []string{"africa", "eu", "north_america"}},
		{"USA and EMEA/EST Timezones", []string{"africa", "eu", "mena", "north_america"}},

		// Narrow geography maps to the nearest macro region.
		{"USA East Coast", []string{"north_america"}},
		{"Western North America", []string{"north_america"}},
		{"Western Asia", []string{"mena"}},

		// Never guesses: an unplaceable label resolves to no regions.
		{"Atlantis", nil},
		{"", nil},
	}

	for _, tc := range cases {
		got := Map(tc.raw)
		if !slices.Equal(got, tc.want) {
			t.Errorf("Map(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// TestMapOutputIsConfinedAndCanonical guards the invariant that every code Map
// emits is a member of enrich.RegionValues, and that the output is sorted and
// de-duplicated regardless of input shape.
func TestMapOutputIsConfinedAndCanonical(t *testing.T) {
	inputs := []string{
		"Worldwide", "USA, Europe", "Ukraine, Poland",
		"USA, CA, UK, DE, FR, NL, AU, JPN", "UTC-8 to UTC+2",
		"USA and EMEA/EST Timezones", "Atlantis",
	}
	for _, in := range inputs {
		got := Map(in)
		if !slices.IsSorted(got) {
			t.Errorf("Map(%q) = %v is not sorted", in, got)
		}
		for i := 1; i < len(got); i++ {
			if got[i] == got[i-1] {
				t.Errorf("Map(%q) = %v has a duplicate %q", in, got, got[i])
			}
		}
		for _, code := range got {
			if !slices.Contains(enrich.RegionValues, code) {
				t.Errorf("Map(%q) emitted %q, not in enrich.RegionValues", in, code)
			}
		}
	}
}
