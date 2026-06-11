package sources

import (
	"context"
	"fmt"
)

// leverBaseURL is the Lever postings API root.
const leverBaseURL = "https://api.lever.co/v0/postings"

// lever adapts the Lever postings API. The JSON-mode endpoint returns a bare array of
// postings carrying a plain-text description, so no per-posting detail request is needed.
type lever struct {
	http HTTPClient
}

// NewLever builds the Lever adapter over the given HTTP client.
func NewLever(c HTTPClient) Source { return lever{http: c} }

func (lever) Provider() string { return "lever" }

func (l lever) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	url := fmt.Sprintf("%s/%s?mode=json", leverBaseURL, e.Board)

	var postings []struct {
		ID               string `json:"id"`
		Text             string `json:"text"`
		HostedURL        string `json:"hostedUrl"`
		CreatedAt        int64  `json:"createdAt"`
		DescriptionPlain string `json:"descriptionPlain"`
		Categories       struct {
			Location string `json:"location"`
		} `json:"categories"`
	}
	if err := l.http.GetJSON(ctx, url, &postings); err != nil {
		return nil, fmt.Errorf("lever: fetch board %s: %w", e.Board, err)
	}

	jobs := make([]Job, 0, len(postings))
	for _, p := range postings {
		jobs = append(jobs, Job{
			ExternalID:  p.ID,
			URL:         p.HostedURL,
			Title:       p.Text,
			Company:     e.Company,
			Location:    p.Categories.Location,
			Description: p.DescriptionPlain,
			Remote:      isRemote(p.Categories.Location),
			PostedAt:    parseEpochMillis(p.CreatedAt),
		})
	}
	return jobs, nil
}
