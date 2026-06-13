package sources

import (
	"context"
	"fmt"
	"strconv"
)

// vk adapts VK's public careers API (team.vk.company), a single-company source with no
// per-tenant board id (boardless). The list API paginates by a next URL and carries no
// description, so each vacancy's body is scraped from its server-rendered job page (which
// exposes schema.org JobPosting microdata, like SuccessFactors), fanned out under a bounded
// worker pool.
type vk struct {
	http HTTPClient
}

const (
	vkListURL       = "https://team.vk.company/career/api/v2/vacancies/?limit=50&offset=0"
	vkVacancyURL    = "https://team.vk.company/vacancy/%d/"
	vkDetailWorkers = 8
)

// NewVK builds the VK adapter over the given HTTP client.
func NewVK(c HTTPClient) Source { return vk{http: c} }

func (vk) Provider() string { return "vk" }

// vk is single-company, so its config entries carry no board.
func (vk) boardless() {}

// vkResult is one vacancy from the list response (no description here). Remote is a flag the
// API sets directly, so no location heuristic is needed.
type vkResult struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Town  struct {
		Name string `json:"name"`
	} `json:"town"`
	Remote bool `json:"remote"`
}

func (a vk) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	results, err := a.list(ctx)
	if err != nil {
		return nil, err
	}

	return fetchDetails(results, vkDetailWorkers, func(r vkResult) (Job, bool) {
		return a.detail(ctx, e, r)
	}), nil
}

// list pages through every vacancy by following the response's next URL until it is null.
func (a vk) list(ctx context.Context) ([]vkResult, error) {
	var results []vkResult
	for url := vkListURL; url != ""; {
		var resp struct {
			Next    string     `json:"next"`
			Results []vkResult `json:"results"`
		}
		if err := a.http.GetJSON(ctx, url, &resp); err != nil {
			return nil, fmt.Errorf("vk: list: %w", err)
		}
		results = append(results, resp.Results...)
		url = resp.Next
	}
	return results, nil
}

// detail scrapes one vacancy's description from its job page and maps it to a Job, returning
// ok=false when the page fetch fails so the caller skips just that posting.
func (a vk) detail(ctx context.Context, e CompanyEntry, r vkResult) (Job, bool) {
	url := fmt.Sprintf(vkVacancyURL, r.ID)
	root, err := a.http.GetHTML(ctx, url)
	if err != nil {
		return Job{}, false
	}

	return Job{
		ExternalID:  strconv.Itoa(r.ID),
		URL:         url,
		Title:       r.Title,
		Company:     e.Company,
		Location:    r.Town.Name,
		Description: sanitizeHTML(itempropHTML(root, "description")),
		Remote:      r.Remote,
		PostedAt:    nil,
	}, true
}
