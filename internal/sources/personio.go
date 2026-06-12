package sources

import (
	"context"
	"fmt"
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
		jobs = append(jobs, Job{
			ExternalID:  pos.ID,
			URL:         fmt.Sprintf("%s/job/%s", host, pos.ID),
			Title:       pos.Name,
			Company:     e.Company,
			Location:    pos.Office,
			Description: sanitizeHTML(body),
			Remote:      isRemote(pos.Office), // the feed has no remote flag
			PostedAt:    parseRFC3339(pos.CreatedAt),
		})
	}
	return jobs, nil
}
