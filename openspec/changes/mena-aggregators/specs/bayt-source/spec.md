## ADDED Requirements

### Requirement: Bayt per-country listing crawl

The system SHALL provide a `bayt` source adapter that crawls Bayt.com's paginated
per-country job listings into the catalogue. The adapter is a **boardless aggregator**:
its configured entries are country scopes (e.g. `saudi-arabia`, `uae`, `qatar`, `kuwait`,
`bahrain`, `oman`, `egypt`, `jordan`), not company boards. For each configured country it
walks the listing pages, collects job-detail URLs, fetches each detail page, and parses
the embedded `JobPosting` JSON-LD. The crawl is keyless (no API key).

#### Scenario: Country scope yields all listed postings

- **WHEN** the adapter crawls a configured country scope
- **THEN** it returns one `Job` per job-detail link discovered across that country's
  listing pages, each populated from the detail page's `JobPosting` JSON-LD (title,
  HTML description, company, free-text location, apply/detail URL, posted-at)

#### Scenario: Pagination is followed to exhaustion

- **WHEN** a country's listings span multiple pages
- **THEN** the adapter advances through the pages until a page yields no new job-detail
  links, so postings beyond the first page are not silently dropped

### Requirement: Self-contained aggregator company identity

The adapter MUST read each posting's company from the JSON-LD `hiringOrganization.name`
rather than a configured board company, because Bayt is a multi-company aggregator where
each posting carries its own employer. The adapter is registered as an aggregator-marker
source so its jobs are stored under their own company identity.

#### Scenario: Company comes from the posting

- **WHEN** a detail page's `JobPosting` names a `hiringOrganization`
- **THEN** that organization name is the job's company

#### Scenario: Posting without an employer is dropped

- **WHEN** a detail page has no resolvable `hiringOrganization.name`
- **THEN** the adapter omits that posting rather than persisting a company-less job

### Requirement: Stable dedup identity

The adapter SHALL derive each job's `ExternalID` from the stable Bayt posting id (the
numeric id in the detail URL / JSON-LD `identifier`), so re-crawling the same country
dedups to the same catalogue row.

#### Scenario: Re-crawl dedups

- **WHEN** the same posting is seen on a later crawl
- **THEN** it maps to the same `ExternalID` and updates the existing row rather than
  creating a duplicate; a posting with no extractable id is dropped rather than persisted
  with an empty key

### Requirement: Anti-bot fingerprint transport

The adapter MUST issue its requests over the shared Chrome-fingerprint HTTP transport
(TLS + HTTP/2 spoofing), because Bayt's Akamai/Cloudflare edge rejects the default Go
fingerprint with a 403. Requests still dial through the SSRF-guarded dialer.

#### Scenario: Requests are served, not blocked

- **WHEN** the adapter fetches a Bayt listing or detail page
- **THEN** it uses the Chrome-fingerprint transport so the edge serves a 200 with real
  markup rather than an anti-bot 403

### Requirement: Resilient JSON-LD parsing

The postings live in `JobPosting` JSON-LD, so the adapter SHALL fail with an error when a
detail page carries no `JobPosting` JSON-LD block (a markup change must surface loudly
rather than silently empty the catalogue), while a listing page with zero job links is
valid and simply ends pagination.

#### Scenario: Missing JobPosting JSON-LD errors

- **WHEN** a fetched detail page has no `JobPosting` JSON-LD
- **THEN** the adapter reports an error for that page rather than treating it as an empty
  success
