package files

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

const (
	largeReadOnlyPreviewLineThreshold = 400
	largeReadOnlyPreviewByteThreshold = 32 * 1024
	virtualHighlightMaxBytes          = 256 * 1024
)

type previewVirtualRowKey struct {
	path string
	line int
}

func hasAtLeastLines(text string, limit int) bool {
	if limit <= 1 {
		return text != ""
	}
	count := 1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			count++
			if count >= limit {
				return true
			}
		}
	}
	return false
}

func shouldVirtualizeReadOnlyPreview(tab *FileTab, text string) bool {
	if len(text) >= largeReadOnlyPreviewByteThreshold {
		return true
	}
	if tab != nil && (tab.Truncated || tab.textPaged) {
		return true
	}
	return hasAtLeastLines(text, largeReadOnlyPreviewLineThreshold)
}

func (t *FileTab) cachedSourceLines(text string) ([]string, []sourceLine) {
	if t != nil && t.sourceLinesText == text && t.sourceLineStrings != nil {
		return t.sourceLineStrings, t.sourceLineSpans
	}
	runes := []rune(text)
	spans := sourceLines(runes)
	lines := make([]string, len(spans))
	for i, sp := range spans {
		lines[i] = string(runes[sp.start:sp.end])
	}
	if t != nil {
		t.sourceLinesText = text
		t.sourceLineSpans = spans
		t.sourceLineStrings = lines
		t.sourceLineRich = nil
		t.sourceRangeKeySig = ""
	}
	return lines, spans
}

func (t *FileTab) virtualList(m Model, key string) *ui.ListState {
	if key == "source" && m.PreviewList != nil {
		return m.PreviewList
	}
	if t != nil {
		if t.virtualLists == nil {
			t.virtualLists = map[string]*ui.ListState{}
		}
		if list := t.virtualLists[key]; list != nil {
			return list
		}
		list := &ui.ListState{}
		t.virtualLists[key] = list
		return list
	}
	return &ui.ListState{}
}

func buildLineHighlightSpans(lines []string, lineSpans []sourceLine, ranges []ui.TextRange, defaultColor ui.Color) [][]ui.Span {
	if len(ranges) == 0 {
		return nil
	}
	out := make([][]ui.Span, len(lines))
	rIdx := 0
	for li, lineStr := range lines {
		sp := lineSpans[li]
		lineLen := sp.end - sp.start
		if lineLen == 0 {
			continue
		}
		for rIdx < len(ranges) && ranges[rIdx].End <= sp.start {
			rIdx++
		}
		if rIdx >= len(ranges) || ranges[rIdx].Start >= sp.end {
			continue
		}
		lineRunes := []rune(lineStr)
		var spans []ui.Span
		pos := 0
		k := rIdx
		for k < len(ranges) && ranges[k].Start < sp.end {
			rng := ranges[k]
			s := max(0, rng.Start-sp.start)
			e := min(lineLen, rng.End-sp.start)
			if s > pos {
				spans = append(spans, ui.Span{Text: string(lineRunes[pos:s]), Color: defaultColor})
			}
			if e > s {
				spans = append(spans, ui.Span{
					Text:   string(lineRunes[s:e]),
					Color:  rng.Color,
					Weight: rng.Weight,
				})
				pos = e
			}
			if rng.End > sp.end {
				break
			}
			k++
		}
		if pos < len(lineRunes) {
			spans = append(spans, ui.Span{Text: string(lineRunes[pos:]), Color: defaultColor})
		}
		if len(spans) > 0 {
			out[li] = spans
		}
		rIdx = k
	}
	return out
}

func splitLineSpansWithHighlight(lineStr string, base []ui.Span, matchStart, matchEnd int, defaultColor, matchBG ui.Color) []ui.Span {
	runes := []rune(lineStr)
	ms := max(0, matchStart)
	me := min(len(runes), matchEnd)
	if me <= ms {
		return base
	}
	if len(base) == 0 {
		var out []ui.Span
		if ms > 0 {
			out = append(out, ui.Span{Text: string(runes[:ms]), Color: defaultColor})
		}
		out = append(out, ui.Span{Text: string(runes[ms:me]), Color: defaultColor, Weight: 600, Background: matchBG})
		if me < len(runes) {
			out = append(out, ui.Span{Text: string(runes[me:]), Color: defaultColor})
		}
		return out
	}
	var out []ui.Span
	offset := 0
	for _, s := range base {
		n := runeCountClip(s.Text)
		end := offset + n
		if end <= ms || offset >= me {
			out = append(out, s)
		} else {
			sr := []rune(s.Text)
			if ms > offset {
				c := s
				c.Text = string(sr[:ms-offset])
				out = append(out, c)
			}
			hiStart := max(0, ms-offset)
			hiEnd := min(n, me-offset)
			if hiEnd > hiStart {
				c := s
				c.Text = string(sr[hiStart:hiEnd])
				c.Background = matchBG
				c.Weight = max(c.Weight, 600)
				out = append(out, c)
			}
			if me < end {
				c := s
				c.Text = string(sr[me-offset:])
				out = append(out, c)
			}
		}
		offset = end
	}
	return out
}

