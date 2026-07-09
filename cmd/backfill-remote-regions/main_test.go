package main

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/strelov1/freehire/internal/db"
)

// fakeStore records every SetCompanyRemoteRegions call and returns a per-slug
// affected-row count, so the loader's matched/unmatched tally can be exercised
// without a database.
type fakeStore struct {
	calls    []db.SetCompanyRemoteRegionsParams
	affected map[string]int64
}

func (f *fakeStore) SetCompanyRemoteRegions(_ context.Context, p db.SetCompanyRemoteRegionsParams) (int64, error) {
	f.calls = append(f.calls, p)
	return f.affected[p.Slug], nil
}

func TestLoad(t *testing.T) {
	const csv = `Name,Website,Region
Acme,https://acme.com,Europe
Ghost,,USA
Foo,,Atlantis
,https://none.com,Worldwide
`
	fs := &fakeStore{affected: map[string]int64{"acme": 1, "ghost": 0, "foo": 1}}
	stats, err := load(context.Background(), fs, strings.NewReader(csv))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if stats.matched != 2 || stats.unmatched != 1 {
		t.Errorf("matched=%d unmatched=%d, want 2 / 1", stats.matched, stats.unmatched)
	}
	if stats.mapped != 2 || stats.unmapped != 1 {
		t.Errorf("mapped=%d unmapped=%d, want 2 / 1", stats.mapped, stats.unmapped)
	}
	if stats.skipped != 1 {
		t.Errorf("skipped=%d, want 1 (blank-name row)", stats.skipped)
	}
	if len(fs.calls) != 3 {
		t.Fatalf("SetCompanyRemoteRegions called %d times, want 3 (blank-name row skipped)", len(fs.calls))
	}

	// Acme: normalized slug, mapped regions, raw source preserved.
	acme := fs.calls[0]
	if acme.Slug != "acme" {
		t.Errorf("slug = %q, want normalized \"acme\"", acme.Slug)
	}
	if !reflect.DeepEqual(acme.RemoteRegions, []string{"eu"}) {
		t.Errorf("remote_regions = %v, want [eu]", acme.RemoteRegions)
	}
	if acme.RemoteRegionsRaw != "Europe" {
		t.Errorf("remote_regions_raw = %q, want %q", acme.RemoteRegionsRaw, "Europe")
	}

	// Foo: unmappable region → empty, but a non-nil slice for the NOT NULL column.
	foo := fs.calls[2]
	if foo.RemoteRegions == nil || len(foo.RemoteRegions) != 0 {
		t.Errorf("remote_regions = %v, want non-nil empty slice", foo.RemoteRegions)
	}
	if foo.RemoteRegionsRaw != "Atlantis" {
		t.Errorf("remote_regions_raw = %q, want %q", foo.RemoteRegionsRaw, "Atlantis")
	}
}

// TestCheckedInDatasetLoadsAndFullyMaps guards the checked-in dataset against the
// dictionary: every row parses (none skipped for a blank name) and every region
// string resolves to at least one macro region (none unmapped). A transcription
// typo or a new region string the dictionary does not cover trips this.
func TestCheckedInDatasetLoadsAndFullyMaps(t *testing.T) {
	f, err := os.Open("../../sources/remote-companies.csv")
	if err != nil {
		t.Fatalf("open dataset: %v", err)
	}
	defer f.Close()

	// A store that matches nothing, so we exercise only parsing + mapping.
	fs := &fakeStore{affected: map[string]int64{}}
	stats, err := load(context.Background(), fs, f)
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}
	if stats.skipped != 0 {
		t.Errorf("skipped=%d, want 0 (every dataset row has a usable name)", stats.skipped)
	}
	if stats.unmapped != 0 {
		t.Errorf("unmapped=%d, want 0 (every region string must resolve; run the dictionary against the new strings)", stats.unmapped)
	}
	if stats.mapped == 0 {
		t.Fatal("mapped=0, dataset appears empty")
	}
	if got := stats.matched + stats.unmatched; got != len(fs.calls) {
		t.Errorf("matched+unmatched=%d but %d rows processed", got, len(fs.calls))
	}
}
