## Spike verdict (2026-07-20): ✅ VALIDATED

Read-only prod spike, word-Jaccard of the normalized description (distinct lowercase tokens
of length >2, `|A∩B|/|A∪B|`):

| Bucket | same-role city variants | distinct roles |
|---|---|---|
| Towa `senior fullstack engineer` | 0.976–1.000 | Data Specialist 0.49, Management Consultant 0.48 |
| speechify `software engineer` | DataInfra 0.954 | Platform 0.44, iOS 0.46 |
| amazon `software development engineer` | (1 near-dupe) | 261 rows, avg 0.186 |

Wide, consistent gap (true dupes ≥0.95, distinct ≤0.5) — and it works EXACTLY where the
embedding approach failed (speechify 0.968 vs 0.966; amazon 0.83–0.97). Amazon's generic-title
bucket correctly does not collapse (avg 0.186 = genuinely distinct jobs). Any threshold in
0.6–0.9 separates cleanly; ≥0.9 is conservative and safe.

## Context

`ingest-content-dedup` collapses on EXACT description match. Near-identical-but-localized
descriptions (Towa Kraków/Wien, >98% identical) stay split. The prior embedding follow-up was
INVALIDATED (`semantic-role-dedup`, shelved). Word-Jaccard of the description is the validated
signal.

## Goals / Non-Goals

**Goals:**
- Collapse near-identical-description reposts that exact matching misses.
- Zero regression to the deterministic exact pass; strictly additive.
- Bounded cost; reversible; measurable before enabling.

**Non-Goals:**
- Merging genuinely distinct jobs under a generic title (amazon SDE) — the threshold must
  leave those split.
- Cross-company/cross-bucket merging.
- The `company_slug` duplication issue (`jp-morgan-chase`/`jpmorganchase`) — separate.

## Decisions

### D1 — Signal: word-Jaccard of the normalized description (not embeddings)
Spike-proven to separate dupes from distinct roles where embeddings could not (boilerplate
dominates embeddings; word-overlap captures the specialty-specific body). This is the exact-md5
guard relaxed to ≥T word overlap.

### D2 — Bucket by (company_slug, stripped-title); cluster within
Same bucketing as the exact pass bounds comparison and prevents cross-role merges. Within the
bucket, cluster by Jaccard ≥ threshold, `min(id)` canon.

### D3 — After the exact pass, additive only
Reads only exact-pass leftover canons (`duplicate_of IS NULL`), only adds markers. Reuses the
column, reindex exclusion, `/copies`, geo-union unchanged.

### D4 — Efficient signature, not naïve O(n²)
Pairwise Jaccard per bucket is O(n²); large buckets (amazon 261, speechify 305) need a
signature: MinHash/LSH banding, a shingle simhash, or a `pg_trgm` GIN with `similarity()`.
Pick during implementation after the spike sizes bucket distribution. Conservative threshold
≥0.9 (well inside the ≥0.95 true-dupe band, far above ≤0.5 distinct).

### D5 — Grade guard
Reuse the seniority-grade guard so senior/staff/etc. of one title are not merged even at high
description similarity.

## Risks / Trade-offs

- **Over-merge of near-but-distinct roles** → bucket + conservative threshold + grade guard;
  measure false-positive rate on a prod sample before enabling. Spike margin is large.
- **O(n²) cost on big buckets** → signature/LSH; only buckets with >1 canon; existing reindex
  cadence.
- **Threshold drift** → single conservative global threshold first; the spike shows the gap is
  company-independent so far.

## Open Questions

- Similarity impl: MinHash/LSH vs `pg_trgm` vs precomputed shingle signature — sized by the
  implementation spike (bucket-size distribution).
- Exact threshold (0.85 vs 0.9 vs 0.95) — tune on a labelled prod sample; measure recall/FP.
- How many cards this actually collapses — a real count (within-bucket ≥T pairs), NOT the
  misleading 849k stripped-title figure (which counted distinct jobs too).
