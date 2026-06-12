package telegram

import (
	"fmt"
	"strings"
)

// ExtractedJob is one vacancy extracted from a Telegram post. Company and
// location are optional — many posts state neither. Salary, contacts, and other
// details stay inside Description: enrichment derives structure from it later,
// exactly as for ATS-ingested jobs.
type ExtractedJob struct {
	Title       string `json:"title"`
	Company     string `json:"company,omitempty"`
	Location    string `json:"location,omitempty"`
	Remote      bool   `json:"remote,omitempty"`
	Description string `json:"description"`
}

// Extraction is the typed result of classifying + extracting one post. Zero jobs
// is a normal outcome: the post was not a vacancy.
type Extraction struct {
	Jobs []ExtractedJob `json:"jobs"`
}

// Validate rejects a malformed extraction before anything is persisted: every
// job needs a title and a description. The LLM is not trusted to be correct —
// an invalid payload is retried and then dead-lettered by the runner.
func (e Extraction) Validate() error {
	for i, j := range e.Jobs {
		if strings.TrimSpace(j.Title) == "" {
			return fmt.Errorf("telegram: extracted job %d has empty title", i)
		}
		if strings.TrimSpace(j.Description) == "" {
			return fmt.Errorf("telegram: extracted job %d (%s) has empty description", i, j.Title)
		}
	}
	return nil
}
