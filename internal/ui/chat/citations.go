package chat

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Citation is one web/search source link for compact UI.
type Citation struct {
	Title string
	URL   string
}

var (
	mdLinkRE  = regexp.MustCompile(`\[[^\]]*\]\((https?://[^)\s]+)\)`)
	bareURLRE = regexp.MustCompile(`https?://[^\s)\]>]+`)
)

// CitationsForMessage extracts citation-like links from persistence raw JSON
// and message body. Persistence events do not define a dedicated citations
// field in generated types; this is best-effort on url/title-shaped JSON.
func CitationsForMessage(msg session.Message) []Citation {
	var out []Citation
	if len(msg.Raw) > 0 {
		out = append(out, extractCitationsFromJSON(msg.Raw)...)
	}
	if msg.Body != "" {
		out = append(out, extractCitationsFromMarkdown(msg.Body)...)
	}
	return dedupeCitations(out, 8)
}

func extractCitationsFromMarkdown(body string) []Citation {
	var out []Citation
	for _, m := range mdLinkRE.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			out = append(out, Citation{URL: strings.TrimSpace(m[1])})
		}
	}
	for _, u := range bareURLRE.FindAllString(body, -1) {
		out = append(out, Citation{URL: strings.TrimSpace(u)})
	}
	return out
}

func extractCitationsFromJSON(raw json.RawMessage) []Citation {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	var out []Citation
	walkCitationJSON(v, &out)
	return out
}

func walkCitationJSON(v any, out *[]Citation) {
	switch t := v.(type) {
	case map[string]any:
		if cites, ok := t["citations"]; ok {
			walkCitationJSON(cites, out)
		}
		if src, ok := t["sources"]; ok {
			walkCitationJSON(src, out)
		}
		if results, ok := t["results"]; ok {
			walkCitationJSON(results, out)
		}
		linkURL := firstHTTPString(t, "url", "href", "link", "uri")
		if linkURL != "" {
			title := firstString(t, "title", "name", "siteName", "site", "snippet")
			*out = append(*out, Citation{Title: title, URL: linkURL})
		}
		for _, child := range t {
			walkCitationJSON(child, out)
		}
	case []any:
		for _, el := range t {
			walkCitationJSON(el, out)
		}
	}
}

func firstHTTPString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			s = strings.TrimSpace(s)
			if isHTTPLike(s) {
				return s
			}
		}
	}
	return ""
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func isHTTPLike(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}

func dedupeCitations(in []Citation, max int) []Citation {
	if max <= 0 {
		max = 8
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]Citation, 0, len(in))
	for _, c := range in {
		u := strings.TrimSpace(c.URL)
		if u == "" || !isHTTPLike(u) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		title := strings.TrimSpace(c.Title)
		if title == "" {
			title = shortenURL(u)
		}
		out = append(out, Citation{Title: title, URL: u})
		if len(out) >= max {
			break
		}
	}
	return out
}

func shortenURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := u.Host
	if host == "" {
		return raw
	}
	if u.Path != "" && u.Path != "/" {
		return host + u.Path
	}
	return host
}

// CitationsView renders a compact source list under a message or tool card.
func CitationsView(c *ui.Context, cites []Citation, muted, accent ui.Color, onOpen func(url string)) {
	if len(cites) == 0 {
		return
	}
	ui.Column(c).Gap(2).Padding(4, 0, 0, 0).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatCitations)).FontSize(10).Bold().TextColor(muted)
		for _, cite := range cites {
			entry := cite
			label := entry.Title
			if len(label) > 72 {
				label = label[:72] + "…"
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if onOpen != nil {
					if ui.Button(c, label).FontSize(10).Clicked() {
						onOpen(entry.URL)
					}
				} else {
					ui.Text(c, "• "+label).FontSize(10).TextColor(accent).Grow(1)
				}
				if strings.TrimSpace(entry.URL) != "" {
					if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(10).Clicked() {
						mygo.Clipboard.WriteText(entry.URL)
					}
				}
			})
		}
	})
}
