## 1. Location parser (`internal/location`)

- [x] 1.1 Write table tests for `location.Parse` covering real prod strings:
  `Remote - Germany`, `Remote - Europe`, `Remote - UK or Europe`, `Remote`,
  `Remote - Anywhere`, `Burlington, Massachusetts, United States; Remote`,
  `United States`, empty, and an unresolvable token (RED).
- [x] 1.2 Implement `Parse(location string) (countries, regions []string)` with the
  three dictionaries (name→country, name→region, country→region drawn from
  `enrich.RegionValues`), tokenization on `, ; / | " - " " or "`, dedup, and the
  "never guess / global only on explicit anywhere" rules (GREEN).
- [x] 1.3 Seed the dictionaries from the high-frequency prod location strings; add a
  test asserting every emitted region ∈ `RegionValues` and every country is ISO
  alpha-2.
- [x] 1.4 Extend `Parse` to return a `Geo` struct that also carries a `WorkMode`
  hint detected from explicit markers (remote/hybrid/onsite, priority
  hybrid>remote>onsite); guard it against `enrich.WorkModeValues`.

## 2. Schema + DB access

- [x] 2.1 Add migration: `jobs.countries text[] NOT NULL DEFAULT '{}'`,
  `jobs.regions text[] NOT NULL DEFAULT '{}'` (follow existing `migrations/`
  numbering).
- [x] 2.2 Update `UpsertJob` in `internal/db/queries/jobs.sql` to write `countries`
  and `regions` in INSERT and in `ON CONFLICT DO UPDATE SET`.
- [x] 2.3 Add `SetJobLocation` query (set `countries`/`regions` by id) for the
  backfill path.
- [x] 2.4 Run `make sqlc` and commit the regenerated `internal/db`.
- [x] 2.5 Add `jobs.work_mode TEXT NOT NULL DEFAULT ''` to the same migration;
  write it in `UpsertJob` and `SetJobLocation`.

## 3. Ingest write path

- [x] 3.1 Add `Countries`/`Regions`/`WorkMode` to `pipeline.Job`; have
  `normalizeJob` call `location.Parse(j.Location)` (RED test on `normalizeJob`).
- [x] 3.2 Thread the new fields through the `Store.Save` implementation into
  `UpsertJob`; verify a re-ingest refreshes geography.
- [x] 3.3 Add structured `WorkMode` to `sources.Job`; populate it from the ATS
  structured fields (ashby/recruitee/smartrecruiters/workable remote flag →
  "remote"; lever `workplaceType` → mapped). Helpers + adapter/helper tests.
- [x] 3.4 `normalizeJob` work-mode precedence: adapter-structured ?? parser
  (RED test first).

## 4. Read-time merge (`internal/jobview`)

- [x] 4.1 Add top-level `Regions`, `Countries`, `WorkMode` to `jobview.Job`;
  `FromRow` unions `jobs.regions ∪ enrichment.regions` (and countries) deduped,
  sets `WorkMode = enrichment.work_mode ?? jobs.work_mode`, then blanks the
  folded `enrichment.regions/countries/work_mode` in the served copy (RED tests
  for union, dedup, work-mode precedence, and blanking first).

## 5. Search

- [x] 5.1 Update the index settings: filterable attributes use top-level
  `regions`/`countries`/`work_mode`, not the `enrichment.*` dot-paths; remove the
  folded enrichment geography/work_mode facets.
- [x] 5.2 Update the search handler's region/country/work_mode filters to build
  against the top-level path; adjust `internal/search` document/filter tests.

## 6. Enrichment prompt

- [x] 6.1 Update the `regions` block in `internal/enrich/langchain.go`
  `buildSystemPrompt` to describe `regions` as geographic area for any work mode
  (drop "only when remote"); adjust the prompt test if it asserts that wording.

## 7. Web SPA

- [x] 7.1 Map the SPA region/work_mode filter query-params to the top-level
  fields; verify via `svelte-check` + lint (no SPA test runner).

## 8. Backfill

- [x] 8.1 Add run-once `cmd/backfill-geo`: keyset scan via `ListJobsByIDAfter`,
  `location.Parse`, write countries/regions/work_mode via `SetJobLocation`
  (work_mode is parser-only at backfill; re-crawl later refreshes structured);
  idempotent and safe to re-run.

## 9. Verification

- [x] 9.1 `go build ./... && go vet ./... && go test ./...` green.
- [x] 9.2 Manually confirm the migration → deploy → backfill → reindex order is
  documented (design Migration Plan) and the reindex step is noted for ops.
