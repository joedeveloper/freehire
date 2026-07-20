## ADDED Requirements

### Requirement: Same-role postings are collapsed by semantic similarity within a company+title bucket

The system SHALL collapse open canonical postings that share a `company_slug` and a
normalized (city-suffix-stripped) title AND whose semantic embeddings are within a
configured cosine threshold, marking all but one `duplicate_of` the chosen canon
(the deterministic `min(id)`), reusing the existing collapse column and mechanism.
Comparison SHALL be bucketed by `(company_slug, normalized-title)` so it is bounded
per bucket and never compares postings of different roles. A posting without a
stored embedding SHALL be left to the exact pass (never semantically merged), so
missing embeddings reduce recall but never cause an incorrect merge.

#### Scenario: Same role, per-city localized descriptions, collapses

- **WHEN** a company posts one role in several cities with per-location descriptions
  that differ in text but describe the same role, and their embeddings are within
  the threshold
- **THEN** the postings collapse to one canonical card, the rest referencing it via
  `duplicate_of`

#### Scenario: Distinct roles in the same bucket are not merged

- **WHEN** two postings share a company and stripped title but describe different
  roles (embeddings beyond the threshold)
- **THEN** they remain separate canonical rows

#### Scenario: A posting without an embedding is not semantically merged

- **WHEN** a candidate posting has no stored semantic embedding
- **THEN** the semantic pass leaves it as the exact pass decided (no semantic merge)

### Requirement: The semantic pass runs after and never overrides the exact pass

The semantic role-dedup SHALL run AFTER the exact role-cluster recompute and operate
only over its remaining open canonical rows, so it merges what exact matching left
split and never re-splits or contradicts a deterministic collapse. It SHALL be
idempotent — re-running yields the same markers — and reversible by the standard
recompute.

#### Scenario: Exact-collapsed reposts are untouched

- **WHEN** the exact pass has already collapsed a set of byte-identical-description
  reposts
- **THEN** the semantic pass leaves those `duplicate_of` markers unchanged

#### Scenario: Re-running is stable

- **WHEN** the semantic pass runs twice with no new postings
- **THEN** the second run changes no `duplicate_of` markers

### Requirement: Over-merge guards are enforced

The semantic pass SHALL guard against merging distinct roles: a conservative cosine
threshold, the shared stripped-title bucket, and a seniority/grade guard so postings
that differ only by grade (e.g. senior vs staff) are not merged into one canon.

#### Scenario: Different grades of one title are not merged

- **WHEN** two postings share a company and base title but carry different seniority
  grades
- **THEN** they are not collapsed together, regardless of embedding proximity
