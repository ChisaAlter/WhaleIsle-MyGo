package files

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type previewMatch struct {
	start     int // rune offset in document
	end       int // rune offset in document (exclusive)
	line      int // zero-based line index
	lineStart int // rune offset within line
	lineEnd   int // rune offset within line (exclusive)
}

func triggerPreviewFind(m Model) {
	t := selectedTab(m)
	if t == nil || m.SelectedFile == nil || *m.SelectedFile == "" {
		return
	}
	path := *m.SelectedFile
	if imageMode(m, path) || mediaMode(m, path) || isTablePreviewPath(path) {
		return
	}
	if t.RenderedMarkdown && isMarkdown(path) {
		t.RenderedMarkdown = false
	}
	t.findOpen = true
	t.findFocusPending = true
	t.findScrollPending = true
}

func findSourceMatches(text, query string, lines []sourceLine) []previewMatch {
	if query == "" || text == "" {
		return nil
	}
	// Match case-insensitively across runes cleanly so UTF-8 rune offsets align
	// with ui.TextArea selection and per-line spans.
	textRunes := []rune(strings.ToLower(text))
	queryRunes := []rune(strings.ToLower(query))
	if len(queryRunes) == 0 || len(queryRunes) > len(textRunes) {
		return nil
	}
	var out []previewMatch
	lineIdx := 0
	for i := 0; i <= len(textRunes)-len(queryRunes); {
		matched := true
		for j := 0; j < len(queryRunes); j++ {
			if textRunes[i+j] != queryRunes[j] {
				matched = false
				break
			}
		}
		if !matched {
			i++
			continue
		}
		for lineIdx+1 < len(lines) && i >= lines[lineIdx+1].start {
			lineIdx++
		}
		lineStart := 0
		lineEnd := 0
		if lineIdx < len(lines) {
			lineStart = max(0, i-lines[lineIdx].start)
			lineEnd = min(lines[lineIdx].end-lines[lineIdx].start, lineStart+len(queryRunes))
		}
		out = append(out, previewMatch{
			start:     i,
			end:       i + len(queryRunes),
			line:      lineIdx,
			lineStart: lineStart,
			lineEnd:   lineEnd,
		})
		i += len(queryRunes)
	}
	return out
}

func activeFindMatchForTab(t *FileTab, text string, lines []sourceLine) ([]previewMatch, int) {
	if t == nil || !t.findOpen || t.findQuery == "" {
		return nil, -1
	}
	matches := findSourceMatches(text, t.findQuery, lines)
	if len(matches) == 0 {
		t.findMatchIndex = 0
		return nil, -1
	}
	if t.findMatchIndex < 0 {
		t.findMatchIndex = 0
	}
	if t.findMatchIndex >= len(matches) {
		t.findMatchIndex = len(matches) - 1
	}
	return matches, t.findMatchIndex
}

func applyFindJump(t *FileTab, matches []previewMatch, idx int) {
	if t == nil || idx < 0 || idx >= len(matches) {
		return
	}
	m := matches[idx]
	t.findMatchIndex = idx
	t.findScrollPending = true
	t.revealMatchStart = m.start
	t.revealMatchEnd = m.end
	t.revealMatchPending = true
}

func paintPreviewFindBar(c *ui.Context, m Model, path string, border, muted ui.Color) {
	t := selectedTab(m)
	if t == nil || !t.findOpen || m.SelectedFile == nil || *m.SelectedFile != path {
		return
	}
	if imageMode(m, path) || mediaMode(m, path) || isTablePreviewPath(path) {
		return
	}
	text := previewText(m)
	_, lineSpans := t.cachedSourceLines(text)
	matches, activeIdx := activeFindMatchForTab(t, text, lineSpans)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)

	ui.Row(c.Key("preview-find-bar")).FillWidth().Height(34).Shrink(0).Gap(6).Padding(4, 12, 4, 12).
		AlignItems(ui.Center).
		DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 0.5, W: r.W, H: 0.5}, border, 0)
		}).
		Children(func() {
			input := ui.TextInput(c.Key("preview-find-input"), &t.findQuery).
				Label(label("文件内查找", "Find in file")).
				Placeholder(label("查找文本…", "Find text…")).
				Width(200).Height(26).Radius(6).Border(0.5, border).
				FontSize(theme.InterfaceSize(12))
			if t.findFocusPending {
				input.Focus()
				t.findFocusPending = false
			}
			if input.Changed() || t.findQuery != t.findLastQuery {
				t.findLastQuery = t.findQuery
				t.findMatchIndex = 0
				matches, activeIdx = activeFindMatchForTab(t, text, lineSpans)
				if activeIdx >= 0 {
					applyFindJump(t, matches, activeIdx)
				}
			}
			counter := "0/0"
			if len(matches) > 0 && activeIdx >= 0 {
				counter = fmt.Sprintf("%d/%d", activeIdx+1, len(matches))
			}
			ui.Text(c, counter).FontSize(theme.InterfaceSize(11)).TextColor(muted).Shrink(0)

			stepMatch := func(delta int) {
				if len(matches) == 0 {
					return
				}
				next := (activeIdx + delta + len(matches)) % len(matches)
				applyFindJump(t, matches, next)
			}
			if input.Shortcut(ui.Shift, ui.KeyEnter) {
				stepMatch(-1)
			} else if input.Submitted() && !input.Composing() {
				stepMatch(1)
			}
			if widgets.Button(c, colors, "↑", widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: len(matches) == 0}).
				Label(label("上一个匹配", "Previous match")).Clicked() {
				stepMatch(-1)
			}
			if widgets.Button(c, colors, "↓", widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: len(matches) == 0}).
				Label(label("下一个匹配", "Next match")).Clicked() {
				stepMatch(1)
			}
			if (!input.Composing() && (input.Shortcut(0, ui.KeyEscape) || c.PageShortcut(0, ui.KeyEscape))) || widgets.Button(c, colors, "×", widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).
				Label(label("关闭查找", "Close find")).Clicked() {
				t.findOpen = false
				t.findFocusPending = false
				t.findScrollPending = false
				t.revealMatchPending = false
			}
		})
}

func runeCountClip(s string) int {
	return utf8.RuneCountInString(s)
}
