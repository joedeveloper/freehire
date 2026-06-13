## Context

freehire ingests jobs by crawling public ATS APIs (`internal/sources` → `pipeline.Runner`
→ `UpsertJob`). LinkedIn has no such API and is bot-walled, so its postings never enter the
pool — yet LinkedIn job-alert emails land daily in the inboxes of exactly our target users.
This change lets a user **forward** those alert emails to a freehire mailbox; we parse them
and upsert the jobs.

Two LinkedIn alert templates were captured and analyzed live (see
`memory/linkedin-alert-emails-feasibility`):
`email_job_alert_digest_01` (saved-search digest) and
`email_jobs_viewed_job_reminder_01` (similar jobs). Each is a MIME multipart with a
`text/html` part whose job cards carry everything we need: title, company, location, the
canonical numeric LinkedIn job id (in the `/jobs/view/<id>/` path), and a (personalized)
URL. There is **no job description** in the email.

Constraints: Go + Fiber v2, sqlc (no ORM), standalone run-once workers scheduled by cron
(`cmd/ingest`, `cmd/enrich`), English only, "no infra before there's a concrete need"
(CLAUDE.md). The project is MVP-stage, so reshaping the ingest seam to fit is preferred over
bolting on a special case.

## Goals / Non-Goals

**Goals:**
- A user can forward LinkedIn job-alert emails to one freehire address and have the jobs
  appear in the shared pool, with **no access granted to their mailbox** (no password, no
  OAuth) — forwarding is one-directional and revocable.
- "Set and forget": after a one-time Gmail forwarding setup, ingestion is fully automatic,
  including auto-confirming Gmail's forwarding handshake so the user never handles a code.
- Never persist user PII or LinkedIn tracking tokens; store only public job fields.
- Reject anything that is not a recognized, authentic LinkedIn alert before any DB write.
- Reuse the existing normalize + `UpsertJob` + enrichment-enqueue write path unchanged.

**Non-Goals:**
- Per-user forwarding addresses or attributing jobs to the forwarder (global firehose only).
- Fetching job descriptions (emails lack them; ingested LinkedIn jobs are stubs). Backfill
  via the `jobs-guest` endpoint is a separate, later effort (it throttles per-IP — measured
  13/16 OK, 3×429 on a burst — and is ToS-grey).
- Non-Gmail providers (Outlook/Yahoo confirmation flows differ) and non-LinkedIn alert
  sources (Indeed/Otta/Wellfound).
- Hosted inbound (SES/Mailgun/Cloudflare) — see Decision 1.

## Decisions

### 1. Receive via IMAP polling of one mailbox, not hosted inbound

A new `cmd/ingest-mail` run-once worker logs into a dedicated mailbox (Gmail/Fastmail) over
IMAP, reads `UNSEEN` messages, processes them, and marks them `\Seen`. Cron-scheduled like
`cmd/ingest`/`cmd/enrich`.

- **Why:** zero infrastructure and architecturally native — a third worker in an existing
  pattern, reusing the same `config.Load` and write path. Honors "no infra before need".
- **Alternative (rejected for MVP):** hosted inbound parse — point a `freehire.dev` subdomain
  MX at SES/Mailgun/Cloudflare Email Workers, receive each mail as a webhook. Lower latency
  and "own domain", with built-in SPF/DKIM handling, but it adds DNS/MX setup, a provider
  account, and a **new inbound HTTP ingress on `cmd/server`** (the worker pattern no longer
  fits). Recorded as the scale path / future seam, not MVP.

### 2. New package `internal/mailingest`; reuse the lower write path, not `pipeline.Runner`

`pipeline.Runner` is tied to `sources.Source.Fetch(CompanyEntry)` — boards we crawl. Email
ingest has no board, so it does not implement `Source`. Instead `mailingest` produces
normalized jobs and calls the **same store write** (`UpsertJob`, which also enqueues
enrichment in-transaction). If the normalize-and-save step is currently entangled with the
Runner, extract it into a small reusable seam so both `cmd/ingest` and `cmd/ingest-mail`
share it (MVP architecture is fluid; reshape rather than duplicate).

Package shape:
- `imap.go` — connect, fetch `UNSEEN`, mark `\Seen` (emersion/go-imap + go-message).
- `router.go` — classify each message: **confirmation** | **alert** | **drop**.
- `parser.go` — parse the two LinkedIn templates into `[]Posting` from the `text/html` part.
- `confirm.go` — handle Gmail forwarding-confirmation emails (Decision 5).
- `scrub.go` — strip tokens, canonicalize URL (Decision 4).
- `runner.go` — orchestrate: fetch → route → (parse → scrub → save) | confirm → mark seen.

### 3. Dedup key and field mapping

`source = "linkedin"`, `external_id = <LinkedIn numeric job id>` (no `board:` prefix — there
is no board; the job id is globally unique on LinkedIn). Title/company/location/remote come
from the job card; `CompanySlug = normalize.Slug(company)`,
`PublicSlug = normalize.JobSlug(...)`, `Description = ""`. Re-forwards by multiple users
dedup naturally on `(source, external_id)`.

### 4. Parse `text/html` job cards; scrub aggressively

- **Primary:** iterate `data-test-id="job-card"` blocks in the `text/html` part. Job id from
  the `/jobs/view/<id>/` path; company from the logo `img alt`; title from the card anchor
  text; location from the `<p>` (a trailing `(Remote)` sets `remote=true`).
