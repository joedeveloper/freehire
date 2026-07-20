## 1. Spike — feasibility & threshold calibration

- [ ] 1.1 On a sample of known clusters (Amazon SDE, JPMorgan LSE, NVIDIA, and the Towa Kraków/Wien pair), compute pairwise cosine over `jobs.semantic_embedding` within `(company_slug, stripped-title)` buckets; pick a threshold that merges true same-role city variants without merging distinct roles/grades. End with a VALIDATED/PARTIAL/INVALIDATED verdict.
- [ ] 1.2 Confirm embedding coverage over the residual buckets (what fraction of the 265,534 clusters have embeddings on ≥2 members).

## 2. Query layer

- [ ] 2.1 Add SQL to stream `(company_slug, normalized-title)` buckets of open canonical rows with their embeddings (only buckets with ≥2 embedded members), reusing the strip helper's normalization.
- [ ] 2.2 Add the `duplicate_of` writer for semantically-merged rows (idempotent `IS DISTINCT FROM` guard), mirroring the exact pass.

## 3. Dedup pass

- [ ] 3.1 Implement the per-bucket clustering (cosine ≥ threshold → same cluster, `min(id)` canon), with the seniority/grade guard, as a pure, unit-tested function.
- [ ] 3.2 Wire it into `cmd/reindex` AFTER `recomputeRoleDuplicates`/`suppressAggregatorDuplicates` (or a dedicated worker), over leftover canons only; skip embeddingless rows.
- [ ] 3.3 Unit-test: same-role/different-desc merges; distinct-role stays split; grade guard; embeddingless left alone; idempotent re-run.

## 4. Verification

- [ ] 4.1 On prod (read-only or a copy), measure how many of the 849,871 collapsible cards the chosen threshold actually merges, and sample merges for false positives.
- [ ] 4.2 Confirm the geo-union (`ingest-content-dedup`) still widens the semantically-merged canons correctly.
- [ ] 4.3 `go build ./... && go vet ./... && go test ./...` green.
