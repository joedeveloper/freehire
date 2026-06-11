package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/db"
)

// companyDetailResponse is the public shape of a company together with a page of
// its jobs. Its Jobs field is []jobResponse, not []db.Job, so the internal job
// id cannot leak through this endpoint — the type enforces the DTO mapping.
type companyDetailResponse struct {
	Company db.Company    `json:"company"`
	Jobs    []jobResponse `json:"jobs"`
}

func newCompanyDetailResponse(company db.Company, jobs []db.Job) companyDetailResponse {
	return companyDetailResponse{Company: company, Jobs: toJobResponses(jobs)}
}

// ListCompanies returns a page of companies with their job counts. Counts are
// computed at query time; there is no denormalized counter yet.
func (h *Handler) ListCompanies(c *fiber.Ctx) error {
	limit, offset := pageParams(c)

	companies, err := h.queries.ListCompanies(c.Context(), db.ListCompaniesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list companies")
	}

	total, err := h.queries.CountCompanies(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count companies")
	}

	return c.JSON(fiber.Map{
		"data": companies,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetCompany returns a single company together with a page of its jobs. The
// company is read from companies and its jobs from a single-table filter on
// company_slug — no join between the two tables.
func (h *Handler) GetCompany(c *fiber.Ctx) error {
	slug := c.Params("slug")

	company, err := h.queries.GetCompany(c.Context(), slug)
	if err != nil {
		// ErrorHandler maps pgx.ErrNoRows to 404, anything else to 500.
		return err
	}

	limit, offset := pageParams(c)

	jobs, err := h.queries.ListJobsByCompany(c.Context(), db.ListJobsByCompanyParams{
		CompanySlug: slug,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list company jobs")
	}

	return c.JSON(fiber.Map{"data": newCompanyDetailResponse(company, jobs)})
}
