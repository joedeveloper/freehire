// Command enrich is the standalone enrichment worker. It enqueues jobs that need
// enriching, then drains the outbox queue: for each claimed job it asks the LLM for
// a structured Enrichment, validates it, and writes it back. Run it on a schedule
// (e.g. cron); it processes a bounded batch and exits.
package main

import (
	"context"
	"log"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/enrich"
)

func main() {
	cfg := config.Load()
	ecfg, err := config.LoadEnrich()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	provider, err := enrich.NewLangChainProvider(ecfg.LLMBaseURL, ecfg.LLMAPIKey, ecfg.LLMModel)
	if err != nil {
		log.Fatalf("provider: %v", err)
	}

	runner := enrich.Runner{Provider: provider, Store: newDBStore(pool)}

	stats, err := runner.Run(ctx, enrich.RunOptions{
		TargetVersion: enrich.Version,
		BatchSize:     ecfg.BatchSize,
		LeaseSeconds:  ecfg.LeaseSeconds,
		MaxAttempts:   ecfg.MaxAttempts,
	})
	if err != nil {
		log.Fatalf("enrich: %v", err)
	}

	log.Printf("enrichment done: enriched=%d failed=%d dead_lettered=%d",
		stats.Enriched, stats.Failed, stats.DeadLettered)
}
