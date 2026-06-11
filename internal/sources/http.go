package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPClient is the narrow transport an adapter needs: fetch a URL and decode its
// JSON body into v. Adapters depend on this interface so tests inject a fake and
// never touch the network; the real client is Client below.
type HTTPClient interface {
	GetJSON(ctx context.Context, url string, v any) error
}

// Client is the real HTTPClient: a timeout-bounded GET with a project User-Agent and
// a bounded retry-with-backoff on transient (5xx / network) failures. A 4xx is not
// retried — it will not recover on its own.
type Client struct {
	httpClient *http.Client
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewClient builds the default ingest HTTP client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		userAgent:  "freehire/0.1 (+https://freehire.dev)",
		maxRetries: 2,
		retryDelay: 500 * time.Millisecond,
	}
}

// GetJSON fetches url and decodes its JSON body into v, retrying transient failures
// up to maxRetries times with a fixed backoff between attempts.
func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && c.retryDelay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("sources: build request %s: %w", url, err)
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // network error — transient
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			err := json.NewDecoder(resp.Body).Decode(v)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("sources: decode %s: %w", url, err)
			}
			return nil
		case resp.StatusCode >= 500:
			resp.Body.Close()
			lastErr = fmt.Errorf("sources: GET %s: status %d", url, resp.StatusCode)
			continue // server error — transient
		default:
			resp.Body.Close()
			return fmt.Errorf("sources: GET %s: status %d", url, resp.StatusCode)
		}
	}
	return fmt.Errorf("sources: GET %s failed after %d attempts: %w", url, c.maxRetries+1, lastErr)
}
