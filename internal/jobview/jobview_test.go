package jobview

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

func ptr[T any](v T) *T { return &v }

func TestFromRow_MapsCoreAndNestedEnrichment(t *testing.T) {
	posted := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, err := json.Marshal(enrich.Enrichment{
		Seniority:       "senior",
		Category:        "backend",
		Domains:         []string{"fintech"},
		VisaSponsorship: ptr(true),
		SalaryMin:       ptr(100000),
		Skills:          []string{"go", "postgres"},
	})
	if err != nil {
		t.Fatalf("marshal enrichment: %v", err)
	}

	view, err := FromRow(db.Job{
		ID:          42,
		Source:      "manual",
		ExternalID:  "ext-1",
		Title:       "Senior Go Developer",
		Company:     "Acme",
		CompanySlug: "acme",
		Location:    "Berlin",
		Remote:      true,
		Description: "Build durable systems",
		PostedAt:    pgtype.Timestamptz{Time: posted, Valid: true},
		PublicSlug:  "senior-go-developer-acme-abcd1234",
		Enrichment:  raw,
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}

	if view.PublicSlug != "senior-go-developer-acme-abcd1234" {
		t.Errorf("PublicSlug = %q", view.PublicSlug)
	}
	if view.Title != "Senior Go Developer" || view.Company != "Acme" || view.Source != "manual" {
		t.Errorf("core fields not mapped: %+v", view)
	}
	if view.PostedAt == nil || *view.PostedAt != "2025-01-02T03:04:05Z" {
		t.Errorf("PostedAt = %v, want RFC3339 UTC", view.PostedAt)
	}
	// Enrichment stays nested and typed.
	if view.Enrichment.Seniority != "senior" || view.Enrichment.Category != "backend" {
		t.Errorf("nested enrichment not mapped: %+v", view.Enrichment)
	}
	if view.Enrichment.SalaryMin == nil || *view.Enrichment.SalaryMin != 100000 {
		t.Errorf("nested salary_min = %v", view.Enrichment.SalaryMin)
	}
	if len(view.Enrichment.Skills) != 2 || view.Enrichment.VisaSponsorship == nil || !*view.Enrichment.VisaSponsorship {
		t.Errorf("nested skills/visa not mapped: %+v", view.Enrichment)
	}
}

// Geography is the union of the parsed-location columns and the enrichment-derived
// values; work_mode is the LLM value when present, else the parsed one. All three
// are served top-level and folded out of the enrichment object.
func TestFromRow_MergesGeographyAndWorkMode(t *testing.T) {
	raw, err := json.Marshal(enrich.Enrichment{
		Regions:   []string{"emea"},
		Countries: []string{"us"},
		WorkMode:  "remote",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	view, err := FromRow(db.Job{
		ID:         1,
		Title:      "Dev",
		PublicSlug: "dev-1",
		Countries:  []string{"us"}, // parsed from location
		Regions:    []string{"us"}, // parsed from location
		WorkMode:   "onsite",       // parsed from location (loses to the LLM)
		Enrichment: raw,
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}

	if got := view.Countries; len(got) != 1 || got[0] != "us" {
		t.Errorf("Countries = %v, want [us] (deduped union)", got)
	}
	if got := view.Regions; len(got) != 2 || got[0] != "emea" || got[1] != "us" {
		t.Errorf("Regions = %v, want [emea us] (sorted union)", got)
	}
	if view.WorkMode != "remote" {
		t.Errorf("WorkMode = %q, want remote (LLM wins over parsed onsite)", view.WorkMode)
	}
	// Folded out of enrichment so geography/work_mode are reported once.
	if len(view.Enrichment.Regions) != 0 || len(view.Enrichment.Countries) != 0 || view.Enrichment.WorkMode != "" {
		t.Errorf("enrichment still carries folded fields: %+v", view.Enrichment)
	}
}

// The LLM emits ISO country codes uppercase ("DE"); the parser emits them
// lowercase ("de"). The merge must case-fold to one canonical form so the facet
// is not split into duplicate buckets and a lowercase filter matches both.
func TestFromRow_CountryCaseIsFolded(t *testing.T) {
	raw, err := json.Marshal(enrich.Enrichment{Countries: []string{"DE", "FR"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	view, err := FromRow(db.Job{
		ID:         1,
		Title:      "x",
		PublicSlug: "x-1",
		Countries:  []string{"de"}, // parser, lowercase
		Enrichment: raw,
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	want := []string{"de", "fr"}
	if len(view.Countries) != 2 || view.Countries[0] != want[0] || view.Countries[1] != want[1] {
		t.Errorf("Countries = %v, want %v (case-folded, deduped)", view.Countries, want)
	}
}

func TestFromRow_WorkModeFallsBackToParsed(t *testing.T) {
	// No enrichment: the parsed-location work_mode surfaces.
	view, err := FromRow(db.Job{ID: 1, Title: "x", PublicSlug: "x-1", WorkMode: "hybrid"})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	if view.WorkMode != "hybrid" {
		t.Errorf("WorkMode = %q, want hybrid (parser fallback when no LLM value)", view.WorkMode)
	}
}

func TestFromRow_EmptyEnrichmentIsZero(t *testing.T) {
	// An unenriched job's column arrives as "{}" (the table default) or, in
	// edge cases, a nil byte slice. Both must decode to the zero Enrichment,
	// never fail.
	for _, payload := range [][]byte{[]byte("{}"), nil} {
		view, err := FromRow(db.Job{ID: 1, Title: "x", PublicSlug: "x-1", Enrichment: payload})
		if err != nil {
			t.Fatalf("FromRow with enrichment %q: %v", payload, err)
		}
		if view.Enrichment.Seniority != "" || view.Enrichment.SalaryMin != nil || len(view.Enrichment.Skills) != 0 {
			t.Errorf("enrichment %q: expected zero enrichment, got %+v", payload, view.Enrichment)
		}
	}
}

// Job's JSON encoding IS the API contract for every jobs read endpoint. These
// tests lock two requirements: the internal numeric id is never exposed and the
// public slug is (specs/job-public-identity), and the enrichment payload
// survives the mapping (specs/job-enrichment): an unenriched job serializes
// enrichment as {} (not null), and an enriched payload keeps its fields.

func TestJobJSON_HidesIDExposesSlug(t *testing.T) {
	fields := marshalToFields(t, db.Job{
		ID:         123,
		Title:      "Go Developer",
		PublicSlug: "go-developer-acme-t35nijto",
	})

	if _, leaked := fields["id"]; leaked {
		t.Error("wire shape leaks the internal numeric id")
	}
	if got := string(fields["public_slug"]); got != `"go-developer-acme-t35nijto"` {
		t.Errorf("public_slug: want the slug, got %s", got)
	}
}

// The raw remote flag is demoted to an internal enrichment hint and must not
// appear in the public job object — "remote" is expressed solely through
// enrichment.work_mode / regions.
func TestJobJSON_OmitsRawRemoteFlag(t *testing.T) {
	fields := marshalToFields(t, db.Job{ID: 1, Title: "x", PublicSlug: "x-1", Remote: true})

	if _, present := fields["remote"]; present {
		t.Error("public job object must not include the raw remote flag")
	}
}

// Un-enriched job: enrichment is {} (not null), enriched_at is null,
// enrichment_version is 0.
func TestJobJSON_Unenriched(t *testing.T) {
	fields := marshalToFields(t, db.Job{ID: 1, Title: "Go Developer"})

	if got := string(fields["enrichment"]); got != "{}" {
		t.Errorf("enrichment: want {}, got %s", got)
	}
	if got := string(fields["posted_at"]); got != "null" {
		t.Errorf("posted_at: want null for an unset timestamp, got %s", got)
	}
	if got := string(fields["enriched_at"]); got != "null" {
		t.Errorf("enriched_at: want null, got %s", got)
	}
	if got := string(fields["enrichment_version"]); got != "0" {
		t.Errorf("enrichment_version: want 0, got %s", got)
	}
}

// Enriched job: the JSONB payload survives the typed decode/encode round-trip,
// enriched_at is the RFC3339 UTC timestamp, version is set.
func TestJobJSON_Enriched(t *testing.T) {
	enrichedAt := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	fields := marshalToFields(t, db.Job{
		ID:                2,
		Title:             "Senior Go Developer",
		Enrichment:        json.RawMessage(`{"seniority":"senior","work_mode":"remote"}`),
		EnrichedAt:        pgtype.Timestamptz{Time: enrichedAt, Valid: true},
		EnrichmentVersion: 1,
	})

	var enrichment map[string]any
	if err := json.Unmarshal(fields["enrichment"], &enrichment); err != nil {
		t.Fatalf("enrichment is not a JSON object: %v", err)
	}
	if enrichment["seniority"] != "senior" {
		t.Errorf("enrichment payload not preserved: %v", enrichment)
	}
	// work_mode is folded into the top-level facet, not duplicated under enrichment.
	if got := string(fields["work_mode"]); got != `"remote"` {
		t.Errorf("work_mode: want top-level \"remote\", got %s", got)
	}
	if _, dup := enrichment["work_mode"]; dup {
		t.Error("work_mode must not also appear under enrichment")
	}
	if got := string(fields["enriched_at"]); got != `"2026-06-09T12:00:00Z"` {
		t.Errorf("enriched_at: want the timestamp, got %s", got)
	}
	if got := string(fields["enrichment_version"]); got != "1" {
		t.Errorf("enrichment_version: want 1, got %s", got)
	}
}

// marshalToFields maps a db.Job through the wire shape and returns its
// top-level JSON fields — the actual public contract.
func marshalToFields(t *testing.T, job db.Job) map[string]json.RawMessage {
	t.Helper()
	view, err := FromRow(job)
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return fields
}

// closed_at rides the wire shape so the SPA can render the closed state on the
// detail page (lists never serve closed jobs — see the job-lifecycle spec).
func TestFromRow_CarriesClosedAt(t *testing.T) {
	closedAt := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	v, err := FromRow(db.Job{ClosedAt: pgtype.Timestamptz{Time: closedAt, Valid: true}})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	if v.ClosedAt == nil || *v.ClosedAt != "2026-06-12T10:00:00Z" {
		t.Fatalf("ClosedAt = %v, want 2026-06-12T10:00:00Z", v.ClosedAt)
	}

	open, err := FromRow(db.Job{})
	if err != nil {
		t.Fatalf("FromRow open: %v", err)
	}
	if open.ClosedAt != nil {
		t.Fatalf("open job ClosedAt = %v, want nil", *open.ClosedAt)
	}
}
