package search

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

// JobView is the public shape of a job in the search index — identical to the
// job wire shape served by the list and detail endpoints, so a single job
// representation is used across the whole API and the SPA renders search hits
// with the same components. It deliberately has no internal id (carried only by
// the enclosing JobDocument), so a search response built from JobView cannot
// leak it. Jobs are identified by PublicSlug.
//
// Enrichment is kept nested (not flattened): the document mirrors the API job,
// and Meilisearch filters/sorts on the nested facets via dot paths (e.g.
// "enrichment.seniority", "enrichment.salary_min"). Timestamps are RFC3339 UTC
// strings — the same format the JSON API emits, and lexicographically sortable
// in chronological order.
type JobView struct {
	PublicSlug        string            `json:"public_slug"`
	Source            string            `json:"source"`
	ExternalID        string            `json:"external_id"`
	URL               string            `json:"url"`
	Title             string            `json:"title"`
	Company           string            `json:"company"`
	CompanySlug       string            `json:"company_slug"`
	Location          string            `json:"location"`
	Remote            bool              `json:"remote"`
	Description       string            `json:"description"`
	PostedAt          *string           `json:"posted_at"`
	CreatedAt         *string           `json:"created_at"`
	UpdatedAt         *string           `json:"updated_at"`
	Enrichment        enrich.Enrichment `json:"enrichment"`
	EnrichedAt        *string           `json:"enriched_at"`
	EnrichmentVersion int32             `json:"enrichment_version"`
}

// JobDocument is a job as stored in the Meilisearch index: the internal id (the
// primary key) plus the public JobView. The embedded view flattens into the
// document JSON, so the stored document is `{ "id": ..., "public_slug": ..., ... }`
// and Meilisearch reads "id" as the primary key. The id is never returned to
// clients — handlers respond with the JobView alone.
type JobDocument struct {
	ID int64 `json:"id"`
	JobView
}

// FromJob maps a database job row to its index document. The enrichment JSONB is
// decoded into the typed Enrichment; an empty or absent payload yields the zero
// Enrichment (the job is still fully searchable by its text).
func FromJob(j db.Job) (JobDocument, error) {
	var e enrich.Enrichment
	if len(j.Enrichment) > 0 {
		if err := json.Unmarshal(j.Enrichment, &e); err != nil {
			return JobDocument{}, fmt.Errorf("search: decode enrichment for job %d: %w", j.ID, err)
		}
	}

	return JobDocument{
		ID: j.ID,
		JobView: JobView{
			PublicSlug:        j.PublicSlug,
			Source:            j.Source,
			ExternalID:        j.ExternalID,
			URL:               j.URL,
			Title:             j.Title,
			Company:           j.Company,
			CompanySlug:       j.CompanySlug,
			Location:          j.Location,
			Remote:            j.Remote,
			Description:       j.Description,
			PostedAt:          rfc3339(j.PostedAt),
			CreatedAt:         rfc3339(j.CreatedAt),
			UpdatedAt:         rfc3339(j.UpdatedAt),
			Enrichment:        e,
			EnrichedAt:        rfc3339(j.EnrichedAt),
			EnrichmentVersion: j.EnrichmentVersion,
		},
	}, nil
}

// rfc3339 renders a nullable Postgres timestamp as an RFC3339 UTC string, or nil
// when unset. UTC keeps the lexicographic order chronological for sorting.
func rfc3339(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}
