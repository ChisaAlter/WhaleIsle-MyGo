package chat

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// searchHitLimit keeps a grep result readable; the rest becomes a count line.
const searchHitLimit = 12

// searchHit is one match line: path, line(:column) and the snippet.
type searchHit struct {
	Path    string
	Line    string
	Snippet string
}

// parseSearchHits reads the Host's grep/glob text output. Recognized shapes are
// `path:line: snippet` and `path:line:column: snippet`; anything else stays a
// plain line so a new format never loses evidence.
func parseSearchHits(output string) []searchHit {
	var hits []searchHit
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		if hit, ok := splitSearchHit(trimmed); ok {
			hits = append(hits, hit)
			continue
		}
		hits = append(hits, searchHit{Snippet: trimmed})
	}
	return hits
}

// splitSearchHit finds the first colon followed by a line number (optionally a
// column number) so Windows drive colons and URLs stay in the path.
func splitSearchHit(line string) (searchHit, bool) {
	for index := 0; index < len(line); index++ {
		if line[index] != ':' {
			continue
		}
		rest := line[index+1:]
		digits := leadingDigits(rest)
		if digits == 0 {
			continue
		}
		lineNumber := rest[:digits]
		after := rest[digits:]
		if strings.HasPrefix(after, ":") {
			if column := leadingDigits(after[1:]); column > 0 && strings.HasPrefix(after[1+column:], ":") {
				snippet := strings.TrimSpace(after[1+column+1:])
				return searchHit{Path: line[:index], Line: lineNumber + ":" + after[1:1+column], Snippet: snippet}, true
			}
		}
		if !strings.HasPrefix(after, ":") {
			continue
		}
		snippet := strings.TrimSpace(after[1:])
		if path := strings.TrimSpace(line[:index]); path != "" {
			return searchHit{Path: path, Line: lineNumber, Snippet: snippet}, true
		}
	}
	return searchHit{}, false
}

func leadingDigits(text string) int {
	count := 0
	for count < len(text) && text[count] >= '0' && text[count] <= '9' {
		count++
	}
	return count
}

// snippetSpans highlights the query inside a hit snippet, keeping the source
// text verbatim. An empty query yields one plain span.
func snippetSpans(snippet, query string, st toolSeatStyle) []ui.Span {
	size := st.size()
	if snippet == "" {
		return nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []ui.Span{{Text: snippet, Size: size, Color: st.FG}}
	}
	lowerSnippet, lowerQuery := strings.ToLower(snippet), strings.ToLower(query)
	var spans []ui.Span
	cursor := 0
	for {
		at := strings.Index(lowerSnippet[cursor:], lowerQuery)
		if at < 0 {
			break
		}
		start := cursor + at
		if start > cursor {
			spans = append(spans, ui.Span{Text: snippet[cursor:start], Size: size, Color: st.FG})
		}
		spans = append(spans, ui.Span{
			Text:       snippet[start : start+len(query)],
			Size:       size,
			Color:      st.FG,
			Background: st.FG.Alpha(.16),
			Weight:     500,
		})
		cursor = start + len(query)
	}
	if cursor < len(snippet) {
		spans = append(spans, ui.Span{Text: snippet[cursor:], Size: size, Color: st.FG})
	}
	if len(spans) == 0 {
		spans = append(spans, ui.Span{Text: snippet, Size: size, Color: st.FG})
	}
	return spans
}

// searchHitsBlock paints grep/glob evidence as one line per hit: path:line in
// muted monospace followed by the highlighted snippet, all horizontally
// scrolled so a long line never widens the transcript.
func searchHitsBlock(c *ui.Context, st toolSeatStyle, key, query, output string, onCopy func(string)) {
	hits := parseSearchHits(output)
	payload := strings.TrimRight(output, "\n")
	size := st.size()
	font := theme.RenderPreferences().Code.Family
	toolSeat(c, st, key, "Search matches", payload, onCopy, func() {
		if len(hits) == 0 {
			ui.Text(c, i18n.T(i18n.KeyChatToolNoMatches)).FontSize(size).TextColor(st.Muted)
			return
		}
		shown := hits
		hidden := 0
		if len(shown) > searchHitLimit {
			hidden = len(shown) - searchHitLimit
			shown = shown[:searchHitLimit]
		}
		ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(st.maxHeight()).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
				for _, hit := range shown {
					hit := hit
					ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
						ui.Row(c).MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
							if loc := searchHitLocation(hit); loc != "" {
								ui.Text(c, loc).FontSize(size).Font(font).FixedLineHeight(toolSeatLine).
									TextColor(st.Muted).NoWrap()
							}
							if hit.Snippet != "" {
								ui.RichText(c, snippetSpans(hit.Snippet, query, st)...).
									Font(font).FixedLineHeight(toolSeatLine).NoWrap().TextColor(st.FG)
							}
						})
					})
				}
			})
		})
		if hidden > 0 {
			files := searchHitsFileCount(hits)
			ui.Text(c, toolSearchMatchesSummary(len(shown), float64(len(hits)), files, true)).
				FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.Muted)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatToolMoreHits), hidden)).
				FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.Muted)
		}
	})
}

