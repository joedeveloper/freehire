package sources

import (
	"context"
	"fmt"
	"strings"
)

// leverBaseURL is the Lever postings API root.
const leverBaseURL = "https://api.lever.co/v0/postings"

// lever adapts the Lever postings API. The JSON-mode endpoint returns a bare array of
// postings whose body is split across HTML description/lists/additional fields, which
// the adapter assembles, so no per-posting detail request is needed.
type lever struct {
	http JSONGetter
}

// NewLever builds the Lever adapter over the given HTTP client.
func NewLever(c JSONGetter) Source { return lever{http: c} }

func (lever) Provider() string { return "lever" }

func (l lever) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	url := fmt.Sprintf("%s/%s?mode=json", leverBaseURL, e.Board)

	var postings []struct {
		ID            string `json:"id"`
		Text          string `json:"text"`
		HostedURL     string `json:"hostedUrl"`
		CreatedAt     int64  `json:"createdAt"`
		Description   string `json:"description"`
		Additional    string `json:"additional"`
		WorkplaceType string `json:"workplaceType"`
		Lists         []struct {
			Text    string `json:"text"`
			Content string `json:"content"`
		} `json:"lists"`
		Categories struct {
			Location string `json:"location"`
		} `json:"categories"`
	}
	if err := l.http.GetJSON(ctx, url, &postings); err != nil {
		return nil, fmt.Errorf("lever: fetch board %s: %w", e.Board, err)
	}

	jobs := make([]Job, 0, len(postings))
	for _, p := range postings {
		// Lever splits the body across description + lists (each a heading and its
		// HTML items) + additional; the plain mirror is unreliable, so assemble the
		// HTML fields into one document.
		var body strings.Builder
		body.WriteString(p.Description)
		for _, list := range p.Lists {
			if list.Text != "" {
				body.WriteString("<h3>")
				body.WriteString(list.Text)
				body.WriteString("</h3>")
			}
			body.WriteString(list.Content)
		}
		body.WriteString(p.Additional)

		jobs = append(jobs, Job{
			ExternalID:  p.ID,
			URL:         p.HostedURL,
			Title:       p.Text,
			Company:     e.Company,
			Location:    p.Categories.Location,
			Description: sanitizeHTML(body.String()),
			Remote:      isRemote(p.Categories.Location),
			WorkMode:    workplaceTypeMode(p.WorkplaceType),
			PostedAt:    parseEpochMillis(p.CreatedAt),
		})
	}
	return jobs, nil
}
