package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
	"github.com/strelov1/freehire/internal/location"
	"github.com/strelov1/freehire/internal/normalize"
	"github.com/strelov1/freehire/internal/telegram"
)

// maxAttempts is the retry budget per post: the first failure leaves the post
// retryable (after its lease expires), the second dead-letters it.
const maxAttempts = 2

// extractStore adapts the generated queries + pool to telegram.ExtractStore.
// Complete writes every extracted job through the canonical UpsertJob (which
// upserts the company and gates on the dedup key) plus the enrichment enqueue,
// and marks the post extracted — all in one transaction, so a crash never
// half-persists a post.
type extractStore struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func newExtractStore(pool *pgxpool.Pool) *extractStore {
	return &extractStore{pool: pool, q: db.New(pool)}
}

func (s *extractStore) Claim(ctx context.Context, leaseSeconds, batchSize int32) ([]telegram.PendingPost, error) {
	rows, err := s.q.ClaimTelegramPosts(ctx, db.ClaimTelegramPostsParams{
		LeaseSeconds: leaseSeconds,
		BatchSize:    batchSize,
	})
	if err != nil {
		return nil, err
	}
	posts := make([]telegram.PendingPost, len(rows))
	for i, r := range rows {
		posts[i] = telegram.PendingPost{
			Channel:  r.Channel,
			MsgID:    r.MsgID,
			Text:     r.Text,
			PostedAt: r.PostedAt.Time,
		}
	}
	return posts, nil
}

func (s *extractStore) Complete(ctx context.Context, post telegram.PendingPost, jobs []telegram.ExtractedJob) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	base := post.Channel + "/" + strconv.FormatInt(post.MsgID, 10)
	for i, j := range jobs {
		externalID := base + "/" + strconv.Itoa(i)
		geo := location.Parse(j.Location)
		saved, err := qtx.UpsertJob(ctx, db.UpsertJobParams{
			Source:      "telegram",
			ExternalID:  externalID,
			URL:         "https://t.me/" + base,
			Title:       j.Title,
			Company:     j.Company,
			CompanySlug: normalize.Slug(j.Company),
			PublicSlug:  normalize.JobSlug(j.Title, j.Company, "telegram", externalID),
			Location:    j.Location,
			Remote:      j.Remote,
			Description: telegram.TextToHTML(j.Description),
			PostedAt:    pgtype.Timestamptz{Time: post.PostedAt, Valid: true},
			Countries:   geo.Countries,
			Regions:     geo.Regions,
			WorkMode:    geo.WorkMode,
		})
		if err != nil {
			return fmt.Errorf("upsert job %s: %w", externalID, err)
		}
		if _, err := qtx.EnqueueJobEnrichment(ctx, db.EnqueueJobEnrichmentParams{
			TargetVersion: int32(enrich.Version),
			JobID:         saved.ID,
		}); err != nil {
			return fmt.Errorf("enqueue enrichment %s: %w", externalID, err)
		}
	}

	if err := qtx.MarkTelegramPostExtracted(ctx, db.MarkTelegramPostExtractedParams{
		Channel: post.Channel,
		MsgID:   post.MsgID,
	}); err != nil {
		return fmt.Errorf("mark extracted %s: %w", base, err)
	}
	return tx.Commit(ctx)
}

func (s *extractStore) Fail(ctx context.Context, post telegram.PendingPost, errMsg string) error {
	_, err := s.q.RecordTelegramPostFailure(ctx, db.RecordTelegramPostFailureParams{
		LastError:   errMsg,
		MaxAttempts: maxAttempts,
		Channel:     post.Channel,
		MsgID:       post.MsgID,
	})
	return err
}

var _ telegram.ExtractStore = (*extractStore)(nil)
