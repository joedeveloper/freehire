// Command backfill-remote-regions is a run-once host worker that loads the curated
// "remote hiring regions" dataset (sources/remote-companies.csv) into the companies
// table. Each row is matched to a company by its normalized-name slug: an existing
// company has its remote_regions facet set (and the raw source string recorded under
// company_info.remote_regions_raw), and an unmatched row is a no-op — this worker
// annotates existing companies only and never inserts reference rows. The region
// string is resolved to macro-region codes by internal/remoteregion (best-effort;
// an unresolvable label yields an empty set). Idempotent: re-running rewrites the
// same values. It never touches job_count, collections, is_reference, or the
// job-derived facet arrays.
//
//	backfill-remote-regions <path/to/remote-companies.csv>   # needs DATABASE_URL
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"strings"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/normalize"
	"github.com/strelov1/freehire/internal/remoteregion"
	"github.com/strelov1/freehire/internal/worker"
)

// store is the slice of the data layer the loader needs; *db.Queries satisfies it and
// tests use a fake.
type store interface {
	SetCompanyRemoteRegions(ctx context.Context, arg db.SetCompanyRemoteRegionsParams) (int64, error)
}

func main() { worker.Main(run) }

func run() int {
	if len(os.Args) < 2 {
		log.Printf("usage: backfill-remote-regions <path/to/remote-companies.csv>")
		return 2
	}
	path := os.Args[1]

	ctx, _, pool, cleanup, err := worker.Bootstrap(context.Background())
	if err != nil {
		log.Printf("database: %v", err)
		return 1
	}
	defer cleanup()

	f, err := os.Open(path)
	if err != nil {
		log.Printf("open %s: %v", path, err)
		return 1
	}
	defer f.Close()

	stats, err := load(ctx, db.New(pool), f)
	if err != nil {
		log.Printf("backfill-remote-regions: %v", err)
		return 1
	}
	log.Printf("backfill-remote-regions done: matched=%d unmatched=%d mapped=%d unmapped=%d skipped=%d",
		stats.matched, stats.unmatched, stats.mapped, stats.unmapped, stats.skipped)
	return 0
}

// loadStats tallies the run: matched/unmatched count rows by whether their slug hit
// an existing company; mapped/unmapped count rows by whether the region string
// resolved to at least one macro region; skipped counts rows with no usable name.
type loadStats struct{ matched, unmatched, mapped, unmapped, skipped int }

// load streams the CSV dataset (header row: Name, Website, Region), resolves each
// row's region string and slug, and updates the matched company. Column order is
// taken from the header, so the file may reorder columns.
func load(ctx context.Context, s store, r io.Reader) (loadStats, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return loadStats{}, err
	}
	nameIdx, regionIdx := columnIndex(header, "name"), columnIndex(header, "region")
	if nameIdx < 0 || regionIdx < 0 {
		return loadStats{}, errors.New("dataset must have Name and Region header columns")
	}

	var stats loadStats
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, err
		}
		name := row[nameIdx]
		slug := normalize.Slug(name)
		if slug == "" {
			stats.skipped++
			continue
		}
		raw := row[regionIdx]
		regions := remoteregion.Map(raw)
		if len(regions) > 0 {
			stats.mapped++
		} else {
			stats.unmapped++
			regions = []string{} // NOT NULL column: send '{}', never NULL
		}
		affected, err := s.SetCompanyRemoteRegions(ctx, db.SetCompanyRemoteRegionsParams{
			Slug:             slug,
			RemoteRegions:    regions,
			RemoteRegionsRaw: raw,
		})
		if err != nil {
			return stats, err
		}
		if affected > 0 {
			stats.matched++
		} else {
			stats.unmatched++
		}
	}
	return stats, nil
}

// columnIndex returns the position of the named column in the header (case-
// insensitive), or -1 if absent.
func columnIndex(header []string, name string) int {
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), name) {
			return i
		}
	}
	return -1
}
