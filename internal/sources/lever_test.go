package sources

import (
	"context"
	"strings"
	"testing"
)

func TestLeverProvider(t *testing.T) {
	if got := NewLever(nil).Provider(); got != "lever" {
		t.Errorf("Provider() = %q, want %q", got, "lever")
	}
}

func TestLeverFetch(t *testing.T) {
	fake := &fakeHTTP{body: `[
		{
			"id": "abc-123",
			"text": "Backend Engineer",
			"hostedUrl": "https://jobs.lever.co/lever/abc-123",
			"createdAt": 1705312800000,
			"categories": {"location": "Remote"},
			"descriptionPlain": "Write Go."
		}
	]`}

	jobs, err := NewLever(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Lever", Provider: "lever", Board: "lever",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if !strings.Contains(fake.gotURL, "lever") {
		t.Errorf("requested URL %q should target the board", fake.gotURL)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}

	j := jobs[0]
	if j.ExternalID != "abc-123" {
		t.Errorf("ExternalID = %q, want %q", j.ExternalID, "abc-123")
	}
	if j.Title != "Backend Engineer" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.URL != "https://jobs.lever.co/lever/abc-123" {
		t.Errorf("URL = %q", j.URL)
	}
	if j.Company != "Lever" {
		t.Errorf("Company = %q, want the configured company", j.Company)
	}
	if j.Location != "Remote" {
		t.Errorf("Location = %q", j.Location)
	}
	if j.Description != "Write Go." {
		t.Errorf("Description = %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true")
	}
	if j.PostedAt == nil {
		t.Fatal("PostedAt = nil, want parsed createdAt")
	}
	if got := j.PostedAt.UTC().Year(); got != 2024 {
		t.Errorf("PostedAt year = %d, want 2024", got)
	}
}
