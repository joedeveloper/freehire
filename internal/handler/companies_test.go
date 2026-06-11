package handler

import (
	"encoding/json"
	"testing"

	"github.com/strelov1/freehire/internal/db"
)

// GetCompany returns a company together with a page of its jobs. The jobs must
// go through the same public DTO as the jobs endpoints — the internal numeric id
// must not leak here either. A typed companyDetailResponse whose Jobs field is
// []jobResponse makes that a compile-time guarantee; this test locks the wire
// contract (no "id", a "public_slug" per job).
func TestCompanyDetailHidesJobID(t *testing.T) {
	resp := newCompanyDetailResponse(db.Company{Slug: "acme", Name: "Acme"}, []db.Job{
		{ID: 123, Title: "Go Developer", PublicSlug: "go-developer-acme-t35nijto"},
	})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body struct {
		Jobs []map[string]json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(body.Jobs))
	}
	if _, leaked := body.Jobs[0]["id"]; leaked {
		t.Error("company jobs leak the internal numeric id")
	}
	if got := string(body.Jobs[0]["public_slug"]); got != `"go-developer-acme-t35nijto"` {
		t.Errorf("public_slug: want the slug, got %s", got)
	}
}
