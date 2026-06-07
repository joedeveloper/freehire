-- name: ListJobs :many
SELECT *
FROM jobs
ORDER BY posted_at DESC NULLS LAST, id DESC
LIMIT $1 OFFSET $2;

-- name: GetJob :one
SELECT *
FROM jobs
WHERE id = $1;

-- name: CountJobs :one
SELECT count(*)
FROM jobs;

-- name: ListJobsByCompany :many
SELECT *
FROM jobs
WHERE company_slug = $1
ORDER BY posted_at DESC NULLS LAST, id DESC
LIMIT $2 OFFSET $3;

-- name: UpsertJob :one
-- Single atomic write: upsert the company (only when the slug is non-empty,
-- via the WHERE on the SELECT) and the job together, keeping the "one write =
-- one job" property of the pipeline's write path.
WITH company_upsert AS (
    INSERT INTO companies (slug, name)
    SELECT sqlc.arg(company_slug), sqlc.arg(company)
    WHERE sqlc.arg(company_slug) <> ''
    ON CONFLICT (slug) DO UPDATE SET
        name       = EXCLUDED.name,
        updated_at = now()
)
INSERT INTO jobs (
    source, external_id, url, title, company, company_slug, location, remote, description, posted_at
) VALUES (
    sqlc.arg(source), sqlc.arg(external_id), sqlc.arg(url), sqlc.arg(title),
    sqlc.arg(company), sqlc.arg(company_slug), sqlc.arg(location), sqlc.arg(remote),
    sqlc.arg(description), sqlc.arg(posted_at)
)
ON CONFLICT (source, external_id) DO UPDATE SET
    url          = EXCLUDED.url,
    title        = EXCLUDED.title,
    company      = EXCLUDED.company,
    company_slug = EXCLUDED.company_slug,
    location     = EXCLUDED.location,
    remote       = EXCLUDED.remote,
    description  = EXCLUDED.description,
    posted_at    = EXCLUDED.posted_at,
    updated_at   = now()
RETURNING *;
