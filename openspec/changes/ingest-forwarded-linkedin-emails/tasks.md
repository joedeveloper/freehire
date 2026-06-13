## 1. Fixtures and package skeleton

- [ ] 1.1 Commit the two captured LinkedIn alert emails as test fixtures under `internal/mailingest/testdata/` (`digest_01.eml`, `similar_jobs_01.eml`), plus one non-LinkedIn email and one Google forwarding-confirmation email for negative/confirmation tests
- [ ] 1.2 Create `internal/mailingest` package with the `Posting` type (ExternalID, URL, Title, Company, Location, Remote) and a doc comment describing the firehose scope

## 2. Email parsing (pure functions, fixture-driven)

- [ ] 2.1 MIME helper: from a raw RFC822 message, decode `quoted-printable` and return the `text/html` and `text/plain` parts
- [ ] 2.2 Template recognition: classify a message as `digest` | `similar` | `unknown` from its markers (`X-LinkedIn-Template` when present, plus job-card structure)
- [ ] 2.3 HTML job-card parser: extract `[]Posting` from `data-test-id="job-card"` blocks (id from `/jobs/view/<id>/`, company from logo `img alt`, title from anchor, location + trailing `(Remote)` from the `<p>`)
- [ ] 2.4 `text/plain` fallback parser used only when HTML yields nothing; assert it skips variable flag lines ("Fast growing", "Easy Apply", …)
- [ ] 2.5 Authenticity gate: a message is ingestable only if it recognizes as a known template AND its cards carry canonical `linkedin.com/jobs/view/<id>` links; everything else returns "drop"

## 3. Scrubbing

- [ ] 3.1 URL canonicalizer: any LinkedIn job link → `https://www.linkedin.com/jobs/view/<id>/` (strip all query params; normalize localized hosts like `ar.linkedin.com` → `www.linkedin.com`)
- [ ] 3.2 Assert no `Posting` field or save call ever carries a tracking token (`midToken`/`eid`/`otpToken`/`midSig`/`trk*`/`refId`) or the recipient name

## 4. Save seam (reuse existing write path)

- [ ] 4.1 Extract or expose a reusable normalize+upsert seam so a `Posting` maps to a job and persists via the existing `UpsertJob` (source="linkedin", external_id=job id, CompanySlug/PublicSlug derived, empty description), sharing the path with `cmd/ingest`
- [ ] 4.2 Test: saving the same job id twice results in one row; a new job is enqueued for enrichment exactly as ATS ingest does

## 5. Gmail forwarding-confirmation handling

- [ ] 5.1 Detect a Google forwarding-confirmation email (sender + subject/body markers) and extract its confirmation link
- [ ] 5.2 Confirm by following the link and submitting the confirmation form if the link lands on one; on failure, log and report not-confirmed (never report success)

## 6. Message routing and run orchestration

- [ ] 6.1 Router: classify each fetched message as `confirmation` | `alert` | `drop`
- [ ] 6.2 Runner: for each unread message route → (parse → scrub → save) or confirm or drop; mark `\Seen` only after success/deliberate-drop; abort the run before marking on infra failure (mailbox/DB unreachable); return run stats
- [ ] 6.3 Tests for the runner over the fixtures: digest+similar ingested, non-LinkedIn dropped, confirmation auto-confirmed, DB-failure aborts leaving the message unread

## 7. IMAP client and worker entry point

- [ ] 7.1 IMAP client (emersion/go-imap + go-message): connect, fetch `UNSEEN`, mark `\Seen`; add the dependency
- [ ] 7.2 Add IMAP config (host/user/app-password) to `internal/config` following existing env-config style; document the new env vars
- [ ] 7.3 `cmd/ingest-mail/main.go`: load config, build the runner, run once, log stats, exit — mirroring `cmd/ingest`

## 8. Wiring and operator docs

- [ ] 8.1 Update CLAUDE.md layout/commands and add a short operator note (provision mailbox + app-password, dry-run against fixtures, cron entry, rollback = remove cron)
- [ ] 8.2 Write the end-user "set up auto-forwarding in Gmail" instructions (filter recipe + the "confirmation completes automatically within a few minutes" note)
