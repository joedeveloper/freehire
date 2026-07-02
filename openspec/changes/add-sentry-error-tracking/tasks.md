## 1. Backend config + observability helper

- [ ] 1.1 Add `SentryDSN` (`SENTRY_DSN`) and `SentryEnvironment` (`SENTRY_ENVIRONMENT`, default `"development"`) to `config.Settings` and `config.Load` (`internal/config/config.go`), following the existing optional-integration pattern.
- [ ] 1.2 Add `github.com/getsentry/sentry-go` (and the fiber submodule) via `go get`; commit `go.mod`/`go.sum`.
- [ ] 1.3 Create `internal/observability` with `Init(dsn, environment string) (flush func())`: no-op (nil init, no-op flush) when DSN empty; otherwise `sentry.Init` with `EnableTracing:false`, `SendDefaultPII:false`, environment tag, returning a `flush` that calls `sentry.Flush(2*time.Second)`. Unit-test the disabled path (no init, no-op flush) and that a bad DSN surfaces an error.

## 2. Worker wiring (all cron workers)

- [ ] 2.1 In `internal/worker/bootstrap.go`, call `observability.Init(cfg.SentryDSN, cfg.SentryEnvironment)` inside `Bootstrap` and fold the returned `flush` into the `cleanup` closure. Update `bootstrap_test.go` expectations if needed.
- [ ] 2.2 Add `worker.Main(run func() int)` (new `internal/worker/main.go` + test): defers a recover that calls `sentry.CurrentHub().Recover(r)`, `sentry.Flush`, then re-panics to preserve crash + non-zero exit; on the normal path calls `os.Exit(run())`.
- [ ] 2.3 Convert each cron worker `func main() { os.Exit(run()) }` to `func main() { worker.Main(run) }` across the 11 Bootstrap-using commands (enrich, ingest, tg-ingest, tg-extract, liveness, notify, reindex, backfill-derive, reslug, import-collections, recount-companies).

## 3. Server wiring

- [ ] 3.1 In `cmd/server/main.go`, call `observability.Init` at startup and invoke its `flush` after `app.ShutdownWithTimeout` (next to `tracerShutdown()`).
- [ ] 3.2 Register the fibersentry middleware early (with `Repanic:true`) so `recover.New()` still renders the standard 500 while panics are captured on a request-scoped hub.
- [ ] 3.3 Update `handler.RenderError` to report to `fibersentry.GetHubFromContext(c)` **only** on the fall-through 500 branch; leave `*fiber.Error`, `pgx.ErrNoRows`→404, and FK-violation→404 unreported. Add/adjust a test asserting a 500-mapped error captures and a 4xx/404 does not (using a stub hub or capture recorder).

## 4. Frontend wiring

- [ ] 4.1 Add `@sentry/sveltekit` to `web/package.json`; install.
- [ ] 4.2 Create `web/src/hooks.client.ts`: init Sentry only when `PUBLIC_SENTRY_DSN` is set (`tracesSampleRate:0`, `sendDefaultPii:false`, environment from `PUBLIC_SENTRY_ENVIRONMENT`); export `handleError = handleErrorWithSentry()`.
- [ ] 4.3 Create `web/src/hooks.server.ts`: same gated init; compose `sentryHandle()` into `handle` and export `handleError = handleErrorWithSentry()`.
- [ ] 4.4 Add `sentrySvelteKit()` to `web/vite.config.ts` (source-map upload inert without `SENTRY_AUTH_TOKEN`); verify `npm run build` succeeds without the token.
- [ ] 4.5 Add a comment in `web/svelte.config.js` recording the Sentry ingest host for a future `connect-src`; confirm no live CSP change is needed (no `default-src`/`connect-src` present today).

## 5. Verification + docs

- [ ] 5.1 `go build ./... && go vet ./... && go test ./...` green; `web` `npm run check` + `npm run build` green.
- [ ] 5.2 Manually verify (with a scratch DSN) that a forced server 500, a forced worker panic, and a forced frontend throw each land in the right Sentry project; confirm no events fire with the DSN unset.
- [ ] 5.3 Document the new env vars (`SENTRY_DSN`, `SENTRY_ENVIRONMENT`, `PUBLIC_SENTRY_DSN`, `PUBLIC_SENTRY_ENVIRONMENT`) and the two-project setup in the repo (CLAUDE.md/AGENT.md conventions) and note the `freehire-ops` injection points.