- **Fallback:** the `text/plain` part's repeating `title / company / location / View job: URL`
  blocks, used only if HTML parsing yields nothing.
- **Scrub (mandatory):** the canonical URL is rebuilt as
  `https://www.linkedin.com/jobs/view/<id>/` — drop every query param (`midToken`, `eid`,
  `otpToken`, `midSig`, `trk*`, `refId`, …) and normalize localized hosts (`ar.linkedin.com`
  → `www.linkedin.com`). The recipient name and tokens are never extracted into a stored
  field. The raw message lives only in worker memory and is discarded after the run.
- Decode `quoted-printable` (both sample emails use it). Auto-forward preserves the original
  body, so the markers survive; a manually-forwarded message may wrap the body in a quote —
  parse defensively and `drop` if no job cards are found.

### 5. Auto-confirm the Gmail forwarding handshake

When a user adds our address as a Gmail forwarding target, Google sends a confirmation email
(`from forwarding-noreply@google.com`) containing a confirmation **link** and a numeric code.
`confirm.go` recognizes this message, extracts the link, and issues an HTTP GET to it; if the
link lands on a confirmation page with a "Confirm" form, follow that form. The user never
touches the code.

- **Why:** this is what makes it "set and forget"; without it the user would have to retrieve
  a code from our inbox (impossible) or we'd require manual coordination.
- **Latency note:** with IMAP polling, confirmation happens on the next poll cycle, not
  instantly — the setup instructions will say "confirmation completes automatically within a
  few minutes."
- **Alternative considered:** ask the user to type the numeric code into their own Gmail
  settings — rejected, it breaks "set and forget".

### 6. Authenticity gate before any write

Only ingest a message that (a) structurally matches a known LinkedIn alert (recognized
template markers: `X-LinkedIn-Template` header when preserved, plus job-card markers with
canonical `linkedin.com/jobs/view/<id>` links and `media.licdn.com` logos) and, (b) where the
original `DKIM-Signature: d=linkedin.com` survives the forward, passes that check as a
strengthening signal. Everything else is dropped silently.

- **Honesty about DKIM:** forwarding frequently breaks DKIM (body re-encoding; this is why
  ARC exists), so DKIM-pass is treated as **best-effort reinforcement, not a hard gate**. The
  necessary gate is the structural template match.
- **Residual risk & blast radius:** a determined attacker could hand-craft a lookalike and
  inject stub jobs. Impact is bounded — every entry is keyed by a real LinkedIn job id, lives
  in a public pool, and is prunable. Stronger verification (resolving the job id against
  LinkedIn's guest endpoint, or requiring ARC-validated chains) is a documented hardening
  seam, out of MVP.

### 7. Mailbox state is the only state; processing is stateless and idempotent

No new DB table. Per message: if processing succeeds (parsed-and-saved, confirmed, or a
recognized-but-unparseable drop) → mark `\Seen`. Only **infra-level** failures (IMAP/DB
unreachable) abort the run **before** marking, so the whole batch safely retries next cron
tick. This avoids poison-message loops without per-message attempt tracking.

## Risks / Trade-offs

- **Stub-only data (no description)** → Accepted, explicit MVP tradeoff. Enrichment runs
  near-empty for LinkedIn jobs until a description-backfill seam exists. Documented so it is
  not mistaken for a bug.
- **DKIM unreliable across forwarding** → Don't hard-gate on it; gate on structural template
  match and accept bounded poisoning risk; note ARC/id-resolution as hardening.
- **Gmail confirmation link may need a form POST, not just a GET** → `confirm.go` parses the
  landing page and submits the confirm form if present; if confirmation can't be completed,
  log it (a human can confirm once) rather than silently failing.
- **Gmail may flag automated IMAP access / app-passwords** → Use an app-password or XOAUTH2;
  if Gmail proves hostile to automation, the same parser/scrubber feeds Option B (hosted
  inbound) without rework — the IMAP layer is isolated behind `router.go`.
- **LinkedIn changes its email template** → Parser is template-versioned and fails closed
  (drop, don't mis-ingest); add a test fixture per template so breakage is caught by tests.
- **Open firehose abuse** → Authenticity gate (Decision 6) + stub blast-radius bound.

## Migration Plan

Additive and reversible. No schema change.

1. Land `internal/mailingest` + `cmd/ingest-mail` behind no caller; nothing runs until cron
   is wired.
2. Provision a mailbox + app-password; set IMAP env vars as an ops secret.
3. Dry-run the worker against a test mailbox seeded with the two captured sample emails
   (committed as fixtures) before pointing real forwarding at it.
4. Add the cron entry. **Rollback:** remove the cron entry (and revoke the app-password); no
   data migration to undo — ingested LinkedIn stubs are ordinary `jobs` rows.

## Open Questions

- **Mailbox provider:** Gmail (app-password/XOAUTH2 friction, automation flags) vs Fastmail
  (clean IMAP, paid) vs a Google Workspace address on `freehire.dev`. Affects only ops, not
  the design. To confirm at implementation time.
- **`\Seen` vs move-to-folder vs delete** for processed mail — `\Seen` is simplest and
  auditable; revisit only if the mailbox needs hygiene.
- **IMAP library:** emersion/go-imap v2 vs v1 — pick the currently-stable line at
  implementation; `go-message` for MIME either way.
