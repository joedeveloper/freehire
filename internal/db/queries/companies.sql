-- name: ListCompanies :many
-- Catalog page: companies with their job counts. The job count is computed on
-- the fly (no denormalized counter yet). This is the one acknowledged place a
-- join to jobs is acceptable; LEFT JOIN keeps companies with zero jobs visible.
SELECT c.slug, c.name, count(j.company_slug) AS job_count
FROM companies c
LEFT JOIN jobs j ON j.company_slug = c.slug
GROUP BY c.slug, c.name
ORDER BY c.name
LIMIT $1 OFFSET $2;

-- name: CountCompanies :one
SELECT count(*)
FROM companies;

-- name: GetCompany :one
SELECT slug, name, created_at, updated_at
FROM companies
WHERE slug = $1;
