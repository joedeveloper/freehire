## 1. Source contract and HTTP transport

- [ ] 1.1 Define `Source` interface, `CompanyEntry`, and raw `Job` in `internal/sources/source.go`
- [ ] 1.2 Define the `HTTPClient` interface and a `reg`/`All` helper shape (empty registry compiles)
- [ ] 1.3 Implement the real HTTP client in `internal/sources/http.go` (timeout, User-Agent, transient retry-with-backoff) against the `HTTPClient` interface

## 2. Config loader

- [ ] 2.1 Implement `sources.yml` loader → `[]CompanyEntry` in `internal/sources/config.go`
- [ ] 2.2 Validate config against the registry: unknown `provider` returns an error (fail-fast wiring lands in cmd/ingest)
- [ ] 2.3 Add a `sources.yml` with a small set of real boards

## 3. Greenhouse adapter

- [ ] 3.1 Implement `NewGreenhouse` + `Fetch` mapping board JSON (with `?content=true`) to `[]Job`, using a fake `HTTPClient` in tests with a canned fixture
- [ ] 3.2 Register Greenhouse in `sources.All`

## 4. Lever adapter

- [ ] 4.1 Implement `NewLever` + `Fetch` mapping postings JSON (`descriptionPlain`, `createdAt`) to `[]Job` against a canned fixture
- [ ] 4.2 Register Lever in `sources.All`

## 5. Ashby adapter

- [ ] 5.1 Implement `NewAshby` + `Fetch` mapping postings JSON (`descriptionPlain`, `publishedAt`) to `[]Job` against a canned fixture
- [ ] 5.2 Register Ashby in `sources.All`

## 6. Write path: preserve enrichment + gated enqueue

- [ ] 6.1 Edit `queries/jobs.sql`: drop enrichment columns from `UpsertJob`'s `ON CONFLICT DO UPDATE` (insert keeps defaults); verify no production caller breaks
- [ ] 6.2 Add a gated enqueue query inserting into `enrichment_outbox` when `enriched_at IS NULL OR enrichment_version < target`, `ON CONFLICT DO NOTHING`
- [ ] 6.3 `make sqlc` and commit regenerated `internal/db`

## 7. Pipeline

- [ ] 7.1 Define the pipeline `Store` interface (`Save` = upsert + gated enqueue in one tx) and a fake for tests
- [ ] 7.2 Implement the pipeline runner: dispatch by provider, set `source`, namespace `external_id` as `<board>:<native-id>`, derive `company_slug`, bounded concurrency, per-source failure isolation, ingested/failed tally
- [ ] 7.3 Pipeline unit tests: namespacing, `company_slug`, enqueue-only-if-needed, one failing source does not abort the run

## 8. DB-backed Store

- [ ] 8.1 Implement `cmd/ingest/store.go`: `Save` runs `UpsertJob` + gated enqueue in one pgx transaction
- [ ] 8.2 Integration test (testcontainers, `-tags=integration`): upsert+enqueue tx, and re-ingest preserves existing enrichment

## 9. Ingest command

- [ ] 9.1 Implement `cmd/ingest/main.go`: load config, fail-fast on unknown provider, build pool + `sources.All(client)`, run pipeline once, log ingested/failed, exit
- [ ] 9.2 Manual end-to-end run against a real board confirms jobs land and new ones get an outbox entry

## 10. Verification

- [ ] 10.1 `go build ./... && go vet ./... && go test ./...` green; integration test green with Docker
