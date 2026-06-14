## 1. Fold NBSP into isRemote (1.1)

- [ ] 1.1 RED: add a test asserting `isRemote` matches text where the keyword is preceded/joined by a non-breaking space (e.g. "Удалённо" with NBSP), failing before the fold.
- [ ] 1.2 GREEN: normalize NBSP inside `isRemote`; remove `normalizeNBSP` and drop it from all call sites (aviasales, dodo, kuper, mtslink, mts, ozon, tbank). Delete `normalizeNBSP`'s own test.

## 2. firstNonEmpty helper (1.2)

- [ ] 2.1 RED: add a `firstNonEmpty` test (first non-blank wins; all-blank → ""; whitespace-only treated as blank).
- [ ] 2.2 GREEN: implement `firstNonEmpty` in `source.go`; replace the inline `x := A; if x == "" { x = B }` fallbacks in dodo/mts/sber (company) and domclick/lamoda (body) where it reads at least as clearly.

## 3. Shared defaultDetailWorkers (1.3)

- [ ] 3.1 Replace the 20 per-adapter `xDetailWorkers = 8` constants with one package-level `defaultDetailWorkers = 8` (in `source.go` near `fetchDetails`); update each `fetchDetails(..., xDetailWorkers, ...)` call.

## 4. Move HTML helpers to html.go (1.4)

- [ ] 4.1 Move `walk`/`attr`/`textContent`/`itempropHTML` verbatim from `successfactors.go` to a new `internal/sources/html.go` (no signature change); confirm `mts`/`vk`/`successfactors` still compile against them.

## 5. Verify

- [ ] 5.1 `go build ./... && go vet ./... && go test ./...` green; `go test ./internal/sources/...` covers the touched adapters.