func searchHitsFileCount(hits []searchHit) int {
	seen := make(map[string]struct{})
	for _, h := range hits {
		if h.Path != "" {
			seen[h.Path] = struct{}{}
		}
	}
	if len(seen) == 0 && len(hits) > 0 {
		return 1
	}
	return len(seen)
}

// searchHitLocation is the "path:line" prefix, muted and separate from the code.
func searchHitLocation(hit searchHit) string {
	if hit.Path == "" {
		return ""
	}
	if hit.Line == "" {
		return hit.Path
	}
	return hit.Path + ":" + hit.Line
}

// webResultBlock paints the typed web metadata: a fetch shows its URL, domain
// and status; a search lists title, domain and a two-line snippet per source,
// each opening the existing citation handler.
func webResultBlock(c *ui.Context, st toolSeatStyle, key string, web *session.WebResult, info ToolCardInfo, onOpenURL func(string), onCopy func(string)) {
	payload := strings.TrimSpace(info.Output)
	if payload == "" {
		// A typed result may carry no raw text; rebuild the evidence so the copy
		// seat still has something to yield.
		payload = webCopyPayload(web)
	}
	size := st.size()
	toolSeat(c, st, key, "Web result", payload, onCopy, func() {
		if web == nil {
			ui.Text(c, info.Output).FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.FG).Selectable()
			return
		}
		if web.Answer != "" {
			ui.Text(c, web.Answer).FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.FG).MaxLines(2)
		}
		if web.Kind == "fetch" && web.URL != "" {
			webSourceRow(c, st, session.WebSource{URL: web.URL, Title: web.URL}, fmt.Sprintf("HTTP %d", web.StatusCode), onOpenURL)
		}
		for _, source := range web.Sources {
			webSourceRow(c, st, source, "", onOpenURL)
		}
		if web.Truncated {
			ui.Text(c, i18n.T(i18n.KeyChatWebTruncated)).FontSize(size - 1).TextColor(st.Muted)
		}
	})
}

// webCopyPayload rebuilds what a typed web result would copy when the Host sent
// no raw text: one line per source plus its snippet.
func webCopyPayload(web *session.WebResult) string {
	if web == nil {
		return ""
	}
	var lines []string
	if web.URL != "" {
		lines = append(lines, web.URL)
	}
	for _, source := range web.Sources {
		line := strings.TrimSpace(source.Title)
		if source.URL != "" {
			line = strings.TrimSpace(line + " " + source.URL)
		}
		if line != "" {
			lines = append(lines, line)
		}
		if snippet := strings.TrimSpace(source.Snippet); snippet != "" {
			lines = append(lines, "  "+snippet)
		}
	}
	return strings.Join(lines, "\n")
}

// webSourceRow is one web card: title, domain, and a two-line snippet. The row
// opens the URL through the existing citation callback when there is one.
func webSourceRow(c *ui.Context, st toolSeatStyle, source session.WebSource, trailing string, onOpenURL func(string)) {
	size := st.size()
	domain := webDomain(source.URL)
	title := strings.TrimSpace(source.Title)
	if title == "" {
		title = domain
	}
	if title == "" {
		title = source.URL
	}
	row := ui.Column(c).FillWidth().MinWidth(0).Padding(4, 4, 4, 4).Radius(6).Gap(1)
	clickable := onOpenURL != nil && strings.TrimSpace(source.URL) != ""
	if clickable {
		row = row.Label(title).Tooltip(source.URL).Focusable().Role(ui.RoleButton)
	}
	if row.Hovered() && clickable {
		row.Background(st.FG.Alpha(.05))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.AccentOr(st.FG)).Grow(1).MinWidth(0).SingleLine()
			if trailing != "" {
				ui.Text(c, trailing).FontSize(size - 1).TextColor(st.Muted).NoWrap()
			}
		})
		meta := make([]string, 0, 2)
		if domain != "" && domain != title {
			meta = append(meta, domain)
		}
		if source.PublishedAt != "" {
			meta = append(meta, source.PublishedAt)
		}
		if len(meta) > 0 {
			ui.Text(c, strings.Join(meta, " · ")).FontSize(size - 1).TextColor(st.Muted).SingleLine()
		}
		if snippet := strings.TrimSpace(source.Snippet); snippet != "" {
			ui.Text(c, snippet).FontSize(size).FixedLineHeight(toolSeatLine).TextColor(st.Muted).MaxLines(2)
		}
	})
	if clickable && row.Clicked() {
		onOpenURL(source.URL)
	}
}

