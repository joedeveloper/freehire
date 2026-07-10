package main

import (
	"context"
	"strings"
	"testing"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobhash"
)

// fakeStore serves keyset pages of jobs (across all sources) and records every description
// update. It pages by id so the backfill's keyset loop terminates.
type fakeStore struct {
	jobs    []db.Job
	updates []db.UpdateJobDescriptionParams
}

func (f *fakeStore) ListJobsByIDAfter(_ context.Context, arg db.ListJobsByIDAfterParams) ([]db.Job, error) {
	var page []db.Job
	for _, j := range f.jobs {
		if j.ID > arg.AfterID {
			page = append(page, j)
			if int32(len(page)) == arg.BatchSize {
				break
			}
		}
	}
	return page, nil
}

func (f *fakeStore) UpdateJobDescription(_ context.Context, arg db.UpdateJobDescriptionParams) (int64, error) {
	f.updates = append(f.updates, arg)
	return 1, nil
}

// TestBackfillDecodesOnlyEncodedDescriptions asserts the backfill rewrites only the rows whose
// stored description is still percent-encoded (marker "%3C"), re-decodes them in place, and
// leaves clean rows untouched — across sources, open and closed alike.
func TestBackfillDecodesOnlyEncodedDescriptions(t *testing.T) {
	store := &fakeStore{jobs: []db.Job{
		// taleo, encoded HTML with a literal "%" that strict PathUnescape choked on.
		{ID: 1, Source: "taleo", Title: "A", Description: `%3Cp style=%22line-height%5C:115%;%22%3EWrite Go. 100% remote. C++%3C/p%3E`},
		// already clean — must be skipped.
		{ID: 2, Source: "greenhouse", Title: "B", Description: "<p>Clean HTML.</p>"},
		// a different source that somehow stored encoded HTML — the marker is source-agnostic.
		{ID: 3, Source: "icims", Title: "C", Description: `%3Cp%3EHello%3C%2Fp%3E`},
	}}

	scanned, updated, err := backfillAll(context.Background(), store)
	if err != nil {
		t.Fatalf("backfillAll: %v", err)
	}
	if scanned != 3 || updated != 2 {
		t.Fatalf("scanned=%d updated=%d, want 3/2", scanned, updated)
	}

	if len(store.updates) != 2 {
		t.Fatalf("updates = %d, want 2 (jobs 1 and 3)", len(store.updates))
	}
	got := map[int64]db.UpdateJobDescriptionParams{}
	for _, u := range store.updates {
		got[u.ID] = u
	}

	u1, ok := got[1]
	if !ok {
		t.Fatal("job 1 not updated")
	}
	if strings.Contains(u1.Description, "%3C") || strings.Contains(u1.Description, "%22") {
		t.Errorf("job 1 still encoded: %q", u1.Description)
	}
	for _, want := range []string{"Write Go.", "100% remote", "C++"} {
		if !strings.Contains(u1.Description, want) {
			t.Errorf("job 1 missing %q: %q", want, u1.Description)
		}
	}
	// content_hash must fingerprint the row with the decoded description, matching what a
	// re-ingest of the fixed adapter would produce.
	want1 := jobhash.Of(hashParams(store.jobs[0], u1.Description))
	if !u1.ContentHash.Valid || u1.ContentHash.String != want1 {
		t.Errorf("job 1 ContentHash = %+v, want %q", u1.ContentHash, want1)
	}

	if _, ok := got[2]; ok {
		t.Error("clean job 2 must not be updated")
	}
	if _, ok := got[3]; !ok {
		t.Error("encoded job 3 (icims) must be updated")
	}
}
