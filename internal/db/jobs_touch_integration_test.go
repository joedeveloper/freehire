//go:build integration

// Integration test for the hydrating-source liveness refresh: TouchJob refreshes last_seen_at
// and reopens a closed row WITHOUT rewriting its content, so a re-listed already-ingested
// posting keeps the description/facets it was hydrated with. Verifiable only against a real
// Postgres. Run with: go test -tags=integration ./internal/db/
package db

import (
	"context"
	"testing"
	"time"
)

func TestTouchJobRefreshesLivenessAndPreservesContent(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()
	truncate(t, pool)

	// Ingest a job that carries a description and skills (as a hydrated justjoin row would).
	p := ingestParams("justjoin:1", "Engineer")
	p.Source = "justjoin"
	p.Description = "Rich hydrated body."
	p.Skills = []string{"go", "typescript"}
	if _, err := ingestUpsert(ctx, q, p); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Simulate a stale, closed row: last_seen_at far in the past, closed_at set.
	if _, err := pool.Exec(ctx,
		`UPDATE jobs SET last_seen_at = now() - interval '10 days', closed_at = now() WHERE source='justjoin' AND external_id='justjoin:1'`,
	); err != nil {
		t.Fatalf("stale/close setup: %v", err)
	}

	companySlug, err := q.TouchJob(ctx, TouchJobParams{Source: "justjoin", ExternalID: "justjoin:1"})
	if err != nil {
		t.Fatalf("TouchJob: %v", err)
	}
	// The touched row's company is returned so the caller can keep it in the sweep scope.
	if companySlug != "acme" {
		t.Errorf("TouchJob company_slug = %q, want acme", companySlug)
	}

	got, err := q.GetJobBySourceExternalID(ctx, GetJobBySourceExternalIDParams{Source: "justjoin", ExternalID: "justjoin:1"})
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	if got.ClosedAt.Valid {
		t.Error("closed_at still set, want reopened (NULL)")
	}
	if time.Since(got.LastSeenAt.Time) > time.Minute {
		t.Errorf("last_seen_at = %v, want refreshed to ~now", got.LastSeenAt.Time)
	}
	// Content must be untouched.
	if got.Description != "Rich hydrated body." {
		t.Errorf("Description = %q, want preserved", got.Description)
	}
	if len(got.Skills) != 2 {
		t.Errorf("Skills = %v, want preserved [go typescript]", got.Skills)
	}
}
