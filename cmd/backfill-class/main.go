// Command backfill-class populates the title-derived classification columns
// (seniority, category) on existing jobs after the classification feature ships.
// Ingest fills these on every crawl, but rows that predate the change — and closed
// jobs that never re-crawl — keep the empty defaults until this one-off worker
// parses their stored title and writes the result. It pages the whole table and
// exits. Idempotent: classification is a pure function of the title, so a second
// run rewrites nothing.
package main

import (
	"context"
	"log"

	"github.com/strelov1/freehire/internal/classify"
	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/db"
)

// backfillBatchSize bounds how many jobs are read per keyset page.
const backfillBatchSize = 500

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	scanned, updated, err := backfillAll(ctx, db.New(pool))
	if err != nil {
		log.Fatalf("backfill-class: %v", err)
	}
	log.Printf("backfill-class done: scanned=%d updated=%d", scanned, updated)
}

// backfillAll parses every job's stored title and rewrites the rows whose
// classification columns differ. It pages by keyset (id > last seen) so concurrent
// writes cannot skip or repeat rows.
func backfillAll(ctx context.Context, q *db.Queries) (scanned, updated int, err error) {
	var afterID int64
	for {
		jobs, err := q.ListJobsByIDAfter(ctx, db.ListJobsByIDAfterParams{
			AfterID:   afterID,
			BatchSize: backfillBatchSize,
		})
		if err != nil {
			return scanned, updated, err
		}
		if len(jobs) == 0 {
			break
		}
		afterID = jobs[len(jobs)-1].ID

		for _, j := range jobs {
			scanned++
			class := classify.Parse(j.Title)
			if class.Seniority == j.Seniority && class.Category == j.Category {
				continue
			}
			if err := q.SetJobClassification(ctx, db.SetJobClassificationParams{
				ID:        j.ID,
				Seniority: class.Seniority,
				Category:  class.Category,
			}); err != nil {
				return scanned, updated, err
			}
			updated++
		}

		if len(jobs) < backfillBatchSize {
			break
		}
	}

	return scanned, updated, nil
}
