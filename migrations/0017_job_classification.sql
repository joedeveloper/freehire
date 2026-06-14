-- Job classification (see specs/2026-06-14-deterministic-classification): the
-- seniority and role category derived deterministically from the title at ingest
-- by internal/classify. These are SOURCE facts (parsed from the title), stored
-- beside — not inside — the `enrichment` JSONB payload, so the LLM enrichment
-- worker (the sole writer of that blob) never clobbers them. Both DEFAULT '' so a
-- title that resolves nothing stores the empty string, never NULL.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS seniority TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS category  TEXT NOT NULL DEFAULT '';