// AccentOr keeps titles accent-colored when the seat has an accent, falling back
// to the body color on surfaces that do not.
func (s toolSeatStyle) AccentOr(fallback ui.Color) ui.Color {
	if s.Accent.A != 0 {
		return s.Accent
	}
	return fallback
}

// webDomain is the host of a result URL, without the www prefix.
func webDomain(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.TrimPrefix(parsed.Host, "www.")
}

// todoItem is one parsed todo row.
type todoItem struct {
	Text   string
	Status string
}

// parseTodos reads a checklist output: `- [x] text`, `1. [ ] text`, a bare
// `[done] text` prefix, or plain lines. Unknown shapes stay visible as pending.
func parseTodos(output string) []todoItem {
	var items []todoItem
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		trimmed = strings.TrimLeft(trimmed, "-*•")
		trimmed = strings.TrimSpace(trimmed)
		// Drop an ordered-list number.
		if digits := leadingDigits(trimmed); digits > 0 && digits < len(trimmed) && trimmed[digits] == '.' {
			trimmed = strings.TrimSpace(trimmed[digits+1:])
		}
		status := "pending"
		if strings.HasPrefix(trimmed, "[") {
			if end := strings.IndexByte(trimmed, ']'); end > 0 {
				marker := strings.ToLower(strings.TrimSpace(trimmed[1:end]))
				switch marker {
				case "x", "✓", "done", "completed", "complete":
					status = "completed"
				case "~", "-", "doing", "running", "in_progress", "in-progress":
					status = "in_progress"
				case "", " ":
					status = "pending"
				default:
					status = marker
				}
				trimmed = strings.TrimSpace(trimmed[end+1:])
			}
		}
		if trimmed == "" {
			continue
		}
		items = append(items, todoItem{Text: trimmed, Status: status})
	}
	return items
}

// todoBlock paints the checklist: a state dot, the text and the status word, with
// completed rows quieted so the open work stays readable.
func todoBlock(c *ui.Context, st toolSeatStyle, key, output string, onCopy func(string)) {
	items := parseTodos(output)
	size := st.size()
	toolSeat(c, st, key, "Todo list", strings.TrimRight(output, "\n"), onCopy, func() {
		if len(items) == 0 {
			ui.Text(c, i18n.T(i18n.KeyChatTodoEmpty)).FontSize(size).TextColor(st.Muted)
			return
		}
		doneCount := 0
		for _, item := range items {
			if item.Status == "completed" {
				doneCount++
			}
		}
		ui.Text(c, todoCompletedFractionCount(doneCount, len(items))).FontSize(size - 1).TextColor(st.Muted)
		for _, item := range items {
			item := item
			done := item.Status == "completed"
			textColor := st.FG
			if done {
				textColor = st.Muted
			}
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				widgets.StateDot(c, widgetState(todoRowStatus(item.Status)))
				text := ui.Text(c, item.Text).FontSize(size).FixedLineHeight(toolSeatLine).TextColor(textColor).Grow(1).MinWidth(0)
				if done {
					text.Strikethrough()
				}
				ui.Text(c, todoStatusLabel(item.Status)).FontSize(size - 1).TextColor(st.Muted).NoWrap()
			})
		}
	})
}

// todoRowStatus maps a checklist status onto the shared state dot.
func todoRowStatus(status string) rowStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "✓":
		return statusDone
	case "in_progress", "in-progress", "running", "doing", "~":
		return statusRunning
	case "failed", "error", "blocked":
		return statusError
	case "cancelled", "canceled", "skipped":
		return statusIdle
	}
	return statusIdle
}

// todoStatusLabel is the localized status word; unknown statuses stay verbatim.
func todoStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "✓":
		return i18n.T(i18n.KeyChatTodoDone)
	case "in_progress", "in-progress", "running", "doing", "~":
		return i18n.T(i18n.KeyChatTodoRunning)
	case "pending", "", "todo":
		return i18n.T(i18n.KeyChatTodoPending)
	}
	return status
}
