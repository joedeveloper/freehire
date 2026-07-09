## 1. Schema

- [ ] 1.1 Migration `migrations/0006_company_yc_facets.sql` adding
      `companies.yc_batch text[]` + `yc_status text[]` (NOT NULL DEFAULT '{}').
- [ ] 1.2 Regenerate `internal/db` via `make sqlc`; confirm `db.Company` gains
      `YcBatch`/`YcStatus` and the build passes.

## 2. yc-oss mapping (`internal/ycdir`)

- [ ] 2.1 RED: table tests for the entry struct + `Map(entry) → record` — tagline
      from one_liner, industries = industry+tags deduped, employee_count/team_size,
      year from launched_at, hq_country via location.Parse(all_locations), batch/
      status passthrough, JSONB extras, blank-name skip, absent-optional handling.
- [ ] 2.2 GREEN: implement the struct + pure `Map`; reuse `internal/location`.
- [ ] 2.3 REFACTOR + simplify; tests green.

## 3. Persistence (`UpsertYCCompany`) + recompute guard

- [ ] 3.1 Add `UpsertYCCompany` to `companies.sql`: INSERT reference row on new
      slug / UPDATE company-info columns + yc_batch + yc_status on existing; never
      touch job_count/collections/job-derived facets. Regenerate via `make sqlc`.
- [ ] 3.2 Integration test (`//go:build integration`): existing company enriched
      (facets + company-info set, job_count/collections untouched); unmatched slug
      inserted as reference row; idempotent re-run.
- [ ] 3.3 Integration test: `RefreshCompanyFacets` leaves `yc_batch`/`yc_status`
      untouched.

## 4. Importer (`cmd/import-yc`)

- [ ] 4.1 RED: loader unit test against a fake store + a fake fetcher — maps entries,
      upserts, tallies matched/inserted, skips blank names.
- [ ] 4.2 GREEN: implement `cmd/import-yc` (worker.Main/Bootstrap, fetch yc-oss with
      a timeout mirroring `import-collections`, `ycdir.Map`, `UpsertYCCompany`).
- [ ] 4.3 REFACTOR + simplify; tests green.

## 5. Company list facets (API)

- [ ] 5.1 Add `yc_batch`/`yc_status` facet params to `ListCompanies`/`CountCompanies`
      (`&&` overlap + empty short-circuit; keep both WHEREs identical); regenerate.
- [ ] 5.2 Wire both through the companies list handler (repeatable params).
- [ ] 5.3 Handler test: `?yc_status=Active&yc_batch=...` filters and reports total.

## 6. Web filter UI

- [ ] 6.1 Add `yc_status` (pills) + `yc_batch` (searchable select) facets to
      `COMPANY_FACETS` (`web/src/lib/facets.ts`); status options from the controlled
      set, batch options from a curated/served list. Model round-trip test.
- [ ] 6.2 Verify (svelte-check + vitest + visual) the facets filter the list.

## 7. Docs

- [ ] 7.1 AGENT.md: `cmd/import-yc`, `internal/ycdir`, the yc_batch/yc_status curated
      facets, and the yc-oss enrichment convention.

## 8. Finish + deploy

- [ ] 8.1 `go build ./... && go vet ./... && go test ./...` + `internal/db`/handler
      integration green.
- [ ] 8.2 Apply migration on prod, deploy (`release.sh`), build+run `cmd/import-yc`
      on host, verify the facets live and reference rows loaded.
