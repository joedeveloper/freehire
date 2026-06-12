-- Job lifecycle: liveness tracking + soft closing (see openspec change close-stale-jobs).
--
-- last_seen_at: stamped by UpsertJob on every ingest write; the DEFAULT backfills
-- existing rows at migration time so nothing closes until a full grace window has
-- passed after deploy.
-- closed_at: NULL = open. Set by the post-ingest sweep when a job hasn't been seen
-- within the grace window; cleared by UpsertJob when the posting reappears.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS closed_at    TIMESTAMPTZ;
