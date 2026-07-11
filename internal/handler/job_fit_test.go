package handler

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
)

func TestStampsFresh(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	ts := func(tm time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: tm, Valid: true} }
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

	row := db.GetUserJobAnalysisRow{CvUploadedAt: ts(now), JobContentHash: txt("hash-1"), Model: "model-a"}

	t.Run("all stamps match → fresh", func(t *testing.T) {
		if !stampsFresh(row, &now, txt("hash-1"), "model-a") {
			t.Error("want fresh when CV time, job hash, and model all match")
		}
	})
	t.Run("CV re-uploaded → stale", func(t *testing.T) {
		later := now.Add(time.Hour)
		if stampsFresh(row, &later, txt("hash-1"), "model-a") {
			t.Error("want stale when the CV upload time changed")
		}
	})
	t.Run("job re-ingested → stale", func(t *testing.T) {
		if stampsFresh(row, &now, txt("hash-2"), "model-a") {
			t.Error("want stale when the job content hash changed")
		}
	})
	t.Run("model upgraded → stale", func(t *testing.T) {
		if stampsFresh(row, &now, txt("hash-1"), "model-b") {
			t.Error("want stale when LLM_MODEL changed since the analysis")
		}
	})
	t.Run("missing live CV time → stale (cannot confirm)", func(t *testing.T) {
		if stampsFresh(row, nil, txt("hash-1"), "model-a") {
			t.Error("want stale when the live CV upload time is unknown")
		}
	})
	t.Run("both job hashes null → fresh (non-board job, never re-crawled)", func(t *testing.T) {
		nullHashRow := db.GetUserJobAnalysisRow{CvUploadedAt: ts(now), Model: "model-a"}
		if !stampsFresh(nullHashRow, &now, pgtype.Text{}, "model-a") {
			t.Error("want fresh when neither the stored nor the live job hash exists")
		}
	})
	t.Run("job gains a hash later → stale", func(t *testing.T) {
		nullHashRow := db.GetUserJobAnalysisRow{CvUploadedAt: ts(now), Model: "model-a"}
		if stampsFresh(nullHashRow, &now, txt("hash-1"), "model-a") {
			t.Error("want stale when the job acquired a content hash after the analysis")
		}
	})
}

func TestNewFitQuota(t *testing.T) {
	cases := []struct {
		used          int64
		wantRemaining int64
		wantExhausted bool
	}{
		{used: 0, wantRemaining: fitAnalysisLimit, wantExhausted: false},
		{used: fitAnalysisLimit - 1, wantRemaining: 1, wantExhausted: false},
		{used: fitAnalysisLimit, wantRemaining: 0, wantExhausted: true},
		{used: fitAnalysisLimit + 5, wantRemaining: 0, wantExhausted: true}, // remaining never negative
	}
	for _, tc := range cases {
		q := newFitQuota(tc.used)
		if q.Limit != fitAnalysisLimit {
			t.Errorf("used=%d: limit = %d, want %d", tc.used, q.Limit, fitAnalysisLimit)
		}
		if q.Used != tc.used {
			t.Errorf("used=%d: Used = %d, want %d", tc.used, q.Used, tc.used)
		}
		if q.Remaining != tc.wantRemaining {
			t.Errorf("used=%d: remaining = %d, want %d", tc.used, q.Remaining, tc.wantRemaining)
		}
		if q.exhausted() != tc.wantExhausted {
			t.Errorf("used=%d: exhausted = %v, want %v", tc.used, q.exhausted(), tc.wantExhausted)
		}
	}
}
