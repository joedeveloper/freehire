package sources

import (
	"context"
	"fmt"

	"golang.org/x/net/html"
)

// personio adapts the Personio public XML feed. Each board is its own subdomain and
// publishes every open position in one document, with the body inline across one or
// more jobDescription HTML values — so no per-posting detail request is needed. The
// feed carries no posting URL, so the adapter builds one from the board and position id.
type personio struct {
	http HTTPClient
}

// NewPersonio builds the Personio adapter over the given HTTP client.
func NewPersonio(c HTTPClient) Source { return personio{http: c} }

func (personio) Provider() string { return "personio" }

func (p personio) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	host := fmt.Sprintf("https://%s.jobs.personio.com", e.Board)

	var resp struct {
		Positions []struct {
			ID           string `xml:"id"`
			Office       string `xml:"office"`
			Name         string `xml:"name"`
			CreatedAt    string `xml:"createdAt"`
			Descriptions []struct {
				Value string `xml:"value"`
			} `xml:"jobDescriptions>jobDescription"`
		} `xml:"position"`
	}
	if err := p.http.GetXML(ctx, host+"/xml", &resp); err != nil {
		return nil, fmt.Errorf("personio: fetch board %s: %w", e.Board, err)
	}

	jobs := make([]Job, 0, len(resp.Positions))
	for _, pos := range resp.Positions {
		// Personio splits the body across one or more jobDescription HTML values.
		var body string
		for _, d := range pos.Descriptions {
			body += d.Value
		}
		url := fmt.Sprintf("%s/job/%s", host, pos.ID)
		description := sanitizeHTML(body)
		// The board's default-locale feed omits a posting's body when it is published in
		// another locale (jobDescriptions comes back empty); the detail page still
		// server-renders it as a schema.org JobPosting, so fall back to that.
		if description == "" {
			if d, ok := p.detailDescription(ctx, url); ok {
				description = d
			}
		}
		jobs = append(jobs, Job{
			ExternalID:  pos.ID,
			URL:         url,
			Title:       pos.Name,
			Company:     e.Company,
			Location:    pos.Office,
			Description: description,
			Remote:      isRemote(pos.Office), // the feed has no remote flag
			PostedAt:    parseRFC3339(pos.CreatedAt),
		})
	}
	return jobs, nil
}

// detailDescription fetches a posting's detail page and returns its schema.org JobPosting
// body, sanitized, with ok=false when the page fetch fails or carries no such block.
func (p personio) detailDescription(ctx context.Context, url string) (string, bool) {
	root, err := p.http.GetHTML(ctx, url)
	if err != nil {
		return "", false
	}
	var ld struct {
		Description string `json:"description"`
	}
	if !ldJobPosting(root, &ld) || ld.Description == "" {
		return "", false
	}
	return sanitizeHTML(html.UnescapeString(ld.Description)), true
}
