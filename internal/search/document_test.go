package search

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

func ptr[T any](v T) *T { return &v }

func TestFromJob_MapsCoreAndNestedEnrichment(t *testing.T) {
	posted := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, err := json.Marshal(enrich.Enrichment{
		Seniority:      "senior",
		Category:       "backend",
		Domains:        []string{"fintech"},
		VisaSponsorship: ptr(true),
		SalaryMin:      ptr(100000),
		Skills:         []string{"go", "postgres"},
	})
	if err != nil {
		t.Fatalf("marshal enrichment: %v", err)
	}

	doc, err := FromJob(db.Job{
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
		t.Fatalf("FromJob: %v", err)
	}

	if doc.ID != 42 {
		t.Errorf("ID = %d, want 42", doc.ID)
	}
	if doc.PublicSlug != "senior-go-developer-acme-abcd1234" {
		t.Errorf("PublicSlug = %q", doc.PublicSlug)
	}
	if doc.Title != "Senior Go Developer" || doc.Company != "Acme" || doc.Source != "manual" {
		t.Errorf("core fields not mapped: %+v", doc.JobView)
	}
	if doc.PostedAt == nil || *doc.PostedAt != "2025-01-02T03:04:05Z" {
		t.Errorf("PostedAt = %v, want RFC3339 UTC", doc.PostedAt)
	}
	// Enrichment stays nested, matching the list/detail wire shape.
	if doc.Enrichment.Seniority != "senior" || doc.Enrichment.Category != "backend" {
		t.Errorf("nested enrichment not mapped: %+v", doc.Enrichment)
	}
	if doc.Enrichment.SalaryMin == nil || *doc.Enrichment.SalaryMin != 100000 {
		t.Errorf("nested salary_min = %v", doc.Enrichment.SalaryMin)
	}
	if len(doc.Enrichment.Skills) != 2 || doc.Enrichment.VisaSponsorship == nil || !*doc.Enrichment.VisaSponsorship {
		t.Errorf("nested skills/visa not mapped: %+v", doc.Enrichment)
	}
}

func TestFromJob_UnenrichedHasZeroEnrichment(t *testing.T) {
	doc, err := FromJob(db.Job{
		ID:         1,
		Title:      "Go Developer",
		Company:    "Acme",
		PublicSlug: "go-developer-acme-x",
		Enrichment: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("FromJob: %v", err)
	}

	if doc.Title != "Go Developer" {
		t.Errorf("Title = %q, want mapped", doc.Title)
	}
	if doc.Enrichment.Seniority != "" || doc.Enrichment.SalaryMin != nil || len(doc.Enrichment.Skills) != 0 {
		t.Errorf("expected zero enrichment, got %+v", doc.Enrichment)
	}
	if doc.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil for an unset timestamp", doc.PostedAt)
	}
}

func TestFromJob_NilEnrichmentByteSliceIsSafe(t *testing.T) {
	// A job whose enrichment column round-trips as a nil/empty byte slice must
	// not fail decoding — it is simply unenriched.
	doc, err := FromJob(db.Job{ID: 2, Title: "x", PublicSlug: "x-2", Enrichment: nil})
	if err != nil {
		t.Fatalf("FromJob with nil enrichment: %v", err)
	}
	if doc.Enrichment.Seniority != "" {
		t.Errorf("expected no enrichment, got seniority %q", doc.Enrichment.Seniority)
	}
}

func TestJobDocument_FlattensIDAndViewToTopLevelJSON(t *testing.T) {
	// Meilisearch reads the primary key "id" from the top level of the document,
	// and the embedded JobView must flatten (no nesting) so its fields are the
	// searchable attributes. A json tag on JobView would break this. Enrichment
	// itself stays a nested object (filtered via dot paths).
	doc := JobDocument{ID: 7, JobView: JobView{PublicSlug: "x-7", Title: "x"}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["id"]; !ok {
		t.Errorf("missing top-level id in %s", raw)
	}
	if _, ok := m["public_slug"]; !ok {
		t.Errorf("public_slug not flattened to top level in %s", raw)
	}
	if _, ok := m["enrichment"]; !ok {
		t.Errorf("enrichment should be a nested object in %s", raw)
	}
}
