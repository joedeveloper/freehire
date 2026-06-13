## ADDED Requirements

### Requirement: Ingest jobs from forwarded LinkedIn alert emails

The system SHALL drain a dedicated mailbox over IMAP on each run, parse recognized LinkedIn
job-alert emails into job postings, and upsert each posting into the shared job pool with
`source = "linkedin"` and `external_id` set to the LinkedIn numeric job id, reusing the
existing `UpsertJob` write path (which enqueues enrichment in the same transaction).

#### Scenario: Saved-search digest email is ingested
- **WHEN** the mailbox contains an unread, authentic `email_job_alert_digest_01` message with N job cards
- **THEN** the system upserts N jobs with `source = "linkedin"`, each `external_id` equal to that card's `/jobs/view/<id>/` id, and marks the message read

#### Scenario: Similar-jobs email is ingested
- **WHEN** the mailbox contains an unread, authentic `email_jobs_viewed_job_reminder_01` message
- **THEN** the system upserts one job per job card using the same mapping

#### Scenario: The same job forwarded by multiple users is deduplicated
- **WHEN** two messages contain a card for the same LinkedIn job id
- **THEN** the system results in a single `jobs` row for `(source = "linkedin", external_id = <id>)`

#### Scenario: Ingested LinkedIn jobs are stubs without a description
- **WHEN** a job card is ingested
- **THEN** the stored job has its title, company, location, remote flag, and canonical URL set, and an empty description

### Requirement: Only authentic, recognized LinkedIn alerts are ingested

The system SHALL ingest a message only when it structurally matches a known LinkedIn alert
template (recognized markers plus job cards with canonical `linkedin.com/jobs/view/<id>`
links). Every other message — spam, non-LinkedIn mail, or a recognized message with no
parsable job cards — MUST be dropped without writing to the job pool.

#### Scenario: Unrecognized message is dropped
- **WHEN** an unread message does not match any known LinkedIn alert template
- **THEN** the system writes nothing to the job pool and marks the message read

#### Scenario: Recognized template with no parsable cards is dropped
- **WHEN** a message matches a template but yields zero job cards (e.g. a manual forward that quoted away the body)
- **THEN** the system writes nothing and marks the message read

### Requirement: User PII and tracking tokens are never stored

The system MUST NOT persist the raw forwarded email, the recipient's name, or any LinkedIn
per-recipient tracking token (`midToken`, `eid`, `otpToken`, `midSig`, `trk*`, `refId`). Each
stored job URL MUST be the canonical form `https://www.linkedin.com/jobs/view/<id>/` with all
query parameters removed and any localized host normalized to `www.linkedin.com`.

#### Scenario: Tracking parameters are stripped from the job URL
- **WHEN** a job card links to `https://ar.linkedin.com/comm/jobs/view/4423193208/?trackingId=...&midToken=...&otpToken=...`
- **THEN** the stored URL is exactly `https://www.linkedin.com/jobs/view/4423193208/`

#### Scenario: No raw email or forwarder identity is retained
- **WHEN** a message has been processed
- **THEN** no stored field contains the raw message body, the recipient name, or who forwarded it

### Requirement: Gmail forwarding confirmation is auto-confirmed

The system SHALL recognize Google's forwarding-confirmation email and complete the
confirmation automatically (following the confirmation link, and submitting the confirmation
form if the link lands on one), so a user enabling auto-forwarding never has to retrieve a
code from the freehire mailbox.

#### Scenario: Confirmation email is auto-confirmed
- **WHEN** the mailbox contains an unread Google forwarding-confirmation email
- **THEN** the system follows its confirmation link, completes any confirmation form, and marks the message read

#### Scenario: Confirmation that cannot be completed is surfaced, not silently lost
- **WHEN** the confirmation link cannot be completed automatically
- **THEN** the system logs the failure (leaving it for manual confirmation) rather than reporting success

### Requirement: Processing is stateless and idempotent

The system SHALL keep no per-message state beyond the IMAP read flag. A message MUST be
marked read only after it has been successfully processed (saved, confirmed, or deliberately
dropped). On an infrastructure failure (mailbox or database unreachable), the run MUST abort
before marking any unprocessed message read, so the batch safely retries on the next run.

#### Scenario: Successful processing marks the message read
- **WHEN** a message is parsed and its jobs are saved
- **THEN** the message is marked read so the next run skips it

#### Scenario: Database failure aborts without losing the message
- **WHEN** the database is unreachable while saving a parsed message
- **THEN** the run aborts and the message remains unread for retry on the next run
