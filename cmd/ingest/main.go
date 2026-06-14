// Command ingest is the standalone source-ingest worker. It loads one board file
// (sources/<provider>.yml, or a mixed sources/custom.yml — passed as the first argument
// or via SOURCES_FILE), fetches each board through its adapter, normalizes the postings,
// and upserts them — enqueuing new ones for enrichment in the same write. A file's
// entries usually share the file-name provider, but an entry may name its own, so one
// file can cover several single-source providers. After the run each provider that
// ingested at least one job has its stale jobs swept. Run on a schedule (e.g. cron); it
// processes its boards once and exits.
package main

import (
	"context"
	"log"
	"os"
	"sort"
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

	// The board file is usually one provider's list (sources/<provider>.yml, provider =
	// file name), but an entry may name its own provider, so it may be a mixed file
	// (sources/custom.yml). Accept it as the first argument (cron passes the file) or via
	// SOURCES_FILE.
	path := os.Getenv("SOURCES_FILE")
	if len(os.Args) > 1 && os.Args[1] != "" {
		path = os.Args[1]
	}
	if path == "" {
		log.Fatal("config: no board file given (pass sources/<provider>.yml as an argument or set SOURCES_FILE)")
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

	runStats, err := runner.Run(ctx, sourceCfg.Sources)
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}

	total := runStats.Total()
	log.Printf("ingest done: file=%s providers=%d ingested=%d failed=%d skipped=%d",
		path, len(runStats), total.Ingested, total.Failed, total.Skipped)

	// Post-run sweep (job-lifecycle spec): per provider, close that provider's open jobs
	// unseen for the whole grace window. Scoped per provider so one provider's run never
	// closes another's jobs, and guarded per provider (only those that ingested at least
	// one job) so a total crawl outage for one provider cannot mass-close its catalogue —
	// even when several providers share one run (e.g. custom.yml).
	queries := db.New(pool)
	cutoff := pgtype.Timestamptz{Time: time.Now().Add(-staleAfter), Valid: true}
	for _, provider := range sweepableProviders(runStats) {
		closed, err := queries.CloseUnseenJobs(ctx, db.CloseUnseenJobsParams{
			Source: provider,
			Cutoff: cutoff,
		})
		if err != nil {
			log.Fatalf("close stale jobs: %v", err)
		}
		log.Printf("closed %d stale %s jobs (unseen for %s)", closed, provider, staleAfter)
	}
}

// sweepableProviders returns, sorted, the providers in a run that ingested at least one
// job — the only ones safe to sweep (a zero-ingest provider proves only that its crawl
// failed). Sorting gives a deterministic sweep order across runs and tests.
func sweepableProviders(rs pipeline.RunStats) []string {
	var providers []string
	for provider, s := range rs {
		if shouldSweep(s) {
			providers = append(providers, provider)
		}
	}
	sort.Strings(providers)
	return providers
}

// staleAfter is the grace window before an unseen job is closed: many crawl cycles
// at the hourly per-provider cadence, so a board failing several runs in a row keeps
// its jobs open.
const staleAfter = 48 * time.Hour

// shouldSweep reports whether the run saw enough of the world to justify closing
// jobs: a run that ingested nothing proves only that the crawl failed.
func shouldSweep(stats pipeline.Stats) bool {
	return stats.Ingested > 0
}
