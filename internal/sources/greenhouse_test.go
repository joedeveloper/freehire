package sources

import (
	"context"
	"strings"
	"testing"
)

func TestGreenhouseProvider(t *testing.T) {
	if got := NewGreenhouse(nil).Provider(); got != "greenhouse" {
		t.Errorf("Provider() = %q, want %q", got, "greenhouse")
	}
}

func TestGreenhouseFetch(t *testing.T) {
	fake := &fakeHTTP{body: `{
		"jobs": [
			{
				"id": 123,
				"title": "Senior Go Developer",
				"absolute_url": "https://boards.greenhouse.io/gitlab/jobs/123",
				"updated_at": "2024-01-15T10:00:00Z",
				"location": {"name": "Remote - US"},
				"content": "<p>Build things</p>"
			}
		]
	}`}

	jobs, err := NewGreenhouse(fake).Fetch(context.Background(), CompanyEntry{
		Company: "GitLab", Provider: "greenhouse", Board: "gitlab",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if !strings.Contains(fake.gotURL, "gitlab") || !strings.Contains(fake.gotURL, "content=true") {
		t.Errorf("requested URL %q should target the board with content=true", fake.gotURL)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}

	j := jobs[0]
	if j.ExternalID != "123" {
		t.Errorf("ExternalID = %q, want %q", j.ExternalID, "123")
	}
	if j.Title != "Senior Go Developer" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.URL != "https://boards.greenhouse.io/gitlab/jobs/123" {
		t.Errorf("URL = %q", j.URL)
	}
	if j.Company != "GitLab" {
		t.Errorf("Company = %q, want the configured company", j.Company)
	}
	if j.Location != "Remote - US" {
		t.Errorf("Location = %q", j.Location)
	}
	if j.Description != "<p>Build things</p>" {
		t.Errorf("Description = %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true for a Remote location")
	}
	if j.PostedAt == nil {
		t.Error("PostedAt = nil, want parsed updated_at")
	}
}
