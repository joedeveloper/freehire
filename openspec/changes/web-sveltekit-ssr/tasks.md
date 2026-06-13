## 1. SvelteKit scaffold & shared client

- [x] 1.1 Add SvelteKit + `adapter-node` to `web/` (deps, `svelte.config.js`,
  `vite.config.ts`, `app.html`, `src/app.d.ts`); keep `app.css`, `lib/`
  components, `types.ts`. `svelte-check` passes on the empty skeleton.
- [x] 1.2 Refactor `lib/api.ts` to accept an injected `fetch` so the same client
  works in server `load` and in the browser; keep the existing function shapes.
- [x] 1.3 Wire the dev server to proxy `/api` (and `/health`) to the Go backend
  so dev stays same-origin (replaces the Vite proxy); a dev request to `/api/v1/jobs`
  reaches the backend.

## 2. Public read routes (SSR)

- [ ] 2.1 `src/routes/+page.svelte` + `+page.server.ts`: jobs list, first page
  server-rendered from `GET /api/v1/jobs` via `event.fetch`; rows present in
  initial HTML; "load more" and reach indicators preserved.
- [ ] 2.2 `src/routes/jobs/[slug]/+page.svelte` + `+page.server.ts`: job detail
  server-rendered; 404 from the API yields an error page with a not-found status;
  closed-job state and Apply link preserved.
- [ ] 2.3 `src/routes/companies/+page.svelte` + `load`: companies list
  server-rendered, with the debounced name search URL-synced (`?q=`) preserved.
- [ ] 2.4 `src/routes/companies/[slug]/+page.svelte` + `+page.server.ts`: company
  detail server-rendered, reusing the job-row presentation.

## 3. Layout, auth, theme, interactive surfaces

- [ ] 3.1 Root `+layout.svelte` + `+layout.server.ts`: resolve the current user
  from `/me` (cookie forwarded) so signed-in chrome renders server-side without a
  post-mount flash; client hydrates from layout data.
- [ ] 3.2 Theme: persist the choice in a cookie so SSR sets the `.dark` class on
  `<html>`; keep the system-mode inline fallback in `app.html`; no theme FOUC.
- [ ] 3.3 Port filter URL-sync to SvelteKit (`$page.url.searchParams` +
  `replaceState`), preserving the synchronous write-state→URL-in-handler pattern
  (no controlled-input revert).
- [ ] 3.4 Port `/my/jobs` and `/my/api-keys` routes (auth-guarded) and the job
  "record view" client behavior; retire `App.svelte` and `router.svelte.ts`.

## 4. SEO artifacts

- [ ] 4.1 Per-route `<head>`: job-specific `<title>`/description/canonical/OG on
  `/jobs/:slug`; page-appropriate metadata on the list pages.
- [ ] 4.2 `JobPosting` JSON-LD builder (unit-testable, from the job-view shape) +
  `Organization` JSON-LD on company pages; emitted server-side; closed jobs
  reflect their status.
- [ ] 4.3 `src/routes/robots.txt/+server.ts`: real `text/plain` robots file
  referencing the sitemap.
- [ ] 4.4 `src/routes/sitemap.xml/+server.ts`: generated XML of job and company
  URLs (decide source: existing listing vs. a minimal slug+timestamp query; if
  capped, `log` what is omitted).

## 5. Deploy topology

- [ ] 5.1 New `web/Dockerfile` running the `adapter-node` server; update
  `web/nginx.conf` so nginx fronts `/api`+`/health`→Go and everything else→Node;
  verify the full stack in Docker (`make up`) serves SSR pages.

## 6. Verification

- [ ] 6.1 Verify each spec scenario: `curl` `/jobs/:slug`, `/`, `/companies`
  return content-bearing HTML; `JobPosting` JSON-LD validates; `/robots.txt` is
  text/plain; `/sitemap.xml` is valid XML; signed-in SSR renders correct chrome;
  no hydration warnings; `svelte-check` clean.
