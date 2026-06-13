-- name: RecordJobView :one
-- Record (or refresh) a user's view of a job. Idempotent on (user_id, job_id):
-- the first view creates the row, a repeat view touches viewed_at. Returns the
-- row so the caller learns the current applied_at in the same round-trip.
INSERT INTO user_jobs (user_id, job_id)
VALUES ($1, $2)
ON CONFLICT (user_id, job_id) DO UPDATE SET viewed_at = now()
RETURNING *;

-- name: MarkJobApplied :one
-- Mark a job as applied for a user. Idempotent and independent of a prior view:
-- it inserts the row (viewed_at defaults) or updates applied_at in place.
INSERT INTO user_jobs (user_id, job_id, applied_at)
VALUES ($1, $2, now())
ON CONFLICT (user_id, job_id) DO UPDATE SET applied_at = now()
RETURNING *;

-- name: SaveJob :one
-- Save (bookmark) a job for a user. Idempotent and independent of a prior view:
-- it inserts the row (viewed_at defaults) or refreshes saved_at in place.
INSERT INTO user_jobs (user_id, job_id, saved_at)
VALUES ($1, $2, now())
ON CONFLICT (user_id, job_id) DO UPDATE SET saved_at = now()
RETURNING *;

-- name: UnsaveJob :one
-- Clear a job's saved mark without deleting the interaction row, so view and
-- apply history survive unsaving. No interaction row -> pgx.ErrNoRows; the
-- handler treats that as "already not saved", never as a failure.
UPDATE user_jobs
SET saved_at = NULL
WHERE user_id = $1 AND job_id = $2
RETURNING *;

-- name: ListUserJobs :many
-- A user's job interactions joined with the job rows, most recently touched
-- first (GREATEST ignores NULLs; viewed_at is always set). filter narrows to
-- viewed-only/saved/applied subsets; 'all' is every interaction, 'viewed' is
-- the passive history (rows neither saved nor applied). Closed jobs stay
-- listed: a user's history must not shrink when a posting closes.
SELECT sqlc.embed(jobs), uj.viewed_at, uj.saved_at, uj.applied_at
FROM user_jobs uj
JOIN jobs ON jobs.id = uj.job_id
WHERE uj.user_id = $1
  AND (sqlc.arg(filter)::text = 'all'
       OR (sqlc.arg(filter)::text = 'viewed' AND uj.saved_at IS NULL AND uj.applied_at IS NULL)
       OR (sqlc.arg(filter)::text = 'saved' AND uj.saved_at IS NOT NULL)
       OR (sqlc.arg(filter)::text = 'applied' AND uj.applied_at IS NOT NULL))
ORDER BY GREATEST(uj.viewed_at, uj.saved_at, uj.applied_at) DESC, uj.job_id DESC
LIMIT $2 OFFSET $3;

-- name: CountUserJobs :one
-- Per-filter row counts for the my-jobs tabs, in one aggregate pass. "all" is
-- every interaction row; "viewed" is the view-only subset (neither saved nor
-- applied), matching the ListUserJobs filter.
SELECT count(*)                                        AS "all",
       count(*) FILTER (WHERE saved_at IS NULL
                          AND applied_at IS NULL)      AS viewed,
       count(*) FILTER (WHERE saved_at   IS NOT NULL) AS saved,
       count(*) FILTER (WHERE applied_at IS NOT NULL) AS applied
FROM user_jobs
WHERE user_id = $1;
