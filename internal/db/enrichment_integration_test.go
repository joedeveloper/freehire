//go:build integration

// Integration tests for the enrichment_outbox queue semantics — claim/lease,
// idempotent enqueue, and dead-lettering — which are SQL behavior and can only be
// verified against a real Postgres. Run with: go test -tags=integration ./internal/db/
// Requires Docker (testcontainers spins up a throwaway Postgres with the migrations).
package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const targetVersion int32 = 1

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	var scripts []string
	for _, f := range []string{
		"0001_init.sql", "0002_companies.sql",
		"0003_job_enrichment.sql", "0004_enrichment_outbox.sql",
		"0005_users.sql", "0006_user_jobs.sql",
	} {
		abs, err := filepath.Abs(filepath.Join("..", "..", "migrations", f))
		if err != nil {
			t.Fatalf("resolve migration path: %v", err)
		}
		scripts = append(scripts, abs)
	}

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("hire"),
		postgres.WithUsername("hire"),
		postgres.WithPassword("hire"),
		postgres.WithInitScripts(scripts...),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertJob(t *testing.T, pool *pgxpool.Pool, externalID string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (source, external_id, url, title)
		 VALUES ('test', $1, 'http://example.test', 'A job') RETURNING id`,
		externalID).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"TRUNCATE enrichment_outbox, jobs, companies RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestEnrichmentQueue(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()

	t.Run("enqueue is idempotent", func(t *testing.T) {
		truncate(t, pool)
		insertJob(t, pool, "idem")

		for i := 0; i < 2; i++ {
			if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
		}
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM enrichment_outbox").Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("outbox rows = %d, want 1 (one per (job_id, target_version))", n)
		}
	})

	t.Run("claim leases entries so concurrent claims are disjoint", func(t *testing.T) {
		truncate(t, pool)
		insertJob(t, pool, "j1")
		insertJob(t, pool, "j2")
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}

		first, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 1})
		if err != nil || len(first) != 1 {
			t.Fatalf("first claim: rows=%d err=%v, want 1", len(first), err)
		}
		second, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(second) != 1 {
			t.Fatalf("second claim: rows=%d err=%v, want 1 (the other entry)", len(second), err)
		}
		if first[0].ID == second[0].ID {
			t.Errorf("both claims returned outbox id %d — not disjoint", first[0].ID)
		}
		third, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(third) != 0 {
			t.Errorf("third claim: rows=%d, want 0 (all leased)", len(third))
		}
	})

	t.Run("a stale lease is reclaimable", func(t *testing.T) {
		truncate(t, pool)
		insertJob(t, pool, "stale")
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}

		if c, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Fatalf("claim: rows=%d err=%v, want 1", len(c), err)
		}
		// Still within the lease → not reclaimable.
		if c, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10}); err != nil || len(c) != 0 {
			t.Fatalf("re-claim within lease: rows=%d, want 0", len(c))
		}
		// Lease of 0s → the prior claim is now stale and reclaimable.
		if c, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 0, BatchSize: 10}); err != nil || len(c) != 1 {
			t.Errorf("re-claim with expired lease: rows=%d err=%v, want 1", len(c), err)
		}
	})

	t.Run("attempts reaching max dead-letters the entry", func(t *testing.T) {
		truncate(t, pool)
		insertJob(t, pool, "dead")
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}
		claimed, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim: rows=%d err=%v, want 1", len(claimed), err)
		}
		id := claimed[0].ID

		first, err := q.RecordEnrichmentFailure(ctx, RecordEnrichmentFailureParams{LastError: "boom", MaxAttempts: 2, ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if first.Attempts != 1 || first.FailedAt.Valid {
			t.Errorf("after 1st failure: attempts=%d failed=%v, want 1/not-dead", first.Attempts, first.FailedAt.Valid)
		}
		second, err := q.RecordEnrichmentFailure(ctx, RecordEnrichmentFailureParams{LastError: "boom", MaxAttempts: 2, ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if second.Attempts != 2 || !second.FailedAt.Valid {
			t.Errorf("after 2nd failure: attempts=%d failed=%v, want 2/dead-lettered", second.Attempts, second.FailedAt.Valid)
		}
		// Dead-lettered → never claimed again, even with an expired lease.
		if c, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 0, BatchSize: 10}); err != nil || len(c) != 0 {
			t.Errorf("claim after dead-letter: rows=%d, want 0", len(c))
		}
	})
}
