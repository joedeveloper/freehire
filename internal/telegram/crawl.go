package telegram

import (
	"context"
	"log"
	"time"
)

// PostFetcher reads a channel's latest posts (implemented by Fetcher; tests fake it).
type PostFetcher interface {
	Fetch(ctx context.Context, channel string) ([]Post, error)
}

// PostStore persists crawled posts. Insert stores a post once — done marks it
// already-processed (the prefilter found no vacancy) — and reports whether the
// post was new; an already-stored post is left untouched.
type PostStore interface {
	Insert(ctx context.Context, channel string, p Post, done bool) (bool, error)
}

// CrawlStats summarizes one crawl run.
type CrawlStats struct {
	Stored   int // new posts queued for extraction
	Filtered int // new posts recorded as non-vacancies
	Failed   int // channels whose fetch or store failed
}

// CrawlRunner crawls every configured channel once. Channels are fetched
// sequentially with a polite delay — the volume is small and t.me rate limits
// are the binding constraint, not throughput.
type CrawlRunner struct {
	Fetcher PostFetcher
	Store   PostStore
	Delay   time.Duration // pause between channels; zero in tests
}

// Run crawls the channels, isolating failures: a channel whose fetch or store
// errors is counted failed and the run continues.
func (r CrawlRunner) Run(ctx context.Context, channels []ChannelEntry) (CrawlStats, error) {
	var stats CrawlStats
	for i, ch := range channels {
		if i > 0 && r.Delay > 0 {
			select {
			case <-ctx.Done():
				return stats, ctx.Err()
			case <-time.After(r.Delay):
			}
		}

		posts, err := r.Fetcher.Fetch(ctx, ch.Channel)
		if err != nil {
			log.Printf("telegram: channel %s failed: %v", ch.Channel, err)
			stats.Failed++
			continue
		}

		failed := false
		for _, p := range posts {
			isVacancy := LooksLikeVacancy(p.Text)
			inserted, err := r.Store.Insert(ctx, ch.Channel, p, !isVacancy)
			if err != nil {
				log.Printf("telegram: store %s/%d failed: %v", ch.Channel, p.MsgID, err)
				failed = true
				break
			}
			if !inserted {
				continue // already stored by an earlier crawl
			}
			if isVacancy {
				stats.Stored++
			} else {
				stats.Filtered++
			}
		}
		if failed {
			stats.Failed++
		}
	}
	return stats, nil
}
