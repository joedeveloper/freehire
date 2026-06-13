package location

import (
	"reflect"
	"slices"
	"testing"

	"github.com/strelov1/freehire/internal/enrich"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     Geo
	}{
		{
			name:     "named country yields code, region and remote mode",
			location: "Remote - Germany",
			want:     Geo{Countries: []string{"de"}, Regions: []string{"eu"}, WorkMode: "remote"},
		},
		{
			name:     "country shorthand USA",
			location: "Remote - USA",
			want:     Geo{Countries: []string{"us"}, Regions: []string{"us"}, WorkMode: "remote"},
		},
		{
			name:     "plain country name states no work mode",
			location: "United States",
			want:     Geo{Countries: []string{"us"}, Regions: []string{"us"}},
		},
		{
			name:     "macro region name yields region without country",
			location: "Remote - Europe",
			want:     Geo{Regions: []string{"eu"}, WorkMode: "remote"},
		},
		{
			name:     "multiple locations union and dedup",
			location: "Remote - UK or Europe",
			want:     Geo{Countries: []string{"gb"}, Regions: []string{"eu", "uk"}, WorkMode: "remote"},
		},
		{
			name:     "bare remote yields work mode but no geography",
			location: "Remote",
			want:     Geo{WorkMode: "remote"},
		},
		{
			name:     "explicit anywhere yields global and remote",
			location: "Remote - Anywhere",
			want:     Geo{Regions: []string{"global"}, WorkMode: "remote"},
		},
		{
			name:     "hybrid marker with city",
			location: "Hybrid - London",
			want:     Geo{Countries: []string{"gb"}, Regions: []string{"uk"}, WorkMode: "hybrid"},
		},
		{
			name:     "onsite marker in parentheses keeps the city",
			location: "Berlin (On-site)",
			want:     Geo{Countries: []string{"de"}, Regions: []string{"eu"}, WorkMode: "onsite"},
		},
		{
			name:     "hybrid wins over a remote marker in the same string",
			location: "Hybrid / Remote - London",
			want:     Geo{Countries: []string{"gb"}, Regions: []string{"uk"}, WorkMode: "hybrid"},
		},
		{
			name:     "country buried among unknown tokens",
			location: "Burlington, Massachusetts, United States; Remote",
			want:     Geo{Countries: []string{"us"}, Regions: []string{"us"}, WorkMode: "remote"},
		},
		{
			name:     "empty location",
			location: "",
			want:     Geo{},
		},
		{
			name:     "unresolvable token guesses nothing",
			location: "Atlantis",
			want:     Geo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.location)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.location, got, tt.want)
			}
		})
	}
}

// TestParseEmitsOnlyKnownVocabulary guards the controlled-vocabulary invariant:
// every region the parser emits is a member of enrich.RegionValues and every
// work mode a member of enrich.WorkModeValues — the parser never invents a value
// outside the enrichment contract's vocabularies.
func TestParseEmitsOnlyKnownVocabulary(t *testing.T) {
	samples := []string{
		"Remote - Germany", "Remote - UK or Europe", "Remote - Anywhere",
		"Remote - USA", "Remote - Singapore", "Remote - Canada",
		"Hybrid - London", "Berlin (On-site)",
		"Burlington, Massachusetts, United States; Remote", "Remote", "",
	}
	for _, s := range samples {
		got := Parse(s)
		for _, r := range got.Regions {
			if !slices.Contains(enrich.RegionValues, r) {
				t.Errorf("Parse(%q) emitted region %q outside RegionValues", s, r)
			}
		}
		if got.WorkMode != "" && !slices.Contains(enrich.WorkModeValues, got.WorkMode) {
			t.Errorf("Parse(%q) emitted work_mode %q outside WorkModeValues", s, got.WorkMode)
		}
	}
}
