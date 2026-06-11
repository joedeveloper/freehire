package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGetJSONDecodesAndSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"acme"}`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), userAgent: "freehire-test"}

	var out struct {
		Name string `json:"name"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if out.Name != "acme" {
		t.Errorf("decoded name = %q, want %q", out.Name, "acme")
	}
	if gotUA != "freehire-test" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "freehire-test")
	}
}

func TestClientGetJSONRetriesOnServerError(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), maxRetries: 2}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if !out.OK {
		t.Error("expected ok=true after retry")
	}
}

func TestClientGetJSONErrorsOnClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client()}

	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL, &out); err == nil {
		t.Error("expected error on 404, got nil")
	}
}
