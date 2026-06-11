// Package pipeline is the ingest write path: it dispatches each configured board to
// its source adapter, normalizes the postings, and persists them. It is independent
// of the DB layer (via Store) and of any specific platform (via the source registry).
package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/strelov1/freehire/internal/normalize"
	"github.com/strelov1/freehire/internal/sources"
)

// defaultConcurrency bounds how many boards are fetched at once when the Runner does
// not set Concurrency.
const defaultConcurrency = 8

// Job is a normalized posting ready to persist: the pipeline has set the platform as
// source, namespaced the external id by board, and derived the company slug.
type Job struct {
	Source      string
	ExternalID  string
	URL         string
	Title       string
	Company     string
	CompanySlug string
	Location    string
	Remote      bool
	Description string
	PostedAt    *time.Time
}

// Store persists one normalized job and enqueues it for enrichment when needed,
// atomically. The pipeline is unaware of the schema version or the outbox — that is
// the Store implementation's concern.
type Store interface {
	Save(ctx context.Context, job Job) error
}

// Stats reports what a run did: Ingested counts saved jobs, Failed counts boards that
// errored (unknown provider or a fetch failure).
type Stats struct {
	Ingested int
	Failed   int
}

// Runner drives ingest: for each configured board it looks up the adapter, fetches,
// normalizes, and saves. Boards run concurrently up to Concurrency; a board failure is
// isolated and never aborts the run.
type Runner struct {
	Registry    map[string]sources.Source
	Store       Store
	Concurrency int
}

// Run ingests every configured board and returns the aggregate stats. It returns an
// error only for a context cancellation, never for a single board's failure.
func (r Runner) Run(ctx context.Context, entries []sources.CompanyEntry) (Stats, error) {
	limit := r.Concurrency
	if limit <= 0 {
		limit = defaultConcurrency
	}

	var (
		mu    sync.Mutex
		stats Stats
		wg    sync.WaitGroup
	)
	sem := make(chan struct{}, limit)

	for _, e := range entries {
		wg.Add(1)
		go func(e sources.CompanyEntry) {
			defer wg.Done()

			// Acquire a slot, but abandon the board if the run is cancelled — both
			// before starting and while parked waiting for a slot.
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}

			ingested, failed := r.ingestBoard(ctx, e)

			mu.Lock()
			stats.Ingested += ingested
			stats.Failed += failed
			mu.Unlock()
		}(e)
	}
	wg.Wait()

	return stats, ctx.Err()
}

// ingestBoard fetches and saves one board, returning how many jobs it ingested and
// whether the board itself failed (1) or not (0). A missing adapter or a fetch error
// fails the board; a per-job save error is skipped without failing the board.
func (r Runner) ingestBoard(ctx context.Context, e sources.CompanyEntry) (ingested, failed int) {
	src, ok := r.Registry[e.Provider]
	if !ok {
		return 0, 1
	}

	raw, err := src.Fetch(ctx, e)
	if err != nil {
		return 0, 1
	}

	for _, j := range raw {
		if err := r.Store.Save(ctx, normalizeJob(e, j)); err != nil {
			continue
		}
		ingested++
	}
	return ingested, 0
}

// normalizeJob turns a raw posting into a persistable Job: the platform becomes the
// source, the external id is namespaced by board so two companies on one platform
// cannot collide, and the company slug is derived with the shared normalizer.
func normalizeJob(e sources.CompanyEntry, j sources.Job) Job {
	return Job{
		Source:      e.Provider,
		ExternalID:  fmt.Sprintf("%s:%s", e.Board, j.ExternalID),
		URL:         j.URL,
		Title:       j.Title,
		Company:     j.Company,
		CompanySlug: normalize.Slug(j.Company),
		Location:    j.Location,
		Remote:      j.Remote,
		Description: j.Description,
		PostedAt:    j.PostedAt,
	}
}
