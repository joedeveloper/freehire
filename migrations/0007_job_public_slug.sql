-- Public, non-enumerable job identifier exposed by the API in place of the
-- sequential bigint id. Generated deterministically from (source, external_id)
-- plus the normalized title/company (see internal/normalize.JobSlug), so it is
-- stable across re-ingests. Applied on first volume init (same as 0001) and also
-- serves as schema source for sqlc.
--
-- Added on a freshly-initialized (empty) jobs table, so NOT NULL needs no
-- backfill. The UNIQUE constraint doubles as the lookup index for GetJobBySlug.
ALTER TABLE jobs ADD COLUMN public_slug TEXT NOT NULL UNIQUE;
