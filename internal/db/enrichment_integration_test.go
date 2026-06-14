//go:build integration

// Integration tests for the enrichment_outbox queue semantics — claim/lease,
// idempotent enqueue, and dead-lettering — which are SQL behavior and can only be
// verified against a real Postgres. Run with: go test -tags=integration ./internal/db/
// Requires Docker (testcontainers spins up a throwaway Postgres with the migrations).
package db

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const targetVersion int32 = 1

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	// Apply every migration, in name order — the same way Postgres initdb runs
	// the mounted migrations/ dir — so a new migration is never silently missing
	// from the test schema.
	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}
	scripts, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil || len(scripts) == 0 {
		t.Fatalf("list migrations: %v (found %d)", err, len(scripts))
	}
	sort.Strings(scripts)

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
		`INSERT INTO jobs (source, external_id, url, title, public_slug)
		 VALUES ('test', $1, 'http://example.test', 'A job', 'job-' || $1) RETURNING id`,
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

// setPostedAt stamps a job's posted_at so claim-ordering tests can control freshness.
func setPostedAt(t *testing.T, pool *pgxpool.Pool, jobID int64, posted string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"UPDATE jobs SET posted_at = $1 WHERE id = $2", posted, jobID); err != nil {
		t.Fatalf("set posted_at: %v", err)
	}
}

// closeJob soft-closes a job (sets closed_at) so claim/enqueue exclusion can be tested.
func closeJob(t *testing.T, pool *pgxpool.Pool, jobID int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"UPDATE jobs SET closed_at = now() WHERE id = $1", jobID); err != nil {
		t.Fatalf("close job: %v", err)
	}
}

func TestEnrichmentClaimPriority(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()

	t.Run("fresher open jobs are claimed first", func(t *testing.T) {
		truncate(t, pool)
		// Insert the older-posted job first so its outbox id is lower than the newer
		// job's — proving the claim orders by posted_at, not insertion id.
		older := insertJob(t, pool, "older")
		newer := insertJob(t, pool, "newer")
		setPostedAt(t, pool, older, "2024-01-01T00:00:00Z")
		setPostedAt(t, pool, newer, "2024-06-01T00:00:00Z")
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}

		claimed, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(claimed) != 2 {
			t.Fatalf("claim: rows=%d err=%v, want 2", len(claimed), err)
		}
		if claimed[0].JobID != newer || claimed[1].JobID != older {
			t.Errorf("claim order = [%d, %d], want [%d, %d] (newer posted_at first)",
				claimed[0].JobID, claimed[1].JobID, newer, older)
		}
	})

	t.Run("undated jobs rank by created_at, not last", func(t *testing.T) {
		truncate(t, pool)
		// An old dated job vs a job with no posted_at but freshly ingested. Under
		// NULLS LAST the undated one would sort last; COALESCE(posted_at, created_at)
		// ranks it by its (recent) created_at, so it is claimed first.
		dated := insertJob(t, pool, "dated")
		setPostedAt(t, pool, dated, "2024-01-01T00:00:00Z")
		undated := insertJob(t, pool, "undated") // posted_at NULL, created_at = now()
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}

		claimed, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil || len(claimed) != 2 {
			t.Fatalf("claim: rows=%d err=%v, want 2", len(claimed), err)
		}
		if claimed[0].JobID != undated || claimed[1].JobID != dated {
			t.Errorf("claim order = [%d, %d], want [%d, %d] (undated-but-recent first)",
				claimed[0].JobID, claimed[1].JobID, undated, dated)
		}
	})

	t.Run("closed jobs are not enqueued", func(t *testing.T) {
		truncate(t, pool)
		open := insertJob(t, pool, "open")
		gone := insertJob(t, pool, "closed")
		closeJob(t, pool, gone)
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}

		var jobIDs []int64
		rows, err := pool.Query(ctx, "SELECT job_id FROM enrichment_outbox ORDER BY job_id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			jobIDs = append(jobIDs, id)
		}
		if len(jobIDs) != 1 || jobIDs[0] != open {
			t.Errorf("enqueued job_ids = %v, want only the open job %d", jobIDs, open)
		}
	})

	t.Run("entries for closed jobs are not claimed", func(t *testing.T) {
		truncate(t, pool)
		open := insertJob(t, pool, "open")
		gone := insertJob(t, pool, "gone")
		if _, err := q.EnqueuePendingJobs(ctx, targetVersion); err != nil {
			t.Fatal(err)
		}
		// Close one job after it was queued: the claim-time filter must skip it.
		closeJob(t, pool, gone)

		claimed, err := q.ClaimEnrichmentBatch(ctx, ClaimEnrichmentBatchParams{LeaseSeconds: 3600, BatchSize: 10})
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if len(claimed) != 1 || claimed[0].JobID != open {
			t.Errorf("claimed = %+v, want only the open job %d", claimed, open)
		}
	})
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
