package sources

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

// ldJobPosting decodes the first application/ld+json JobPosting block on the page into v,
// returning false when the page carries no such block. Shared by the HTML detail adapters
// (teamtailor, breezy) whose job pages server-render a schema.org JobPosting; each passes
// a struct selecting just the fields it needs.
func ldJobPosting(root *html.Node, v any) bool {
	found := false
	walk(root, func(n *html.Node) bool {
		if found {
			return false
		}
		if n.Type == html.ElementNode && n.Data == "script" &&
			attr(n, "type") == "application/ld+json" {
			raw := []byte(textContent(n))
			var probe struct {
				Type string `json:"@type"`
			}
			if json.Unmarshal(raw, &probe) == nil &&
				strings.EqualFold(probe.Type, "JobPosting") &&
				json.Unmarshal(raw, v) == nil {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
