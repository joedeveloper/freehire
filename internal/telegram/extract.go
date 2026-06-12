package telegram

import (
	"context"
	"log"
	"time"
)

// PendingPost is a claimed telegram_posts row awaiting extraction.
type PendingPost struct {
	Channel  string
	MsgID    int64
	Text     string
	PostedAt time.Time
}

// Extractor classifies a post and extracts its vacancies via an LLM. The kind
// steers the prompt (board: expect one vacancy; authored: expect 0..N). The
// result is not trusted — the runner validates before persisting.
type Extractor interface {
	Extract(ctx context.Context, text string, kind Kind) (Extraction, error)
}

// ExtractStore is the persistence boundary of the extraction worker. Complete
// writes the extracted jobs through the canonical job upsert and marks the post
// extracted in one transaction; Fail counts a failed attempt (dead-lettering at
// the attempt cap is the store's concern).
type ExtractStore interface {
	Claim(ctx context.Context, leaseSeconds, batchSize int32) ([]PendingPost, error)
	Complete(ctx context.Context, post PendingPost, jobs []ExtractedJob) error
	Fail(ctx context.Context, post PendingPost, errMsg string) error
}

// ExtractStats summarizes one extraction run.
type ExtractStats struct {
	Processed int // posts completed (jobs written or none found)
	Jobs      int // vacancies written
	Failed    int // posts whose extraction failed this run
}

// Extraction queue tuning. The lease must outlive the slowest plausible LLM
// call; its expiry doubles as the crash reaper (see the enrichment runner).
const (
	leaseSeconds = 600
	batchSize    = 50
)

// ExtractRunner drains one batch of pending posts: claim, extract, validate,
// persist. A post whose payload is invalid or whose LLM call fails is failed —
// the store retries it once (on a later run, after the lease expires) and then
// dead-letters it; an invalid payload is never persisted.
type ExtractRunner struct {
	Extractor Extractor
	Store     ExtractStore
	Kinds     map[string]Kind // channel → kind, from channels.yml
}

// Run processes one claimed batch and returns its stats.
func (r ExtractRunner) Run(ctx context.Context) (ExtractStats, error) {
	var stats ExtractStats

	posts, err := r.Store.Claim(ctx, leaseSeconds, batchSize)
	if err != nil {
		return stats, err
	}

	for _, post := range posts {
		extraction, err := r.Extractor.Extract(ctx, post.Text, r.kind(post.Channel))
		if err == nil {
			err = extraction.Validate()
		}
		if err != nil {
			log.Printf("telegram: extract %s/%d failed: %v", post.Channel, post.MsgID, err)
			stats.Failed++
			if ferr := r.Store.Fail(ctx, post, err.Error()); ferr != nil {
				return stats, ferr
			}
			continue
		}

		if err := r.Store.Complete(ctx, post, extraction.Jobs); err != nil {
			return stats, err
		}
		stats.Processed++
		stats.Jobs += len(extraction.Jobs)
	}
	return stats, nil
}

// kind resolves a channel's configured kind, defaulting to board for a post
// whose channel has since left channels.yml (the safer, single-vacancy prompt).
func (r ExtractRunner) kind(channel string) Kind {
	if k, ok := r.Kinds[channel]; ok {
		return k
	}
	return KindBoard
}
