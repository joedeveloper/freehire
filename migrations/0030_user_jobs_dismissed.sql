-- Swipe-mode triage: dismissed_at marks a job the user swiped away in the swipe
-- deck. NULL = not dismissed. Like saved_at it lives on the same one-row-per-
-- (user, job) interaction as viewed_at/applied_at; undismissing clears the column
-- without deleting the row, so view/apply/save history survives. Dismissal only
-- keeps a job out of the swipe deck — it does NOT hide the job from the public
-- /jobs list or search. Applied automatically by Postgres on first volume init
-- (same as 0001) and also serves as schema source for sqlc. Existing volumes/prod
-- need a manual ALTER (the versioned-migration-runner seam from AGENT.md remains open).

ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS dismissed_at TIMESTAMPTZ;
