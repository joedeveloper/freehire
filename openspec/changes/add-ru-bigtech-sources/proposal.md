## Why

The largest Russian IT employers (Yandex, Ozon, Wildberries, Sber, VK, T-Bank, MTS,
Alfa-Bank, Lamoda, …) do not publish on the multi-tenant ATS platforms we already crawl;
each runs its own bespoke career backend. Their postings — thousands of live IT roles —
are reachable through public, unauthenticated JSON APIs (live-verified with our ingest
client). Adding adapters for them brings the RU-domestic bigtech segment into the shared
catalogue, materially expanding coverage with no new infrastructure beyond two small,
reusable seams.

## What Changes

- Add **15 single-company source adapters** in `internal/sources`, each speaking the
  existing `Source` interface and registered with one line in `sources.All`: `yandex`,
  `ozon`, `rwb` (Wildberries), `sber`, `alfabank`, `lamoda`, `kuper`, `aviasales`,
  `dodo`, `domclick`, `mtslink`, `tbank`, `mts`, `tochka`, `vk`. Most follow the
  established **list → detail** fan-out via the shared `fetchDetails` helper.
- **`board` becomes optional for single-company adapters.** A new in-package `boardless`
  marker interface lets `Config.Validate` skip the empty-`board` check for providers that
  have no tenant/board concept. Yandex is the one exception and keeps `board` required
  (its `board` ∈ {`ru`, `com`} selects host + language). **No behavior change** for the
  12 existing multi-tenant adapters.
- **`HTTPClient` gains per-request header variants** (`GetJSONWithHeaders` /
  `PostJSONWithHeaders`) so an adapter can send a custom header. `MTS` needs
  `x-api-key` (a public JWT served to every SPA visitor); `Tochka`'s detail needs
  `RSC: 1`. The existing `GetJSON`/`PostJSON` delegate to the new variants with `nil`
  headers, so no existing call site changes.
- **VK description extraction** parses the vacancy page HTML via `golang.org/x/net/html`
  (already a direct dependency — no new dependency), because VK exposes no description in
  JSON. Extraction is isolated to `vk.go`.
- Add one per-provider config file under `sources/` for each new provider (Yandex gets
  two entries, `ru` and `com`; the rest one entry each, no `board`).

## Capabilities

### New Capabilities
<!-- None. Reuses the source-ingest pipeline and write path unchanged. -->

### Modified Capabilities
- `source-ingest`: register 15 new single-company providers; relax the board requirement
  so a `boardless` provider's entries may omit `board` (Yandex still requires it); allow
  an adapter to send a per-request custom header through the shared client; allow an
  adapter to obtain a posting's description by extracting it from the source's HTML when
  the platform exposes no JSON description (VK).

## Impact

- **New code**: 15 `internal/sources/<provider>.go` + `<provider>_test.go` pairs; 15
  registration lines in `sources.All` (`source.go`); a `boardless` marker interface +
  a two-line `Validate` change (`config.go`); header-aware variants on `HTTPClient` +
  the shared test fake (`http.go`); a small HTML-extraction helper for VK.
- **Config**: 15 new per-provider files under `sources/`. New optional env seam for the
  MTS key is avoided — the public key is harvested at runtime (board-isolated on
  failure).
- **DB**: none — every adapter reuses `UpsertJob` (`source` = the provider key,
  namespaced `external_id`). No migration.
- **Dependencies**: none new (`golang.org/x/net/html` is already direct).
- **Out of scope / known seams**: region/country normalization at ingest (tracked
  separately as `ingest-job-geography`); hh.ru-backed sources (legally off-limits);
  Cian-direct/SberHealth (headless browser); Avito/Kontur/2GIS (HTML/JSON-LD scraping).
  Unconfirmed public vacancy URLs for Kuper/Dodo/T-Bank are synthesized best-effort and
  must be live-confirmed before merge. `Tochka` (RSC flight-payload) and `VK` (page
  markup) are markup-coupled and isolated so a parse failure drops only that posting.
