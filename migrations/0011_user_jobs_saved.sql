-- Bookmarking: saved_at marks a job the user saved for later. NULL = not saved.
-- It lives on the same one-row-per-(user, job) interaction as viewed_at/applied_at;
-- unsaving clears the column without deleting the row, so view/apply history
-- survives. Applied automatically by Postgres on first volume init (same as 0001)
-- and also serves as schema source for sqlc. Existing volumes/prod need a manual
-- ALTER (the versioned-migration-runner seam from AGENT.md remains open).

ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS saved_at TIMESTAMPTZ;
