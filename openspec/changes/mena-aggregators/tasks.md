## 1. Shared Chrome-fingerprint transport

- [x] 1.1 Extract `internal/sources/metahttp.go` into a provider-neutral
  `internal/sources/fingerprinthttp.go` (`fingerprintHTTP` with `get`/`GetHTML`/`GetXML`
  and the `safehttp` SSRF-guarded dialer). Re-point the Meta adapter at it; move the SSRF
  test (`TestMetaHTTPBlocksInternalTarget` → `TestFingerprintHTTPBlocksInternalTarget`) and
  keep it green.
- [x] 1.2 In `All()`, build the fingerprint transport **once** and share it across
  `meta`, `bayt`, `gulftalent` (nil-client marker/listing path stays transport-free).
  `go build ./...` + `go test ./internal/sources/` pass.

## 2. Bayt adapter (`internal/sources/bayt.go`)

- [x] 2.1 Build faithful inline HTML fixtures (`baytDetailHTML`/`baytListingHTML`) mirroring
  the real Bayt JSON-LD structure captured from a live page — matching the repo's JSON-LD
  adapter pattern (metacareers/globalpayments use inline builders, not `testdata/`), which is
  deterministic and network-free.
- [x] 2.2 JSON-LD detail parse: failing test that a detail fixture yields a `Job` with
  title, HTML description, company (`hiringOrganization.name`), free-text location
  (`jobLocation.address` locality + ISO country), posted-at (`datePosted`), and `ExternalID`
  (numeric id from the URL). A detail with no `JobPosting`/no company is dropped.
- [x] 2.3 Listing pagination: failing test that a listing fixture yields all job-detail
  hrefs and that pagination stops on a page with no new links; implemented the walker.
- [x] 2.4 `Fetch` wiring + marker: `Provider() == "bayt"`, board-based (board = country slug),
  `aggregator()` marker; company from the posting (drop company-less and id-less postings);
  bounded detail fan-out (`baytDetailWorkers = 3`, conservative for Bayt's throttling edge).
  Re-crawl dedup identity is the URL id.
- [x] 2.5 Register `NewBayt(fp)` in `All()` over the shared fingerprint transport; add
  `sources/bayt.yml` with the 8 live-validated country scopes (sa/ae/qa/kw/bh/om/eg/jo).
  Config validation accepts the board file (verified via `cmd/ingest`).

## 3. GulfTalent adapter (`internal/sources/gulftalent.go`)

- [x] 3.1 Build faithful inline fixtures (`gtSitemapIndexXML`/`gtShardXML`/`gtDetailHTML`)
  mirroring the real sitemap index, a `jx` shard, and a detail JobPosting captured live.
- [x] 3.2 Sitemap enumeration: failing test that the index + shard fixtures yield the
  job-detail URLs, following ONLY the `jx` job-posting shards (jl/jc/co skipped); implemented
  the enumerator. Live capture corrected the shard marker from `jl` to `jx`.
- [x] 3.3 Detail JSON-LD parse: failing test that a detail fixture yields a `Job`
  (title/description/company/location/posted-at RFC3339/`ExternalID` from the URL id);
  implemented. A detail with no `JobPosting`/no company is dropped; an unparseable index errors.
- [x] 3.4 `Fetch` wiring + markers (`Provider() == "gulftalent"`, `boardless()`,
  `aggregator()`), company from the posting, drop company-less/id-less, bounded fan-out
  (`gulftalentDetailWorkers = 4`). Dedup identity is the URL id.
- [x] 3.5 Register `NewGulfTalent(fp)` in `All()` over the shared fingerprint transport; add
  `sources/gulftalent.yml`. Config validation accepts the board file (verified via `cmd/ingest`).

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
