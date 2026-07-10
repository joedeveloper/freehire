//go:build integration

// Integration test for the hydrating-source seen-set query: ExistingExternalIDs returns the
// external_ids stored for one provider (and only that provider), so a hydrating adapter fetches
// per-posting detail only for postings the catalogue lacks. A SQL behavior, verifiable only
// against a real Postgres. Run with: go test -tags=integration ./internal/db/
package db

import (
	"context"
	"slices"
	"testing"
)

func TestExistingExternalIDsScopedToProvider(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()
	truncate(t, pool)

	// Two greenhouse rows and one lever row.
	for _, ext := range []string{"acme:1", "acme:2"} {
		if _, err := ingestUpsert(ctx, q, ingestParams(ext, "Engineer")); err != nil {
			t.Fatalf("upsert %s: %v", ext, err)
		}
	}
	lever := ingestParams("other:9", "Designer")
	lever.Source = "lever"
	if _, err := ingestUpsert(ctx, q, lever); err != nil {
		t.Fatalf("upsert lever: %v", err)
	}

	ids, err := q.ExistingExternalIDs(ctx, "greenhouse")
	if err != nil {
		t.Fatalf("ExistingExternalIDs: %v", err)
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []string{"acme:1", "acme:2"}) {
		t.Errorf("ids = %v, want [acme:1 acme:2] (lever row excluded)", ids)
	}
}
