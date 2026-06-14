## Context

`internal/sources` holds ~35 adapters over a shared layer (`source.go`, `http.go`,
`sanitize.go`, `jsonld.go`). Four small duplications have accumulated across the
adapters. This change pulls them into the shared layer. No behavior changes — same
input yields the same `Job`.

## Goals / Non-Goals

- **Goal:** single-source the four duplicated pieces; keep adapters thinner.
- **Non-Goal:** any change to crawl behavior, output shape, or the generic
  pagination loop (the offset/skip/page loops vary by envelope and are left as-is —
  a generic helper there would be as long as the duplication).

## Decisions

- **1.1 NBSP inside `isRemote`.** Fold `strings.ReplaceAll(s, " ", " ")` into
  `isRemote` rather than keeping a separate `normalizeNBSP`. Rationale: NBSP-folding
  is only ever used to feed the remote heuristic; making the heuristic robust to NBSP
  is strictly desirable and removes 7 wrapper calls. Alternative (keep both, just
  share `normalizeNBSP`) leaves the redundant call at every site — rejected.
  `normalizeNBSP` is deleted (no other callers).

- **1.2 `firstNonEmpty(parts ...string) string`.** Sibling to `joinNonEmpty` in
  `source.go`; returns the first non-blank trimmed part, else "". Used for the
  `company` fallback (dodo/mts/sber: `firstNonEmpty(it.Brand, e.Company)`) and the
  body fallback where it reads cleanly (domclick/lamoda). Only replace inline blocks
  where the result is at least as readable.

- **1.3 `defaultDetailWorkers = 8`.** All 20 `xDetailWorkers` constants equal 8 —
  it is a shared default, not a per-adapter tuning knob. Collapse to one
  package-level const in `source.go` near `fetchDetails`. If a future adapter needs
  a different bound, it reintroduces its own const at that one site.

- **1.4 Move HTML helpers to `html.go`.** `walk`/`attr`/`textContent`/`itempropHTML`
  are generic DOM helpers currently living in `successfactors.go` but used by `mts`
  and `vk`. Relocate verbatim to a new `internal/sources/html.go` next to `jsonld.go`
  (which already holds DOM-adjacent helpers). Pure move; no signature change.

## Risks / Trade-offs

- [Behavior drift from the NBSP fold] → covered by existing adapter tests; the fold
  only widens what `isRemote` matches (NBSP→space), never narrows it.
- [Over-applying `firstNonEmpty`] → only replace sites where it is at least as clear;
  leave idiomatic two-line fallbacks that don't fit the variadic shape.

## Migration Plan

None — internal refactor, no schema/API/config change. Rollback is a revert.

## Open Questions

None.
