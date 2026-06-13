package sources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// sfDetailWorkers caps how many per-job detail page requests a single board issues
// concurrently.
const sfDetailWorkers = 8

// successfactors adapts SAP SuccessFactors career sites. The board is the career-site
// host (e.g. "jobs.tetrapak.com"). The site's job sitemap enumerates the postings; each
// job page is server-rendered HTML carrying schema.org JobPosting microdata, so the
// description comes from a per-job detail fetch (bounded-concurrency), like the other
// detail-fetching adapters.
type successfactors struct {
	http HTTPClient
}

// NewSuccessFactors builds the SuccessFactors adapter over the given HTTP client.
func NewSuccessFactors(c HTTPClient) Source { return successfactors{http: c} }

func (successfactors) Provider() string { return "successfactors" }

// sfSitemapEntry is one <url> of the job sitemap: the job page URL and its last-modified
// date (used as posted_at).
type sfSitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

func (s successfactors) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	var sitemap struct {
		URLs []sfSitemapEntry `xml:"url"`
	}
	url := fmt.Sprintf("https://%s/job_sitemap.xml", e.Board)
	if err := s.http.GetXML(ctx, url, &sitemap); err != nil {
		return nil, fmt.Errorf("successfactors: sitemap %s: %w", e.Board, err)
	}

	// Each job's title and description come from its own page fetch, fanned out under a
	// bounded worker pool.
	return fetchDetails(sitemap.URLs, sfDetailWorkers, func(entry sfSitemapEntry) (Job, bool) {
		return s.detail(ctx, e, entry)
	}), nil
}

// detail fetches one job page and maps it to a Job, returning ok=false when the page
// fetch fails so the caller can skip just that posting.
func (s successfactors) detail(ctx context.Context, e CompanyEntry, entry sfSitemapEntry) (Job, bool) {
	id := sfJobID(entry.Loc)
	if id == "" {
		return Job{}, false // no native id → would collide on the dedup key; skip it
	}

	root, err := s.http.GetHTML(ctx, entry.Loc)
	if err != nil {
		return Job{}, false
	}

	title := itempropText(root, "title")
	if title == "" {
		title = metaProperty(root, "og:title")
	}

	return Job{
		ExternalID: id,
		URL:        entry.Loc,
		Title:      title,
		Company:    e.Company,
		// Location is intentionally empty: SuccessFactors does not expose it in the
		// microdata, and enrichment derives it from the description.
		Location:    "",
		Description: sanitizeHTML(itempropHTML(root, "description")),
		Remote:      isRemote(title),
		PostedAt:    parseDate(entry.LastMod),
	}, true
}

// sfJobIDPattern captures the leading digits of a job URL's last path segment, ignoring a
// trailing locale suffix (e.g. ".../98012-en_GB" → "98012", ".../12345/" → "12345").
var sfJobIDPattern = regexp.MustCompile(`/(\d+)(?:-[^/]*)?/?$`)

// sfJobID extracts the native numeric posting id from a job page URL.
func sfJobID(loc string) string {
	if m := sfJobIDPattern.FindStringSubmatch(loc); m != nil {
		return m[1]
	}
	return ""
}

// itempropText returns the concatenated text of the first element carrying the given
// schema.org itemprop, or "" when none is present.
func itempropText(root *html.Node, prop string) string {
	ns := findItemprops(root, prop)
	if len(ns) == 0 {
		return ""
	}
	return textContent(ns[0])
}

// itempropHTML returns the rendered inner HTML of the richest element carrying the given
// itemprop, so it can be sanitized, or "" when none is present. SuccessFactors wraps
// several near-empty itemprop="description" layout regions around the real body, so the
// element with the most text content is chosen rather than the first.
func itempropHTML(root *html.Node, prop string) string {
	ns := findItemprops(root, prop)
	if len(ns) == 0 {
		return ""
	}
	best, bestLen := ns[0], len(textContent(ns[0]))
	for _, n := range ns[1:] {
		if l := len(textContent(n)); l > bestLen {
			best, bestLen = n, l
		}
	}
	return innerHTML(best)
}

// metaProperty returns the content of <meta property="..."> (e.g. og:title), or "".
func metaProperty(root *html.Node, property string) string {
	var found string
	walk(root, func(n *html.Node) bool {
		if found != "" {
			return false
		}
		if n.Type == html.ElementNode && n.Data == "meta" &&
			attr(n, "property") == property {
			found = attr(n, "content")
			return false
		}
		return true
	})
	return found
}

// findItemprops returns every element node whose itemprop attribute equals prop, in
// document order.
func findItemprops(root *html.Node, prop string) []*html.Node {
	var out []*html.Node
	walk(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && attr(n, "itemprop") == prop {
			out = append(out, n)
		}
		return true
	})
	return out
}

// walk visits nodes depth-first, descending into a node's children only while visit
// returns true for it.
func walk(n *html.Node, visit func(*html.Node) bool) {
	if !visit(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}

// attr returns the value of the named attribute, or "".
func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// textContent returns the concatenated text of n's descendants, trimmed.
func textContent(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
		return true
	})
	return strings.TrimSpace(b.String())
}

// innerHTML renders n's children back to HTML markup.
func innerHTML(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&b, c)
	}
	return b.String()
}
