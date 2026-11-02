package chat

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Citation is one web/search source link for compact UI.
type Citation struct {
	Title       string
	URL         string
	Snippet     string
	PublishedAt string
}

var (
	mdLinkRE  = regexp.MustCompile(`\[[^\]]*\]\((https?://[^)\s]+)\)`)
	bareURLRE = regexp.MustCompile(`https?://[^\s)\]>]+`)
)

// CitationsForMessage prefers the typed web presentation metadata carried by
// tool/result.meta, then keeps a raw JSON/Markdown fallback for older or
// plugin-owned result shapes.
func CitationsForMessage(msg session.Message) []Citation {
	var out []Citation
	if msg.Web != nil {
		out = append(out, citationsFromWebResult(msg.Web)...)
	}
	if len(msg.Raw) > 0 {
		out = append(out, extractCitationsFromJSON(msg.Raw)...)
	}
	if msg.Body != "" {
		out = append(out, extractCitationsFromMarkdown(msg.Body)...)
	}
	return dedupeCitations(out, 8)
}

func citationsFromWebResult(result *session.WebResult) []Citation {
	if result == nil {
		return nil
	}
	if result.Kind == "fetch" {
		if strings.TrimSpace(result.URL) == "" {
			return nil
		}
		return []Citation{{Title: result.URL, URL: result.URL}}
	}
	if result.Kind != "search" {
		return nil
	}
	out := make([]Citation, 0, len(result.Sources))
	for _, source := range result.Sources {
		out = append(out, Citation{
			Title:       source.Title,
			URL:         source.URL,
			Snippet:     source.Snippet,
			PublishedAt: source.PublishedAt,
		})
	}
	return out
}

func extractCitationsFromMarkdown(body string) []Citation {
	if !strings.Contains(body, "http://") && !strings.Contains(body, "https://") {
		return nil
	}
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
	if !bytes.Contains(raw, []byte("http://")) && !bytes.Contains(raw, []byte("https://")) {
		return nil
	}
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
		out = append(out, Citation{Title: title, URL: u, Snippet: c.Snippet, PublishedAt: c.PublishedAt})
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
		ui.Text(c, i18n.T(i18n.KeyChatCitations)).FontSize(theme.InterfaceSize(10)).Bold().TextColor(muted)
		for _, cite := range cites {
			entry := cite
			label := entry.Title
			if len(label) > 72 {
				label = label[:72] + "…"
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if onOpen != nil {
					link := ui.Button(c, label).FontSize(theme.InterfaceSize(10)).BorderWidth(0).
						Background(ui.Transparent).TextColor(accent).Padding(0, 2, 0, 2)
					if link.Clicked() {
						onOpen(entry.URL)
					}
				} else {
					ui.Text(c, "• "+label).FontSize(theme.InterfaceSize(10)).TextColor(accent).Grow(1)
				}
				if strings.TrimSpace(entry.URL) != "" {
					copyAction := ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(theme.InterfaceSize(10)).
						BorderWidth(0).Background(ui.Transparent).TextColor(muted).Padding(0, 2, 0, 2)
					if copyAction.Clicked() {
						mygo.Clipboard.WriteText(entry.URL)
					}
				}
			})
			if snippet := strings.TrimSpace(entry.Snippet); snippet != "" {
				ui.Text(c, snippet).FontSize(theme.InterfaceSize(10)).TextColor(muted).Grow(1)
			}
			if published := strings.TrimSpace(entry.PublishedAt); published != "" {
				ui.Text(c, published).FontSize(theme.InterfaceSize(9)).TextColor(muted).Grow(1)
			}
		}
	})
}
