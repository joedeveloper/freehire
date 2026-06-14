// Command backfill-skills populates the deterministic skills column on existing
// jobs after the skill-tagging feature ships. Ingest fills it on every crawl, but
// rows that predate the change — and orphan jobs that never re-crawl — keep the
// empty default until this one-off worker parses their stored description and
// writes the result. It pages the whole table and exits. Idempotent: skills are a
// pure function of the description, so a second run rewrites nothing.
package main

import (
	"context"
	"log"
	"slices"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/skilltag"
)

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
		log.Fatalf("backfill-skills: %v", err)
	}
	log.Printf("backfill-skills done: scanned=%d updated=%d", scanned, updated)
}

// backfillAll parses every job's stored description and rewrites the rows whose
// skills column differs. It pages by keyset (id > last seen) so concurrent writes
// cannot skip or repeat rows.
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
			skills := skilltag.Parse(j.Description)
			if slices.Equal(skills, j.Skills) {
				continue
			}
			if err := q.SetJobSkills(ctx, db.SetJobSkillsParams{
				ID:     j.ID,
				Skills: skills,
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
