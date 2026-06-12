-- name: ListCompanies :many
-- Catalog page: companies with their job counts. The job count is computed on
-- the fly (no denormalized counter yet). This is the one acknowledged place a
-- join to jobs is acceptable; LEFT JOIN keeps companies with zero jobs visible.
-- An empty `search` short-circuits the ILIKE, so the same prepared statement
-- serves both the full list and a name search (`search` is a case-insensitive
-- substring of the name).
SELECT c.slug, c.name, count(j.company_slug) AS job_count
FROM companies c
LEFT JOIN jobs j ON j.company_slug = c.slug
WHERE sqlc.arg('search')::text = '' OR c.name ILIKE '%' || sqlc.arg('search') || '%'
GROUP BY c.slug, c.name
ORDER BY c.name
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCompanies :one
-- Total companies matching the same optional name filter as ListCompanies, so
-- search pagination reports the filtered total.
SELECT count(*)
FROM companies
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%';

-- name: GetCompany :one
SELECT slug, name, created_at, updated_at
FROM companies
WHERE slug = $1;
