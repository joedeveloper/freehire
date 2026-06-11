package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/strelov1/freehire/internal/sources"
)

// fakeStore records every saved job. It is safe for the runner's concurrent Save calls.
type fakeStore struct {
	mu    sync.Mutex
	saved []Job
	err   error
}

func (s *fakeStore) Save(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, job)
	return nil
}

// fakeSource returns canned jobs or an error, keyed by provider.
type fakeSource struct {
	provider string
	jobs     []sources.Job
	err      error
}

func (f fakeSource) Provider() string { return f.provider }

func (f fakeSource) Fetch(context.Context, sources.CompanyEntry) ([]sources.Job, error) {
	return f.jobs, f.err
}

func registry(srcs ...sources.Source) map[string]sources.Source {
	m := make(map[string]sources.Source)
	for _, s := range srcs {
		m[s.Provider()] = s
	}
	return m
}

func TestRunNormalizesAndNamespaces(t *testing.T) {
	src := fakeSource{provider: "greenhouse", jobs: []sources.Job{
		{ExternalID: "42", Title: "Senior Go Developer", Company: "Acme Inc", URL: "u", Location: "Remote", Remote: true},
	}}
	store := &fakeStore{}
	r := Runner{Registry: registry(src), Store: store}

	stats, err := r.Run(context.Background(), []sources.CompanyEntry{
		{Company: "Acme Inc", Provider: "greenhouse", Board: "acme"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Ingested != 1 || stats.Failed != 0 {
		t.Fatalf("stats = %+v, want Ingested=1 Failed=0", stats)
	}
	if len(store.saved) != 1 {
		t.Fatalf("len(saved) = %d, want 1", len(store.saved))
	}

	j := store.saved[0]
	if j.Source != "greenhouse" {
		t.Errorf("Source = %q, want %q", j.Source, "greenhouse")
	}
	if j.ExternalID != "acme:42" {
		t.Errorf("ExternalID = %q, want %q (board-namespaced)", j.ExternalID, "acme:42")
	}
	if j.CompanySlug != "acme-inc" {
		t.Errorf("CompanySlug = %q, want %q", j.CompanySlug, "acme-inc")
	}
	if j.Title != "Senior Go Developer" || j.URL != "u" || !j.Remote {
		t.Errorf("passthrough fields wrong: %+v", j)
	}
}

func TestRunIsolatesSourceFailure(t *testing.T) {
	good := fakeSource{provider: "greenhouse", jobs: []sources.Job{{ExternalID: "1", Title: "ok"}}}
	bad := fakeSource{provider: "lever", err: errors.New("boom")}
	store := &fakeStore{}
	r := Runner{Registry: registry(good, bad), Store: store}

	stats, err := r.Run(context.Background(), []sources.CompanyEntry{
		{Company: "Good", Provider: "greenhouse", Board: "good"},
		{Company: "Bad", Provider: "lever", Board: "bad"},
	})
	if err != nil {
		t.Fatalf("Run should not return an error when a single source fails: %v", err)
	}
	if stats.Failed != 1 {
		t.Errorf("stats.Failed = %d, want 1", stats.Failed)
	}
	if stats.Ingested != 1 {
		t.Errorf("stats.Ingested = %d, want 1 (the healthy board)", stats.Ingested)
	}
	if len(store.saved) != 1 || store.saved[0].Source != "greenhouse" {
		t.Errorf("only the healthy board's job should be saved, got %+v", store.saved)
	}
}

func TestRunSkipsWorkOnCancelledContext(t *testing.T) {
	src := fakeSource{provider: "greenhouse", jobs: []sources.Job{{ExternalID: "1", Title: "x"}}}
	store := &fakeStore{}
	r := Runner{Registry: registry(src), Store: store}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before Run

	stats, err := r.Run(ctx, []sources.CompanyEntry{
		{Company: "C", Provider: "greenhouse", Board: "c"},
	})
	if err == nil {
		t.Fatal("Run should return the context error on a cancelled context")
	}
	if stats.Ingested != 0 {
		t.Errorf("stats.Ingested = %d, want 0 (no work on a cancelled context)", stats.Ingested)
	}
	if len(store.saved) != 0 {
		t.Errorf("saved %d jobs, want 0 on a cancelled context", len(store.saved))
	}
}

func TestRunIsolatesPerJobSaveError(t *testing.T) {
	src := fakeSource{provider: "greenhouse", jobs: []sources.Job{{ExternalID: "1", Title: "x"}}}
	store := &fakeStore{err: errors.New("write failed")}
	r := Runner{Registry: registry(src), Store: store}

	stats, err := r.Run(context.Background(), []sources.CompanyEntry{
		{Company: "C", Provider: "greenhouse", Board: "c"},
	})
	if err != nil {
		t.Fatalf("Run: a per-job save error must not fail the run: %v", err)
	}
	// A save error is skipped: the job is not counted ingested, but the board did not fail.
	if stats.Ingested != 0 {
		t.Errorf("stats.Ingested = %d, want 0 (save failed)", stats.Ingested)
	}
	if stats.Failed != 0 {
		t.Errorf("stats.Failed = %d, want 0 (a save error is not a board failure)", stats.Failed)
	}
}

func TestRunCountsUnknownProviderAsFailed(t *testing.T) {
	store := &fakeStore{}
	r := Runner{Registry: registry(), Store: store}

	stats, err := r.Run(context.Background(), []sources.CompanyEntry{
		{Company: "X", Provider: "myspace", Board: "x"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Failed != 1 || stats.Ingested != 0 {
		t.Errorf("stats = %+v, want Failed=1 Ingested=0", stats)
	}
}
