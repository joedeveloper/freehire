// Command ingest is the standalone source-ingest worker. It loads the configured
// boards from sources.yml, fetches each through its platform adapter, normalizes the
// postings, and upserts them — enqueuing new ones for enrichment in the same write.
// Run it on a schedule (e.g. cron); it processes every board once and exits.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
	"github.com/strelov1/freehire/internal/pipeline"
	"github.com/strelov1/freehire/internal/sources"
)

func main() {
	cfg := config.Load()

	path := os.Getenv("SOURCES_FILE")
	if path == "" {
		path = "sources.yml"
	}
	sourceCfg, err := sources.LoadConfig(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	registry := sources.All(sources.NewClient())
	// Fail fast before touching the DB: a misconfigured board should not start a run.
	if err := sourceCfg.Validate(registry); err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	runner := pipeline.Runner{
		Registry: registry,
		Store:    newDBStore(pool, enrich.Version),
	}

	stats, err := runner.Run(ctx, sourceCfg.Sources)
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}

	log.Printf("ingest done: ingested=%d failed=%d", stats.Ingested, stats.Failed)

	// Post-run sweep (job-lifecycle spec): close open jobs unseen for the whole
	// grace window. Guarded so a run that ingested nothing (total crawl outage)
	// can never mass-close the catalogue.
	if shouldSweep(stats) {
		cutoff := pgtype.Timestamptz{Time: time.Now().Add(-staleAfter), Valid: true}
		closed, err := db.New(pool).CloseUnseenJobs(ctx, cutoff)
		if err != nil {
			log.Fatalf("close stale jobs: %v", err)
		}
		log.Printf("closed %d stale jobs (unseen for %s)", closed, staleAfter)
	}
}

// staleAfter is the grace window before an unseen job is closed: ~8 crawl cycles
// at the 6h cadence, so a board failing a few runs in a row keeps its jobs open.
const staleAfter = 48 * time.Hour

// shouldSweep reports whether the run saw enough of the world to justify closing
// jobs: a run that ingested nothing proves only that the crawl failed.
func shouldSweep(stats pipeline.Stats) bool {
	return stats.Ingested > 0
}
