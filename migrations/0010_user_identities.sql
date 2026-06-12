-- External sign-in identities (OAuth providers): links a provider-side user id
-- to a local account. Reference-only — no provider tokens, no profile copy;
-- users stays canonical.
-- Applied automatically by Postgres on first volume init (same as 0001) and
-- also serves as schema source for sqlc.

CREATE TABLE IF NOT EXISTS user_identities (
    provider         TEXT        NOT NULL,
    provider_user_id TEXT        NOT NULL,
    user_id          BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One local account per provider identity; the natural key doubles as the
    -- dedup key for concurrent first-sign-in callbacks.
    PRIMARY KEY (provider, provider_user_id)
);

-- Reverse lookup: all identities of one user (future unlink/management UI).
CREATE INDEX IF NOT EXISTS user_identities_user_id_idx ON user_identities (user_id);
