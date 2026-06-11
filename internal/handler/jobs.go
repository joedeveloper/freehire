package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
)

// jobResponse is the public wire shape of a job. It carries the public_slug and
// deliberately omits the internal numeric id, which must never be exposed: the
// id is enumerable and its growth leaks inventory size and fill rate. All other
// source fields pass through unchanged, including the enrichment passthrough.
type jobResponse struct {
	Source            string             `json:"source"`
	ExternalID        string             `json:"external_id"`
	URL               string             `json:"url"`
	Title             string             `json:"title"`
	Company           string             `json:"company"`
	CompanySlug       string             `json:"company_slug"`
	Location          string             `json:"location"`
	Remote            bool               `json:"remote"`
	Description       string             `json:"description"`
	PostedAt          pgtype.Timestamptz `json:"posted_at"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	UpdatedAt         pgtype.Timestamptz `json:"updated_at"`
	PublicSlug        string             `json:"public_slug"`
	Enrichment        json.RawMessage    `json:"enrichment"`
	EnrichedAt        pgtype.Timestamptz `json:"enriched_at"`
	EnrichmentVersion int32              `json:"enrichment_version"`
}

func toJobResponse(j db.Job) jobResponse {
	return jobResponse{
		Source:            j.Source,
		ExternalID:        j.ExternalID,
		URL:               j.URL,
		Title:             j.Title,
		Company:           j.Company,
		CompanySlug:       j.CompanySlug,
		Location:          j.Location,
		Remote:            j.Remote,
		Description:       j.Description,
		PostedAt:          j.PostedAt,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
		PublicSlug:        j.PublicSlug,
		Enrichment:        j.Enrichment,
		EnrichedAt:        j.EnrichedAt,
		EnrichmentVersion: j.EnrichmentVersion,
	}
}

func toJobResponses(jobs []db.Job) []jobResponse {
	out := make([]jobResponse, len(jobs))
	for i, j := range jobs {
		out[i] = toJobResponse(j)
	}
	return out
}

// ListJobs returns a page of jobs using limit/offset pagination.
func (h *Handler) ListJobs(c *fiber.Ctx) error {
	limit, offset := pageParams(c)

	jobs, err := h.queries.ListJobs(c.Context(), db.ListJobsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list jobs")
	}

	total, err := h.queries.CountJobs(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count jobs")
	}

	return c.JSON(fiber.Map{
		"data": toJobResponses(jobs),
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetJob returns a single job addressed by its public slug.
func (h *Handler) GetJob(c *fiber.Ctx) error {
	job, err := h.queries.GetJobBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		// ErrorHandler maps pgx.ErrNoRows to 404, anything else to 500.
		return err
	}

	return c.JSON(fiber.Map{"data": toJobResponse(job)})
}
