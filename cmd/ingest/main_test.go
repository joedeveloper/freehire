package main

import (
	"testing"

	"github.com/strelov1/freehire/internal/pipeline"
)

// The sweep guard: closing stale jobs is only safe after a run that actually saw
// postings — a zero-ingest run (total crawl outage) must not trigger the sweep.
func TestShouldSweep(t *testing.T) {
	cases := []struct {
		name  string
		stats pipeline.Stats
		want  bool
	}{
		{"normal run", pipeline.Stats{Ingested: 100, Failed: 3}, true},
		{"zero ingested", pipeline.Stats{Ingested: 0, Failed: 550}, false},
		{"all boards ok but empty", pipeline.Stats{Ingested: 0, Failed: 0}, false},
	}
	for _, tc := range cases {
		if got := shouldSweep(tc.stats); got != tc.want {
			t.Errorf("%s: shouldSweep = %v, want %v", tc.name, got, tc.want)
		}
	}
}
