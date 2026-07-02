// Package observability wires optional error reporting (Sentry) for the Go HTTP
// server and the standalone workers. It is opt-in and env-gated: without a DSN it
// is a no-op, mirroring the other optional integrations (search, blobstore) so an
// unconfigured deployment runs unchanged.
package observability

import (
	"time"

	"github.com/getsentry/sentry-go"
)

// flushTimeout bounds how long the returned flush waits for buffered events to
// reach Sentry. Short-lived cron workers call it on exit, so it stays small enough
// not to stall shutdown yet large enough for one delivery round trip.
const flushTimeout = 2 * time.Second

// Init initializes Sentry error reporting and returns a flush to run before the
// process exits (delivers buffered events). When dsn is empty it initializes
// nothing and returns a no-op flush, so an unconfigured deployment is unaffected.
// A malformed DSN is returned as an error so a misconfigured process fails fast.
//
// PII is off by default (SendDefaultPII false) and tracing is disabled: this is an
// errors-only integration — no request bodies, cookies, or auth headers are shipped.
func Init(dsn, environment string) (flush func(), err error) {
	if dsn == "" {
		return func() {}, nil
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:            dsn,
		Environment:    environment,
		EnableTracing:  false,
		SendDefaultPII: false,
	}); err != nil {
		return nil, err
	}
	return func() { sentry.Flush(flushTimeout) }, nil
}
