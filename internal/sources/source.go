// Package sources holds the modular job-source adapters and the registry that maps
// a platform key to its adapter. Each adapter implements one ATS platform; adding a
// platform is a new file plus one line in All.
package sources

import (
	"context"
	"strings"
	"time"
)

// CompanyEntry is one configured board from sources.yml: the company whose jobs we
// crawl, the platform it uses (Provider), and the platform-specific board id.
type CompanyEntry struct {
	Company  string `yaml:"company"`
	Provider string `yaml:"provider"`
	Board    string `yaml:"board"`
}

// Job is a raw posting as an adapter yields it, before the pipeline normalizes it
// into the catalogue. ExternalID carries the platform's native posting id; the
// pipeline namespaces it by board before persisting.
type Job struct {
	ExternalID  string
	URL         string
	Title       string
	Company     string
	Location    string
	Description string
	Remote      bool
	PostedAt    *time.Time
}

// Source adapts one job-source platform. Provider is the platform key that selects
// the adapter (it matches CompanyEntry.Provider and the stored jobs.source); Fetch
// returns all current postings for one configured board.
type Source interface {
	Provider() string
	Fetch(ctx context.Context, e CompanyEntry) ([]Job, error)
}

// All assembles the registered adapters into a provider-keyed registry, sharing one
// HTTP client across them. Adding a platform is a new adapter plus one line here.
func All(c HTTPClient) map[string]Source {
	return reg(
		NewGreenhouse(c),
		NewLever(c),
		NewAshby(c),
	)
}

// isRemote infers a job's remote flag from its location text. Adapters share it so
// the heuristic stays consistent across platforms.
func isRemote(location string) bool {
	return strings.Contains(strings.ToLower(location), "remote")
}

// parseRFC3339 parses a platform timestamp into a posted_at, returning nil on an
// empty or unparseable value (posted_at is nullable — a missing date is not an error).
func parseRFC3339(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// parseEpochMillis converts a Unix-millisecond timestamp into a posted_at, returning
// nil for a zero value (treated as "no date").
func parseEpochMillis(ms int64) *time.Time {
	if ms == 0 {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

// reg indexes sources by provider key. A duplicate key means two adapters claim the
// same platform — a programming error — so it panics rather than silently dropping one.
func reg(sources ...Source) map[string]Source {
	m := make(map[string]Source, len(sources))
	for _, s := range sources {
		if _, dup := m[s.Provider()]; dup {
			panic("sources: duplicate provider " + s.Provider())
		}
		m[s.Provider()] = s
	}
	return m
}
