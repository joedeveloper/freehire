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

func TestFromJob_MapsCoreAndEnrichmentFacets(t *testing.T) {
	posted := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, err := json.Marshal(enrich.Enrichment{
		WorkMode:           "remote",
		EmploymentType:     "full_time",
		Seniority:          "senior",
		Category:           "backend",
		Domains:            []string{"fintech"},
		Countries:          []string{"DE"},
		CompanyType:        "product",
		CompanySize:        "51-200",
		VisaSponsorship:    ptr(true),
		SalaryCurrency:     "USD",
		SalaryPeriod:       "year",
		Skills:             []string{"go", "postgres"},
		SalaryMin:          ptr(100000),
		SalaryMax:          ptr(150000),
		ExperienceYearsMin: ptr(5),
	})
	if err != nil {
		t.Fatalf("marshal enrichment: %v", err)
	}

	doc, err := FromJob(db.Job{
		ID:          42,
		Source:      "manual",
		ExternalID:  "ext-1",
		URL:         "https://example.com/jobs/1",
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
	if doc.Title != "Senior Go Developer" || doc.Company != "Acme" || doc.Location != "Berlin" {
		t.Errorf("core text fields not mapped: %+v", doc)
	}
	if doc.Source != "manual" || doc.ExternalID != "ext-1" || doc.CompanySlug != "acme" {
		t.Errorf("source/external/company_slug not mapped: %+v", doc)
	}
	if !doc.Remote {
		t.Error("Remote = false, want true")
	}
	if doc.PostedAt == nil || *doc.PostedAt != posted.Unix() {
		t.Errorf("PostedAt = %v, want %d", doc.PostedAt, posted.Unix())
	}
	if doc.WorkMode != "remote" || doc.EmploymentType != "full_time" || doc.Seniority != "senior" || doc.Category != "backend" {
		t.Errorf("enum facets not mapped: %+v", doc)
	}
	if doc.CompanyType != "product" || doc.CompanySize != "51-200" || doc.SalaryCurrency != "USD" || doc.SalaryPeriod != "year" {
		t.Errorf("company/salary facets not mapped: %+v", doc)
	}
	if len(doc.Domains) != 1 || doc.Domains[0] != "fintech" || len(doc.Countries) != 1 || doc.Countries[0] != "DE" {
		t.Errorf("multi-value facets not mapped: %+v", doc)
	}
	if len(doc.Skills) != 2 {
		t.Errorf("Skills = %v, want 2", doc.Skills)
	}
	if doc.VisaSponsorship == nil || !*doc.VisaSponsorship {
		t.Errorf("VisaSponsorship = %v, want true", doc.VisaSponsorship)
	}
	if doc.SalaryMin == nil || *doc.SalaryMin != 100000 || doc.SalaryMax == nil || *doc.SalaryMax != 150000 {
		t.Errorf("salary range not mapped: min=%v max=%v", doc.SalaryMin, doc.SalaryMax)
	}
	if doc.ExperienceYearsMin == nil || *doc.ExperienceYearsMin != 5 {
		t.Errorf("ExperienceYearsMin = %v, want 5", doc.ExperienceYearsMin)
	}
}

func TestFromJob_UnenrichedHasNoFacets(t *testing.T) {
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
	if doc.Seniority != "" || doc.Category != "" || doc.WorkMode != "" {
		t.Errorf("expected empty enum facets, got %+v", doc)
	}
	if doc.SalaryMin != nil || doc.VisaSponsorship != nil || len(doc.Skills) != 0 {
		t.Errorf("expected absent optional facets, got %+v", doc)
	}
	if doc.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil for an unset timestamp", doc.PostedAt)
	}
}

func TestJobDocument_FlattensIDAndViewToTopLevelJSON(t *testing.T) {
	// Meilisearch reads the primary key "id" from the top level of the document,
	// and the embedded JobView must flatten (no nesting) so its fields are the
	// searchable/filterable attributes. A json tag on JobView would break this.
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
}

func TestFromJob_EmptyEnrichmentByteSliceIsSafe(t *testing.T) {
	// A job whose enrichment column round-trips as a nil/empty byte slice must
	// not fail decoding — it is simply unenriched.
	doc, err := FromJob(db.Job{ID: 2, Title: "x", PublicSlug: "x-2", Enrichment: nil})
	if err != nil {
		t.Fatalf("FromJob with nil enrichment: %v", err)
	}
	if doc.Seniority != "" {
		t.Errorf("expected no facets, got seniority %q", doc.Seniority)
	}
}
