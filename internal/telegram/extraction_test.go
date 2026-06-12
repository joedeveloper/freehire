package telegram

import (
	"strings"
	"testing"
)

func validJob() ExtractedJob {
	return ExtractedJob{
		Title:       "Senior Go Engineer",
		Company:     "Acme",
		Location:    "London",
		Remote:      true,
		Description: "Building the thing. $150k. Apply via @acme_hr.",
	}
}

func TestExtractionValidate(t *testing.T) {
	t.Run("zero jobs is valid (post was not a vacancy)", func(t *testing.T) {
		if err := (Extraction{}).Validate(); err != nil {
			t.Errorf("empty extraction: %v, want nil", err)
		}
	})

	t.Run("a well-formed job passes", func(t *testing.T) {
		e := Extraction{Jobs: []ExtractedJob{validJob()}}
		if err := e.Validate(); err != nil {
			t.Errorf("valid: %v, want nil", err)
		}
	})

	t.Run("company and location may be empty", func(t *testing.T) {
		j := validJob()
		j.Company, j.Location = "", ""
		if err := (Extraction{Jobs: []ExtractedJob{j}}).Validate(); err != nil {
			t.Errorf("optional fields empty: %v, want nil", err)
		}
	})

	t.Run("empty title is rejected", func(t *testing.T) {
		j := validJob()
		j.Title = "  "
		err := (Extraction{Jobs: []ExtractedJob{j}}).Validate()
		if err == nil || !strings.Contains(err.Error(), "title") {
			t.Errorf("err = %v, want a title error", err)
		}
	})

	t.Run("empty description is rejected", func(t *testing.T) {
		j := validJob()
		j.Description = ""
		err := (Extraction{Jobs: []ExtractedJob{j}}).Validate()
		if err == nil || !strings.Contains(err.Error(), "description") {
			t.Errorf("err = %v, want a description error", err)
		}
	})
}
