-- Deterministic skill tags parsed from the job description at ingest by
-- internal/skilltag. Sibling of the location-derived columns (0012): a source fact
-- stored next to (not inside) the enrichment blob, so the enrichment worker — which
-- rewrites the whole blob — never clobbers it. NOT NULL with a '{}' default so a job
-- with no resolvable skills stores an empty array, never NULL.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS skills TEXT[] NOT NULL DEFAULT '{}';
