> **STATUS: SHELVED — spike INVALIDATED this approach (2026-07-20).** A prod embedding
> spike showed cosine within a `(company_slug, stripped-title)` bucket does NOT separate
> same-role city variants from distinct roles/specialties, and that most of the measured
> "collapsible" residual is genuinely distinct jobs under a generic title, not dupes. See
> `design.md` → "Spike verdict". Kept as a record; not for implementation as written.

## Why

The role-cluster collapse (`ingest-content-dedup`) folds per-city variants of one
role into a single card, but its identity key includes the **full description** as
the over-merge guard. That guard is exact: two postings collapse only when their
descriptions match byte-for-byte. Many employers — especially big-tech — post the
same role across dozens of cities with **per-location descriptions** (localized
salary/legal/relocation text), so each city keeps a distinct fingerprint and stays
a separate card even though it is unmistakably the same role.

A production measurement quantifies the residual: **265,534** clusters share a
company + a normalized (city-suffix-stripped) title yet did NOT collapse because
their descriptions differ — **1,115,405** open cards that fold to **265,534**
canons, i.e. **~849,871 cards** still shown redundantly (tech subset: 34,133
clusters, 116,025 cards, **~81,892** collapsible). Concrete offenders: one Amazon
"Software Development Engineer" = 228 separate cards; JPMorgan "Lead Software
Engineer" = 201; NVIDIA 111; Google 80; Apple 77.

Exact-description matching cannot reach these. Semantic similarity can: the
catalogue already stores a per-job embedding (`jobs.semantic_embedding`,
`semantic-embedding`), so same-role/different-localized-text postings sit very
close in vector space.

## What Changes

- Add a **semantic role-dedup** pass that, within a `(company_slug,
  normalized-title)` bucket, clusters open canonical postings whose semantic
  embeddings are within a cosine threshold and marks all but one `duplicate_of`
  the chosen canon — the same collapse mechanism, a fuzzier identity.
- It runs AFTER the exact role-cluster recompute (over its leftover canons only),
  so it never fights the deterministic pass; it only merges what exact matching
  left split.
- Bucketing by `(company_slug, stripped-title)` bounds each cosine comparison to a
  small candidate set (never a global O(n²) scan) and prevents cross-role merges.
- Guardrails against over-merge: a conservative threshold, the shared
  stripped-title bucket (so distinct roles never compare), and a seniority/grade
  guard so different grades of one title are not merged.

## Capabilities

### New Capabilities
- `semantic-role-dedup`: cluster and collapse same-role postings that exact
  description matching misses, using stored embeddings within a company+title
  bucket.

### Modified Capabilities
<!-- none: reuses ingest-content-dedup's duplicate_of column and the reindex collapse; introduces no change to those requirements. -->

## Impact

- **Code:** a new pass (likely in `cmd/reindex` after `recomputeRoleDuplicates`,
  or a dedicated worker), new SQL to fetch bucketed canon embeddings and write
  `duplicate_of`, reusing `jobs.semantic_embedding`.
- **Data:** more `duplicate_of` rows; the same reindex/copies/geo-union machinery
  applies unchanged. Reversible (recompute clears it).
- **Risk:** over-merge of near-but-distinct roles — mitigated by the bucket, the
  threshold, and the grade guard; tunable and measurable before enabling.
- **Prereq:** embedding coverage — postings without an embedding fall back to the
  exact pass (no semantic merge), so coverage gaps only reduce recall, never
  correctness.
