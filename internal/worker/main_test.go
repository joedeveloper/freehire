package worker

import (
	"context"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// recordingTransport is a sentry.Transport that captures sent events in memory so
// a test can assert what would have been delivered without any network.
type recordingTransport struct{ events []*sentry.Event }

func (t *recordingTransport) Configure(sentry.ClientOptions)        {}
func (t *recordingTransport) SendEvent(e *sentry.Event)             { t.events = append(t.events, e) }
func (t *recordingTransport) Flush(time.Duration) bool              { return true }
func (t *recordingTransport) FlushWithContext(context.Context) bool { return true }
func (t *recordingTransport) Close()                                {}

// A panic flowing through the worker guard must be reported to Sentry AND
// re-raised, so the process still crashes with the original stack and non-zero exit.
func TestCapturePanicReportsAndRepanics(t *testing.T) {
	tr := &recordingTransport{}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:       "https://public@o0.ingest.sentry.io/0",
		Transport: tr,
	}); err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("capturePanic swallowed the panic; want it re-raised")
		}
		if len(tr.events) != 1 {
			t.Fatalf("captured %d events, want exactly 1", len(tr.events))
		}
	}()

	func() {
		defer capturePanic()
		panic("boom")
	}()
}
