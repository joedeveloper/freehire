package search

import (
	"encoding/json"
	"fmt"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

// JobDocument is the shape of a job as stored in the Meilisearch index and
// returned by a search. ID is the index primary key and is internal: it is
// never exposed by the public API (the search handler strips it and identifies
// jobs by PublicSlug, consistent with the rest of the public job reads).
//
// Types are deliberately plain JSON values (string/int64/bool/slices) rather
// than pgtype, so the index shape stays simple and self-describing. Enrichment
// is flattened into first-class facet fields here so Meilisearch can filter and
// sort on them directly; an unenriched job simply leaves them empty.
type JobDocument struct {
	ID          int64  `json:"id"`
	PublicSlug  string `json:"public_slug"`
	Source      string `json:"source"`
	ExternalID  string `json:"external_id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	CompanySlug string `json:"company_slug"`
	Location    string `json:"location"`
	Remote      bool   `json:"remote"`
	Description string `json:"description"`
	// PostedAt is unix seconds for a sortable attribute; nil when the job has no
	// posting date.
	PostedAt *int64 `json:"posted_at,omitempty"`

	// Enrichment facets (filterable / sortable). Absent on unenriched jobs.
	WorkMode           string   `json:"work_mode,omitempty"`
	EmploymentType     string   `json:"employment_type,omitempty"`
	Seniority          string   `json:"seniority,omitempty"`
	Category           string   `json:"category,omitempty"`
	Domains            []string `json:"domains,omitempty"`
	Countries          []string `json:"countries,omitempty"`
	CompanyType        string   `json:"company_type,omitempty"`
	CompanySize        string   `json:"company_size,omitempty"`
	VisaSponsorship    *bool    `json:"visa_sponsorship,omitempty"`
	SalaryCurrency     string   `json:"salary_currency,omitempty"`
	SalaryPeriod       string   `json:"salary_period,omitempty"`
	Skills             []string `json:"skills,omitempty"`
	SalaryMin          *int     `json:"salary_min,omitempty"`
	SalaryMax          *int     `json:"salary_max,omitempty"`
	ExperienceYearsMin *int     `json:"experience_years_min,omitempty"`
}

// FromJob maps a database job row to its index document. The enrichment JSONB is
// decoded into typed facet fields; an empty or absent payload yields a document
// with no facets (the job is still fully searchable by its text).
func FromJob(j db.Job) (JobDocument, error) {
	doc := JobDocument{
		ID:          j.ID,
		PublicSlug:  j.PublicSlug,
		Source:      j.Source,
		ExternalID:  j.ExternalID,
		URL:         j.URL,
		Title:       j.Title,
		Company:     j.Company,
		CompanySlug: j.CompanySlug,
		Location:    j.Location,
		Remote:      j.Remote,
		Description: j.Description,
	}

	if j.PostedAt.Valid {
		unix := j.PostedAt.Time.Unix()
		doc.PostedAt = &unix
	}

	// An empty/absent enrichment payload means "unenriched" — leave facets empty.
	if len(j.Enrichment) > 0 {
		var e enrich.Enrichment
		if err := json.Unmarshal(j.Enrichment, &e); err != nil {
			return JobDocument{}, fmt.Errorf("search: decode enrichment for job %d: %w", j.ID, err)
		}
		doc.WorkMode = e.WorkMode
		doc.EmploymentType = e.EmploymentType
		doc.Seniority = e.Seniority
		doc.Category = e.Category
		doc.Domains = e.Domains
		doc.Countries = e.Countries
		doc.CompanyType = e.CompanyType
		doc.CompanySize = e.CompanySize
		doc.VisaSponsorship = e.VisaSponsorship
		doc.SalaryCurrency = e.SalaryCurrency
		doc.SalaryPeriod = e.SalaryPeriod
		doc.Skills = e.Skills
		doc.SalaryMin = e.SalaryMin
		doc.SalaryMax = e.SalaryMax
		doc.ExperienceYearsMin = e.ExperienceYearsMin
	}

	return doc, nil
}
