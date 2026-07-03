## 1. Shared Chrome-fingerprint transport

- [ ] 1.1 Extract `internal/sources/metahttp.go` into a provider-neutral
  `internal/sources/fingerprinthttp.go` (`fingerprintHTTP` with `get`/`GetHTML`/`GetXML`
  and the `safehttp` SSRF-guarded dialer). Re-point the Meta adapter at it; move the SSRF
  test (`TestMetaHTTPBlocksInternalTarget` → fingerprint equivalent) and keep it green.
- [ ] 1.2 In `All()`, build the fingerprint transport **once** and share it across
  `meta`, `bayt`, `gulftalent` (nil-client marker/listing path stays transport-free).
  Confirm `go build ./... && go test ./internal/sources/ -run Fingerprint` passes.

## 2. Bayt adapter (`internal/sources/bayt.go`)

- [ ] 2.1 Record fixtures into `internal/sources/testdata/`: a real Bayt country listing
  page and a real Bayt job-detail page (captured via the fingerprint client), for
  offline parse tests.
- [ ] 2.2 JSON-LD detail parse: failing test that a detail fixture yields a `Job` with
  title, HTML description, company (`hiringOrganization.name`), free-text location
  (`jobLocation`), posted-at (`datePosted`), and `ExternalID` (numeric id from
  URL/`identifier`); then implement the parser. A detail with no `JobPosting` errors.
- [ ] 2.3 Listing pagination: failing test that a listing fixture yields all job-detail
  hrefs and that pagination stops on a page with no new links; then implement the walker.
- [ ] 2.4 `Fetch` wiring + markers: implement `Provider() == "bayt"`, `boardless()`,
  `aggregator()`; company from the posting (drop company-less and id-less postings);
  bounded detail fan-out via `fetchDetails`. Test the re-crawl dedup identity
  (same posting → same `ExternalID`).
- [ ] 2.5 Register `NewBayt(fp)` in `All()`; add `sources/bayt.yml` with the country
  scopes (sa/ae/qa/kw/bh/om/eg/jo). Confirm config validation accepts the board file.

## 3. GulfTalent adapter (`internal/sources/gulftalent.go`)

- [ ] 3.1 Record fixtures: `sitemap.xml` index, one `jl0NN` job sitemap shard, and a real
  detail page.
- [ ] 3.2 Sitemap enumeration: failing test that the index + shard fixtures yield all
  job-detail URLs across every `jl` shard; then implement the enumerator.
- [ ] 3.3 Detail JSON-LD parse: failing test that a detail fixture yields a `Job`
  (title/description/company/location/posted-at/`ExternalID` from the URL id); then
  implement. A detail with no `JobPosting` and an unparseable index both error.
- [ ] 3.4 `Fetch` wiring + markers (`Provider() == "gulftalent"`, `boardless()`,
  `aggregator()`), company from the posting, drop company-less/id-less, bounded fan-out;
  test the re-crawl dedup identity.
- [ ] 3.5 Register `NewGulfTalent(fp)` in `All()`; add `sources/gulftalent.yml`. Confirm
  config validation accepts the board file.

## 4. Live validation + mega-employer seed

- [ ] 4.1 Live-validate both adapters: `go run ./cmd/ingest sources/bayt.yml` and
  `... sources/gulftalent.yml` against a scratch DB; confirm real MENA jobs upsert with
  correct company/location and no duplicates on a second run.
- [ ] 4.2 Curate ~40–60 largest ME employers (KSA/PIF, UAE, Qatar, Egypt clusters), detect
  each one's existing-supported ATS, and add one live-validated board entry per company to
  the matching `sources/<provider>.yml`.
- [ ] 4.3 Ops note in the change: after first prod ingest run `cmd/backfill-derive` +
  `make reindex`; re-check the `regions=mena` company facet count rose from the 2,357
  baseline.

## 5. Verification

- [ ] 5.1 `go build ./... && go vet ./... && go test ./...` all green; `gofmt` clean.
- [ ] 5.2 `openspec validate mena-aggregators --strict` passes; run
  `requesting-code-review` on the branch diff and resolve Critical + Important findings.
