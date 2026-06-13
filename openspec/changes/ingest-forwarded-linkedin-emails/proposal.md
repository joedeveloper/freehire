## Why

LinkedIn job-alert emails reach exactly the people freehire wants — engineers actively
watching the market — but that supply never enters our pool because LinkedIn has no public
API and crawling it is bot-walled and ToS-grey. Letting any user **forward** those alert
emails to a freehire mailbox turns a stream we cannot crawl into one users hand us
voluntarily, growing the shared job pool from real demand signals. The bar is that
participating must be **safe and simple**: forwarding gives us no access to the user's
mailbox (no password, no OAuth), is one-directional, and is revocable by deleting one rule.

## What Changes

- Add a new standalone, run-once-and-exit worker `cmd/ingest-mail` (cron-scheduled, same
  pattern as `cmd/ingest` / `cmd/enrich`) that drains a dedicated mailbox over IMAP.
- Add `internal/mailingest`: an IMAP reader, a LinkedIn job-alert parser (two known
  templates), a Gmail forwarding-confirmation auto-responder, and a PII/token scrubber.
- Parsed postings flow into the existing pipeline write path (`UpsertJob`) with
  `source = "linkedin"`, `external_id = <LinkedIn job id>`, deduped like every other source
  and auto-enqueued for enrichment in the same write.
- **Auto-confirm** the Gmail forwarding handshake: detect Google's confirmation email in the
  mailbox and follow its link automatically, so users never fish a code out of our inbox.
- **Authenticity gate before any write**: only ingest messages that are genuinely from
  LinkedIn (DKIM/sender check) AND match a recognized alert template; silently drop the rest.
- **Strip user PII**: discard recipient name and personal LinkedIn tracking tokens
  (`midToken`, `eid`, `otpToken`, `midSig`), canonicalize each job URL to
  `https://www.linkedin.com/jobs/view/<id>/`, and never persist the raw email or who sent it.
- Add IMAP connection config (host / user / app-password) to `internal/config`.

Accepted tradeoff (decided with the user): emails carry **no job description**, so ingested
LinkedIn jobs are **stubs** (title/company/location/url). Enrichment will run near-empty for
them until description backfill exists. This is intentional for the MVP.

## Capabilities

### New Capabilities
- `email-ingest`: ingest jobs from user-forwarded LinkedIn job-alert emails — mailbox
  draining, authenticity + template gating, PII/token scrubbing, forwarding-confirmation
  auto-response, and normalization into the shared job pool.

### Modified Capabilities
<!-- None. source-ingest's UpsertJob write path is reused as-is, not changed. -->

## Impact

- **New code**: `cmd/ingest-mail/`, `internal/mailingest/`. New IMAP client dependency
  (e.g. `github.com/emersion/go-imap` + `go-message`).
- **Config**: new IMAP env vars in `internal/config`; the worker shares `config.Load`.
- **DB**: none — reuses `jobs` + `enrichment_outbox` via existing `UpsertJob`. No migration.
- **Ops**: a new cron entry; a provisioned mailbox + app-password secret.
- **Out of scope (future seams)**: per-user forwarding addresses + attribution; description
  backfill via the throttled `jobs-guest` endpoint (needs antibot infra); non-Gmail providers
  (Outlook/Yahoo confirmation differs); non-LinkedIn alert sources (Indeed/Otta/Wellfound);
  hosted-inbound (SES/Mailgun/Cloudflare) as an alternative to IMAP polling.
