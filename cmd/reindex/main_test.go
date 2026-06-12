package main

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
)

// The reindex feed deliberately includes closed jobs: open ones are upserted as
// documents, closed ones become deletions so they leave the index (job-search
// spec: the index contains only open jobs).
func TestSplitJobs_OpenBecomeDocsClosedBecomeDeletions(t *testing.T) {
	open := db.Job{ID: 1, Title: "Open", PublicSlug: "open-x"}
	closed := db.Job{ID: 2, Title: "Closed", PublicSlug: "closed-x",
		ClosedAt: pgtype.Timestamptz{Time: open.CreatedAt.Time, Valid: true}}

	docs, deleteIDs, err := splitJobs([]db.Job{open, closed})
	if err != nil {
		t.Fatalf("splitJobs: %v", err)
	}
	if len(docs) != 1 || docs[0].ID != 1 {
		t.Fatalf("docs = %+v, want only the open job", docs)
	}
	if len(deleteIDs) != 1 || deleteIDs[0] != 2 {
		t.Fatalf("deleteIDs = %v, want [2]", deleteIDs)
	}
}
