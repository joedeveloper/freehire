-- Companies as a slug-keyed entity, linked from jobs by a denormalized key.
-- Applied automatically by Postgres on first volume init (same as 0001) and
-- also serves as schema source for sqlc.

CREATE TABLE IF NOT EXISTS companies (
    -- Natural key: the normalized company name. No surrogate id.
    slug        TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Denormalized link key on jobs. The display name stays in jobs.company; this
-- holds the normalized slug so "a company's jobs" is a single-table filter
-- (no join). Empty string means the job has no associated company.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS company_slug TEXT NOT NULL DEFAULT '';

-- Composite to match ListJobsByCompany: WHERE company_slug = $1
-- ORDER BY posted_at DESC NULLS LAST, id DESC. Equality on the leading column
-- plus pre-sorted rows means no per-request sort; the leading column also
-- serves the company-list join. Mirrors jobs_posted_at_id_idx from 0001.
CREATE INDEX IF NOT EXISTS jobs_company_slug_idx
    ON jobs (company_slug, posted_at DESC NULLS LAST, id DESC);
