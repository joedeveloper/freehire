-- Moderator-authored jobs: a user role for authorization, and authorship audit on jobs.

-- role gates the privileged write endpoints. It lives in the DB (not the JWT) so a
-- role change takes effect on the next request. 'admin' is reserved for later; the
-- CHECK keeps the column a closed vocabulary the way enrichment enums are.
ALTER TABLE users
    ADD COLUMN role TEXT NOT NULL DEFAULT 'user'
        CHECK (role IN ('user', 'moderator', 'admin'));

-- Authorship audit for hand-curated jobs. NULL for every automated source (ingest,
-- telegram, link-following) — they have no acting user. created_by is stamped once
-- at creation; updated_by is rewritten on each moderator edit. Both reference users
-- but are not part of the public wire shape.
-- ON DELETE SET NULL (not CASCADE, the convention for the ownership FKs on user_jobs/
-- api_keys/user_identities): these are audit references, not ownership — deleting the
-- authoring user must blank the audit, never delete the job.
ALTER TABLE jobs
    ADD COLUMN created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
