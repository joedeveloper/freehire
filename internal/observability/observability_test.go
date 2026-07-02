package observability

import "testing"

// Without a DSN the integration must stay dormant: Init reports no error and
// returns a callable no-op flush, so an unconfigured deployment runs unchanged.
func TestInitDisabledWithoutDSN(t *testing.T) {
	flush, err := Init("", "test")
	if err != nil {
		t.Fatalf("Init(\"\") returned error: %v", err)
	}
	if flush == nil {
		t.Fatal("Init(\"\") returned nil flush; want a no-op flush")
	}
	flush() // must not panic
}

// A malformed DSN is a misconfiguration the caller should fail fast on, so Init
// surfaces the parse error and returns no flush.
func TestInitRejectsMalformedDSN(t *testing.T) {
	flush, err := Init("not-a-valid-dsn", "test")
	if err == nil {
		t.Fatal("Init with a malformed DSN returned nil error; want an error")
	}
	if flush != nil {
		t.Fatal("Init error path should return a nil flush")
	}
}
