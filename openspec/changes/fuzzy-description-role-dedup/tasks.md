## 1. Sizing spike (feasibility of the impl, not the signal — signal already VALIDATED)

- [ ] 1.1 Measure the bucket-size distribution over `(company_slug, stripped-title)` buckets with >1 open canon (max/median/tail) to decide the similarity impl (naïve pairwise is fine for small buckets; big buckets need MinHash/LSH or `pg_trgm`).
- [ ] 1.2 Pick and validate the threshold on a labelled prod sample (Towa, speechify, amazon + a random draw): confirm recall on true dupes and near-zero false positives at the chosen T (≥0.9 candidate).
- [ ] 1.3 Count the REAL collapse potential (within-bucket pairs/clusters ≥ T) — the honest figure, not the 849k stripped-title count.

## 2. Similarity + query layer

- [ ] 2.1 Implement the normalized-description word-signature (distinct lowercase tokens len>2) + the within-bucket similarity (chosen impl), as a pure, unit-tested function.
- [ ] 2.2 Add SQL to stream `(company_slug, stripped-title)` buckets of exact-pass leftover canons with their description signatures.
- [ ] 2.3 Reuse/extend the `duplicate_of` writer (idempotent `IS DISTINCT FROM`), mirroring the exact pass.

## 3. Dedup pass

- [ ] 3.1 Per-bucket clustering (similarity ≥ T → same cluster, `min(id)` canon) with the seniority-grade guard, unit-tested.
- [ ] 3.2 Wire into `cmd/reindex` AFTER `recomputeRoleDuplicates`/`suppressAggregatorDuplicates` (or a dedicated worker), over leftover canons only.
- [ ] 3.3 Unit tests: near-identical merges; distinct-job (amazon-style) stays split; mixed-specialty (speechify-style) stays split; grade guard; idempotent re-run.

## 4. Verification

- [ ] 4.1 On a prod copy (or read-only dry-run that logs would-merge sets), sample merges for false positives at the chosen T.
- [ ] 4.2 Confirm the geo-union (`ingest-content-dedup`) still widens the fuzzy-merged canons; confirm Towa Kraków/Wien now fold into the fullstack canon.
- [ ] 4.3 `go build ./... && go vet ./... && go test ./...` green.
