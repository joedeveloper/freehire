package sources

import (
	"context"
	"strings"
	"testing"
)

func TestPersonioProvider(t *testing.T) {
	if got := NewPersonio(nil).Provider(); got != "personio" {
		t.Errorf("Provider() = %q, want %q", got, "personio")
	}
}

func TestPersonioFetch(t *testing.T) {
	fake := &fakeHTTP{body: `<?xml version="1.0" encoding="UTF-8"?>
<workzag-jobs>
  <position>
    <id>2596159</id>
    <subcompany>Acme Handwerk GmbH</subcompany>
    <office>Hamburg</office>
    <department>Engineering</department>
    <recruitingCategory>Tech</recruitingCategory>
    <name>Senior Go Engineer (m/w/d)</name>
    <jobDescriptions>
      <jobDescription><name>Tasks</name><value><![CDATA[<p>Build the pipeline.</p>]]></value></jobDescription>
      <jobDescription><name>Requirements</name><value><![CDATA[<ul><li>Go</li></ul>]]></value></jobDescription>
    </jobDescriptions>
    <createdAt>2026-04-09T20:28:25+00:00</createdAt>
  </position>
  <position>
    <id>2596160</id>
    <office>Remote</office>
    <name>Platform Engineer</name>
    <jobDescriptions>
      <jobDescription><name>Tasks</name><value><![CDATA[<p>Work from anywhere.</p>]]></value></jobDescription>
    </jobDescriptions>
    <createdAt>2026-04-09T20:28:25+00:00</createdAt>
  </position>
</workzag-jobs>`}

	jobs, err := NewPersonio(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Acme", Provider: "personio", Board: "acme",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(fake.gotURL, "acme.jobs.personio.com") || !strings.Contains(fake.gotURL, "/xml") {
		t.Errorf("requested URL %q should target the board xml feed", fake.gotURL)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}

	j := jobs[0]
	if j.ExternalID != "2596159" {
		t.Errorf("ExternalID = %q, want the position id", j.ExternalID)
	}
	if j.Title != "Senior Go Engineer (m/w/d)" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Location != "Hamburg" {
		t.Errorf("Location = %q, want the office", j.Location)
	}
	// The feed has no posting URL — the adapter builds one from board + id.
	if j.URL != "https://acme.jobs.personio.com/job/2596159" {
		t.Errorf("URL = %q, want a constructed job URL", j.URL)
	}
	// Description concatenates the jobDescription values, sanitized.
	for _, want := range []string{"Build the pipeline.", "Go"} {
		if !strings.Contains(j.Description, want) {
			t.Errorf("Description missing %q, got %q", want, j.Description)
		}
	}
	if j.Remote {
		t.Error("Remote = true for a Hamburg office, want false")
	}
	if j.PostedAt == nil || j.PostedAt.UTC().Year() != 2026 {
		t.Errorf("PostedAt = %v, want parsed createdAt (2026)", j.PostedAt)
	}

	// The feed carries no remote flag — remote is inferred from the office text.
	if !jobs[1].Remote {
		t.Error("second job office=Remote should infer Remote = true")
	}
}

func TestPersonioFetchEmptyFeed(t *testing.T) {
	fake := &fakeHTTP{body: `<?xml version="1.0" encoding="UTF-8"?><workzag-jobs></workzag-jobs>`}
	jobs, err := NewPersonio(fake).Fetch(context.Background(), CompanyEntry{
		Company: "Acme", Provider: "personio", Board: "acme",
	})
	if err != nil {
		t.Fatalf("Fetch on empty feed should not error: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("len(jobs) = %d, want 0 for an empty feed", len(jobs))
	}
}
