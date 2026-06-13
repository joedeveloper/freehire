## ADDED Requirements

### Requirement: Ingest persists job geography and work mode

The ingest write path SHALL parse each posting's `location` string into
`countries`/`regions`/`work_mode` (via the job-geography parser) and persist them
on the job row. These columns SHALL be written on insert and refreshed on
re-ingest, like the other raw source fields and unlike the enrichment payload
(which ingest never writes). A posting whose location yields no geography SHALL
store empty arrays.

For `work_mode`, when the adapter exposes a STRUCTURED work mode (a workplace-type
enum or an explicit remote flag from the ATS API) it SHALL take precedence over
the parser's free-text heuristic; the parser fills `work_mode` only when the
adapter has no structured signal.

#### Scenario: A new posting stores its parsed geography

- **WHEN** a posting with location `Remote - Germany` is ingested
- **THEN** the stored job has `countries=[de]` and `regions` including `eu`

#### Scenario: Re-ingest refreshes geography from the updated location

- **WHEN** an already-ingested posting is re-ingested with its location changed
  from `Remote - UK` to `Remote - USA`
- **THEN** the job's `countries` updates to `[us]` and its `regions` update
  accordingly

#### Scenario: A location with no geography stores empty arrays

- **WHEN** a posting with location `Remote` is ingested
- **THEN** the stored job has empty `countries` and empty `regions`

#### Scenario: A structured adapter work mode is persisted over the parsed one

- **WHEN** an adapter reports a structured `work_mode` (e.g. Lever's
  `workplaceType=hybrid`) for a posting whose location parses as `remote`
- **THEN** the stored `jobs.work_mode` is the structured value `hybrid`
