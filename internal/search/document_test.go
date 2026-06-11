package search

import (
	"encoding/json"
	"testing"

	"github.com/strelov1/freehire/internal/db"
)

func TestFromJob_DocumentFlattensIDAndViewToTopLevelJSON(t *testing.T) {
	// Meilisearch reads the primary key "id" from the top level of the document,
	// and the embedded jobview.Job must flatten (no nesting) so its fields are
	// the searchable attributes. A json tag on the embedded field would break
	// this. Enrichment itself stays a nested object (filtered via dot paths).
	doc, err := FromJob(db.Job{ID: 42, Title: "Go Dev", PublicSlug: "go-dev-acme-x"})
	if err != nil {
		t.Fatalf("FromJob: %v", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := string(m["id"]); got != "42" {
		t.Errorf("top-level id = %s, want 42 in %s", got, raw)
	}
	if got := string(m["public_slug"]); got != `"go-dev-acme-x"` {
		t.Errorf("public_slug not flattened to top level: %s", raw)
	}
	if _, ok := m["enrichment"]; !ok {
		t.Errorf("enrichment should be a nested object in %s", raw)
	}
}
