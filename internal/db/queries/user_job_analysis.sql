-- name: GetUserJobAnalysis :one
-- The caller's cached fit analysis for one job, with the staleness stamps it was
-- computed against. No row means the pair was never analyzed (the handler serves a
-- null analysis, no LLM call). The handler compares cv_uploaded_at / job_content_hash
-- to the live CV upload time and job content_hash to decide the stale flag.
SELECT analysis, model, cv_uploaded_at, job_content_hash, created_at
FROM user_job_analysis
WHERE user_id = $1 AND job_id = $2;

-- name: UpsertUserJobAnalysis :exec
-- Create-or-replace the cached analysis for a (user, job). The composite PRIMARY KEY
-- makes it idempotent: a recompute overwrites the analysis, model, and both staleness
-- stamps. created_at is deliberately NOT re-bumped on conflict, so it records the
-- FIRST-analysis time — the fit-analysis quota counts distinct jobs a user first
-- analyzed within a rolling window, and a recompute must not re-age its row into it.
-- analysis is the sanitized jobfit.Analysis JSON.
INSERT INTO user_job_analysis (user_id, job_id, analysis, model, cv_uploaded_at, job_content_hash, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (user_id, job_id) DO UPDATE
SET analysis         = EXCLUDED.analysis,
    model            = EXCLUDED.model,
    cv_uploaded_at   = EXCLUDED.cv_uploaded_at,
    job_content_hash = EXCLUDED.job_content_hash;

-- name: CountRecentUserJobAnalyses :one
-- How many distinct jobs the caller first analyzed within the window (created_at is the
-- first-analysis time — see UpsertUserJobAnalysis). This is the fit-analysis quota
-- meter: the PK guarantees one row per (user, job), so the row count is the distinct-job
-- count. A recompute does not add a row, so it never consumes quota.
SELECT count(*)
FROM user_job_analysis
WHERE user_id = $1 AND created_at >= $2;
