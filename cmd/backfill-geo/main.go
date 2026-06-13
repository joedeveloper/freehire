// Command backfill-geo populates the location-derived columns (countries,
// regions, work_mode) on existing jobs after the geography feature ships. Ingest
// fills these on every crawl, but rows that predate the change — and closed jobs
// that never re-crawl — keep the empty defaults until this one-off worker parses
// their stored location and writes the result. It pages the whole table and
// exits. Idempotent: geography is a pure function of the location, so a second
// run rewrites nothing.
//
// work_mode is preserved when already set: a row may carry a structured work_mode
// from the adapter (richer than the free-text parse), so backfill only fills
// work_mode from the parsed location when it is currently empty.
package main

import (
	"context"
	"log"
	"slices"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/location"
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
		log.Fatalf("backfill-geo: %v", err)
	}
	log.Printf("backfill-geo done: scanned=%d updated=%d", scanned, updated)
}

// backfillAll parses every job's stored location and rewrites the rows whose
// location-derived columns differ. It pages by keyset (id > last seen) so
// concurrent writes cannot skip or repeat rows.
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
			geo := location.Parse(j.Location)
			// Preserve an existing (possibly adapter-structured) work_mode; the
			// parsed value only fills an empty one.
			workMode := j.WorkMode
			if workMode == "" {
				workMode = geo.WorkMode
			}
			if slices.Equal(geo.Countries, j.Countries) &&
				slices.Equal(geo.Regions, j.Regions) &&
				workMode == j.WorkMode {
				continue
			}
			if err := q.SetJobLocation(ctx, db.SetJobLocationParams{
				ID:        j.ID,
				Countries: geo.Countries,
				Regions:   geo.Regions,
				WorkMode:  workMode,
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
