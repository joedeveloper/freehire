## 1. Schema migration

- [ ] 1.1 Add `migrations/0003_job_enrichment.sql` adding to `jobs`:
  `enrichment JSONB NOT NULL DEFAULT '{}'`, `enriched_at TIMESTAMPTZ` (nullable),
  `enrichment_version INT NOT NULL DEFAULT 0`.
- [ ] 1.2 Recreate the dev volume (`docker compose down -v && make up`) and
  confirm the three columns exist via `make psql` (`\d jobs`).

## 2. Enrichment contract (`internal/enrich`)

- [ ] 2.1 Create `internal/enrich/enrichment.go` with the `Enrichment` struct:
  one JSON-tagged, `omitempty`/pointer field per the design catalog
  (work arrangement, location/eligibility, compensation, requirements,
  classification, company descriptors). All fields optional.
- [ ] 2.2 Add controlled-vocabulary constants and exported value sets for every
  enum field (`SeniorityValues`, `WorkModeValues`, `EmploymentTypeValues`,
  `RelocationValues`, `EnglishLevelValues`, `EducationLevelValues`,
  `SalaryPeriodValues`, `CategoryValues`, `DomainValues`, `CompanyTypeValues`,
  `CompanySizeValues`).
- [ ] 2.3 Add `Validate()` that checks each enum field against its vocabulary and
  reports the first offending field; non-enum fields are unconstrained.
- [ ] 2.4 Add `internal/enrich/enrichment_test.go`: JSON round-trip fidelity,
  `omitempty` on undetermined fields, and `Validate` accept/reject cases.

## 3. DB access (sqlc)

- [ ] 3.1 Edit `internal/db/queries/jobs.sql`: select `enrichment`,
  `enriched_at`, `enrichment_version` in `GetJob`, `ListJobs`,
  `ListJobsByCompany`; add `enrichment` (and provenance) to `UpsertJob`.
- [ ] 3.2 Confirm `enrichment` is generated as `json.RawMessage` (sqlc.yaml
  override if needed); run `make sqlc` and commit the regenerated
  `internal/db/*.go`.

## 4. API exposure

- [ ] 4.1 Include `enrichment`, `enriched_at`, `enrichment_version` in the job
  objects returned by `GET /api/v1/jobs`, `GET /api/v1/jobs/:id`, and the jobs
  nested under `GET /api/v1/companies/:slug`.
- [ ] 4.2 Ensure an absent payload serializes as `{}` (not `null`) in responses.

## 5. Verification

- [ ] 5.1 `go build ./... && go vet ./...` clean.
- [ ] 5.2 `go test ./internal/enrich/...` passes.
- [ ] 5.3 Insert one enriched job via `make psql`, then `curl` the list, detail,
  and company-nested endpoints to confirm enrichment + provenance are exposed and
  that an un-enriched job shows `enrichment: {}`, `enriched_at: null`,
  `enrichment_version: 0`.
