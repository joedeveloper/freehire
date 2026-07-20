## Spike verdict (2026-07-20): ❌ INVALIDATED — do not implement as designed

A read-only spike over prod embeddings (`jobs.semantic_embedding`, real[768]; cosine via
`unnest(a,b)`) killed the embedding-cosine-within-bucket approach:

- **Mixed-specialty buckets** — speechify `software engineer` bucket (the strip ate the
  `, Data Infrastructure`/`, Platform`/`, iOS` SPECIALTY suffix, not a city): cosine ranges
  overlap fully — DataInfra avg 0.968 (min 0.87), Platform avg 0.966 (max **0.9919**), iOS
  avg 0.963. No threshold separates same-specialty from different-specialty → over-merge.
- **Generic-title buckets** — amazon `software development engineer` (213 rows): cosine to a
  base ranges **0.83–0.97**. These are genuinely DIFFERENT jobs (AWS/retail/Alexa/… teams)
  sharing one generic title, NOT dupes. Collapsing them hides real openings; cosine correctly
  says "different".
- **No universal threshold:** true dupes (Towa Kraków/Wien = 0.978) sit BELOW a distinct role
  (speechify Platform = 0.992), so one cosine cut cannot separate dupes from distinct roles.

**Consequence:** the "≈849,871 collapsible cards" figure is misleading — most of that residual
is genuinely distinct jobs under a generic/over-stripped title, not redundant reposts. An
embedding-cosine pass on `(company_slug, stripped-title)` buckets would over-merge distinct
roles — exactly the failure this project set out to avoid. **Not deploying.**

**If revisited, change the signal, not the threshold:** the real dupes (Towa Kraków/Wien)
differ by only ~120–200 chars out of ~11,000 (>98% identical description text), while distinct
roles differ substantially. A normalized-description near-duplicate measure (shingling /
length-ratio / edit-distance on the cleaned description) separates those two far better than the
boilerplate-dominated embedding. That is a different proposal; this one is shelved.

---

## Context

`ingest-content-dedup` collapses role clusters keyed on `company_slug + stripped-title
+ full-description`. The description is an EXACT over-merge guard, so same-role postings
with per-city localized descriptions do not collapse. Measured residual: 265,534
`(company_slug, stripped-title)` buckets hold >1 canon (1,115,405 cards → 265,534;
~849,871 collapsible; tech: 34,133 / 116,025 / ~81,892). The catalogue already stores
`jobs.semantic_embedding` (`semantic-embedding`), so same-role postings are near in
vector space regardless of localized text.

## Goals / Non-Goals

**Goals:**
- Collapse same-role postings that exact-description matching misses, using embeddings.
- Zero regression to the deterministic exact pass; strictly additive collapse.
- Bounded cost (no global O(n²)); reversible; measurable before enabling.

**Non-Goals:**
- Cross-company or cross-role merging.
- Replacing the exact pass (it stays the primary, deterministic collapse).
- Re-embedding — reuse stored vectors only; embeddingless rows fall back to exact.
- The `company_slug` duplication issue (`jp-morgan-chase` vs `jpmorganchase`) — separate.

## Decisions

### D1 — Bucket by (company_slug, stripped-title), cluster by cosine within the bucket
The stripped-title normalization from `ingest-content-dedup` already groups per-city
variants; within that small bucket, cosine over embeddings decides same-role. Bucketing
bounds comparisons (a few dozen per bucket, not millions) AND is itself an over-merge
guard — distinct roles live in different buckets and never compare. Chosen over global
nearest-neighbour (unbounded, cross-role risk) and over pgvector ANN (adds an index +
extension for a batch job that only needs within-bucket pairwise).

### D2 — Runs after the exact pass, over its leftover canons only
The semantic pass reads only rows the exact pass left canonical (`duplicate_of IS NULL`)
and only adds `duplicate_of` markers. It cannot re-split or contradict a deterministic
collapse, so the two compose cleanly. Reuses the exact pass's column, reindex exclusion,
`/copies`, and geo-union unchanged.

### D3 — Conservative threshold + grade guard, calibrated by spike first
Threshold is calibrated on labelled clusters (spike, task 1.1) — high enough that
distinct roles in one bucket stay split. A seniority/grade guard prevents merging
senior/staff/etc. of one title (grades are legitimately distinct roles, as in the exact
pass's prefix rule). Ship measured, not guessed.

### D4 — Embeddingless rows fall back to exact
A posting without a stored vector is never semantically merged — coverage gaps reduce
recall, never correctness. Task 1.2 quantifies coverage before build.

## Risks / Trade-offs

- **Over-merge of near-but-distinct roles** → Mitigation: bucket + conservative threshold
  + grade guard; measure false-positive rate on a prod sample before enabling.
- **Cost of pairwise cosine** → Mitigation: bounded per bucket; only buckets with ≥2
  embedded members; runs in the existing reindex cadence.
- **Threshold drift across companies** → Mitigation: single conservative global threshold
  first; per-company tuning only if the sample demands it.

## Open Questions

- Home: extend `cmd/reindex` vs a dedicated worker (embedding reads are heavy; a separate
  cadence may be cleaner) — resolve after the spike sizes the cost.
- Exact cosine threshold and grade-guard token list — output of the spike.
