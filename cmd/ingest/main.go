// Command ingest is the standalone source-ingest worker. It loads the configured
// boards from sources.yml, fetches each through its platform adapter, normalizes the
// postings, and upserts them — enqueuing new ones for enrichment in the same write.
// Run it on a schedule (e.g. cron); it processes every board once and exits.
package main

import (
	"context"
	"log"
	"os"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
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
}