func paintVirtualizedReadOnlySource(c *ui.Context, m Model, path, text string, tab *FileTab, key string, fg, muted ui.Color) {
	lines, lineSpans := tab.cachedSourceLines(text)
	listState := tab.virtualList(m, key)
	if listState == nil {
		listState = &ui.ListState{}
	}

	var lineRich [][]ui.Span
	if !isMarkdown(path) && len(text) <= virtualHighlightMaxBytes && markdown.CanSourceHighlight(path, text) {
		style := sourceStyle(m, fg, muted)
		font := theme.RenderPreferences().Code
		sig := fmt.Sprintf("%s|%s|%v|%.2f|%d", path, m.ThemeID, m.SystemDark, font.Size, len(lines))
		if tab != nil && tab.sourceRangeKeySig == sig && tab.sourceLineRich != nil {
			lineRich = tab.sourceLineRich
		} else {
			ranges := sourceRangesFor(m, path, text, style)
			lineRich = buildLineHighlightSpans(lines, lineSpans, ranges, fg)
			if tab != nil {
				tab.sourceRangeKeySig = sig
				tab.sourceLineRich = lineRich
			}
		}
	}

	matches, activeIdx := activeFindMatchForTab(tab, text, lineSpans)
	activeMatchLine := -1
	var activeMatch previewMatch
	if activeIdx >= 0 && activeIdx < len(matches) {
		activeMatch = matches[activeIdx]
		activeMatchLine = activeMatch.line
	}
	if tab != nil {
		if tab.revealLine > 0 && len(lines) > 0 {
			target := min(max(0, tab.revealLine-1), len(lines)-1)
			listState.ScrollTo(target, ui.Center)
			tab.revealLine = 0
		}
		if tab.findScrollPending && activeMatchLine >= 0 {
			listState.ScrollTo(activeMatchLine, ui.Center)
			tab.findScrollPending = false
			tab.revealMatchPending = false
		}
	}

	codeFont := theme.RenderPreferences().Code
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	accent := chrome.Color(colors.Accent)
	matchBG := ui.RGBA(accent.R, accent.G, accent.B, 0.25)
	wrap := previewPreferences(m).WordWrap
	gutterW := float32(max(36, (len(fmt.Sprintf("%d", len(lines)))+1)*8))
	textRevision := uint64(0)
	if tab != nil {
		textRevision = tab.sourceVersionFor(text)
	}

	renderList := func(minWidth float32) {
		list := ui.List(c.Key(path+"/"+key+"/virtual-list"), listState, len(lines)).
			ItemKey(func(i int) any { return previewVirtualRowKey{path: path, line: i} }).
			SelectionText(textRevision, func(i int) []string { return lines[i : i+1] }).
			FillHeight().
			MinHeight(0).
			Padding(6, 8, 6, 8).
			Selectable()
		// PageShortcut deliberately leaves focused selectable text alone.
		// Find in this document is owned by this list, so it remains available
		// after clicking or copying source without widening page shortcuts.
		if list.Shortcut(ui.Cmd, ui.KeyF) {
			triggerPreviewFind(m)
		}
		if minWidth > 0 {
			list.Width(minWidth)
		} else {
			list.FillWidth().MinWidth(0)
		}
		list.Rows(func(row ui.ListRow) {
			idx := row.Index
			rc := row.Context
			lineStr := lines[idx]
			var baseSpans []ui.Span
			if idx < len(lineRich) {
				baseSpans = lineRich[idx]
			}
			if idx == activeMatchLine {
				baseSpans = splitLineSpansWithHighlight(lineStr, baseSpans, activeMatch.lineStart, activeMatch.lineEnd, fg, matchBG)
			}
			rowEl := ui.Row(rc).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start)
			if idx == activeMatchLine {
				rowEl.Background(ui.RGBA(accent.R, accent.G, accent.B, 0.08)).Radius(4)
			}
			rowEl.Children(func() {
				ui.Text(rc, fmt.Sprintf("%d", idx+1)).
					Width(gutterW).Shrink(0).
					Font(codeFont.Family).FontSize(max(11, codeFont.Size-1)).
					TextColor(muted).Unselectable()
				var txt ui.Element
				if len(baseSpans) > 0 {
					txt = ui.RichText(rc, baseSpans...).Label(lineStr).Font(codeFont.Family).FontSize(codeFont.Size).TextColor(fg)
				} else {
					txt = ui.Text(rc, lineStr).Font(codeFont.Family).FontSize(codeFont.Size).TextColor(fg)
				}
				if wrap {
					txt.Grow(1).MinWidth(0)
				} else {
					txt.SingleLine()
				}
			})
		})
	}

	if wrap {
		renderList(0)
		return
	}
	view := sourceView(m, key)
	scroll := ui.ScrollHorizontal(c).Fill().MinWidth(0).MinHeight(0).TrackScroll(&view.scroll)
	viewport := scroll.Bounds().W
	if viewport <= 0 {
		viewport = max(240, paneWidthOf(m))
		c.Invalidate()
	}
	longest := ""
	longestRunes := 0
	for _, ln := range lines {
		if n := len(ln); n > longestRunes {
			longest = ln
			longestRunes = n
		}
	}
	width := sourceAreaWidth(false, viewport, sourceMeasuredWidth(m, longest))
	scroll.Children(func() {
		renderList(width)
	})
}
