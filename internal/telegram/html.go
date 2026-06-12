package telegram

import (
	"html"
	"strings"
)

// TextToHTML converts an extracted plain-text description into minimal safe HTML:
// blank-line-separated chunks become paragraphs, single newlines become <br>, and
// everything is entity-escaped so post content can never inject markup. Stored
// descriptions are HTML across all sources (the SPA renders them directly), so
// telegram jobs must match that contract.
func TextToHTML(text string) string {
	var b strings.Builder
	for _, para := range strings.Split(text, "\n\n") {
		lines := strings.Split(para, "\n")
		var kept []string
		for _, l := range lines {
			if s := strings.TrimSpace(l); s != "" {
				kept = append(kept, html.EscapeString(s))
			}
		}
		if len(kept) == 0 {
			continue
		}
		b.WriteString("<p>")
		b.WriteString(strings.Join(kept, "<br>"))
		b.WriteString("</p>")
	}
	return b.String()
}
