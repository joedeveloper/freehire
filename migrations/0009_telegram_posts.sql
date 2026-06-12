-- Telegram channel posts: the crawl-dedup record AND the durable extraction queue
-- in one table. One row per (channel, msg_id); the crawl inserts with ON CONFLICT
-- DO NOTHING so re-crawling a channel never re-processes a seen post. Rows persist
-- after extraction (unlike enrichment_outbox, which deletes) because the row itself
-- is the "already seen" record for the next crawl.
--
-- Extraction bookkeeping mirrors enrichment_outbox: lease via claimed_at (expiry is
-- the crash reaper), retry counting via attempts, dead-letter via failed_at.
-- extracted_at non-NULL = done — either jobs were written or the post held no
-- vacancy (including posts the ingest prefilter rejected at insert).
--
-- Applied by Postgres on first volume init (same as 0001-0008) and read by sqlc.
CREATE TABLE IF NOT EXISTS telegram_posts (
    channel      TEXT        NOT NULL,
    msg_id       BIGINT      NOT NULL,
    text         TEXT        NOT NULL,
    posted_at    TIMESTAMPTZ NOT NULL,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    attempts     INT         NOT NULL DEFAULT 0,
    claimed_at   TIMESTAMPTZ,            -- lease stamp; NULL = unleased
    failed_at    TIMESTAMPTZ,            -- non-NULL = dead-lettered, never reclaimed
    last_error   TEXT        NOT NULL DEFAULT '',
    extracted_at TIMESTAMPTZ,            -- non-NULL = done (jobs written or none found)

    PRIMARY KEY (channel, msg_id)
);

-- The claim scans only pending work: not done, not dead-lettered. Oldest post first
-- so a backlog drains in posting order.
CREATE INDEX IF NOT EXISTS telegram_posts_claimable_idx
    ON telegram_posts (posted_at)
    WHERE extracted_at IS NULL AND failed_at IS NULL;
