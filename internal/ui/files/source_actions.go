package files

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets/markdown"
)

func selectedTab(m Model) *FileTab {
	if m.SelectedFile == nil {
		return nil
	}
	return findTab(tabsFor(m), *m.SelectedFile)
}

func previewText(m Model) string {
	if m.EditDraft != nil {
		return *m.EditDraft
	}
	if m.Preview != nil {
		return *m.Preview
	}
	return ""
}

func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".mdx":
		return true
	}
	return false
}

// RequestLine reveals a one-based source line. It never changes the draft.
// A request also switches Markdown from Rendered to Source.
func RequestLine(m Model, line int) {
	t := selectedTab(m)
	if t == nil {
		return
	}
	lines := sourceLines([]rune(previewText(m)))
	if line < 1 {
		line = 1
	}
	if line > len(lines) {
		line = len(lines)
	}
	t.RenderedMarkdown = false
	t.revealLine = line
	t.lineDraft = strconv.Itoa(line)
	t.selectionText = ""
	t.selectionStart, t.selectionEnd = 0, 0
	if m.Editing != nil && *m.Editing && editViewModeOf(m) == EditViewHighlight {
		setEditViewMode(m, EditViewEdit)
	}
}

func paintSourceActions(c *ui.Context, m Model, path string, muted ui.Color) {
	t := selectedTab(m)
	if t == nil || imageMode(m, path) || (m.PreviewErr != nil && *m.PreviewErr != "") {
		return
	}
	text := previewText(m)
	ui.Row(c).Padding(2, 12, 4, 12).Gap(6).AlignItems(ui.Center).Children(func() {
		if isMarkdown(path) {
			if ui.Button(c, label("源码", "Source")).FontSize(11).Disabled(!t.RenderedMarkdown).Clicked() {
				t.RenderedMarkdown = false
				t.selectionText = ""
			}
			if ui.Button(c, label("渲染", "Rendered")).FontSize(11).Disabled(t.RenderedMarkdown).Clicked() {
				t.RenderedMarkdown = true
				t.selectionText = ""
				t.revealLine = 0
			}
		}
		ui.TextInput(c, &t.lineDraft).Width(64).Placeholder(label("行号", "Line"))
		if ui.Button(c, label("跳到行", "Go to line")).FontSize(11).Clicked() {
			line, err := strconv.Atoi(strings.TrimSpace(t.lineDraft))
			if err != nil || line < 1 {
				setFileStatus(m, label("请输入大于零的行号", "Enter a positive line number"))
			} else {
				RequestLine(m, line)
			}
		}
		if !t.RenderedMarkdown && t.selectionText == text {
			if fragment, ok := FormatSelectedLines(path, text, t.selectionStart, t.selectionEnd); ok {
				if ui.Button(c, label("复制所选行", "Copy selected lines")).FontSize(11).Clicked() {
					mygo.Clipboard.WriteText(fragment)
				}
				if ui.Button(c, label("添加到对话", "Add to chat")).FontSize(11).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Clicked() {
					if err := m.OnAddToChat(fragment); err != nil {
						setFileStatus(m, err.Error())
					} else {
						setFileStatus(m, label("所选行已添加到对话草稿", "Selected lines added to the chat draft"))
					}
				}
			}
		}
	})
	_ = muted
}

func setFileStatus(m Model, text string) {
	if m.EditStatus != nil {
		*m.EditStatus = text
	}
}

// paintSourceArea uses the native editor's real rune selection. Highlight spans
// decorate the original source; rendered Markdown never substitutes for it.
func paintSourceArea(c *ui.Context, m Model, path string, text *string, readOnly bool, key string, fg, muted ui.Color) {
	ui.Column(c).Key(path + "/" + key).Fill().MinHeight(0).MinWidth(0).Children(func() {
		editor := ui.TextArea(c, text).Fill().MinHeight(0).Font("Consolas").
			FontSize(12).TextColor(fg).Padding(8, 8, 8, 8).ReadOnly(readOnly)
		if !isMarkdown(path) && markdown.CanSourceHighlight(path, *text) {
			editor.TextRanges(sourceRanges(path, *text, sourceStyle(m, fg, muted))...)
		}
		t := selectedTab(m)
		if t == nil {
			return
		}
		if t.revealLine > 0 {
			ranges := sourceLines([]rune(*text))
			line := min(t.revealLine, len(ranges)) - 1
			editor.SetTextSelection(ranges[line].start, ranges[line].end).Focus()
			t.revealLine = 0
		}
		if editor.Focused() {
			if editor.Composing() {
				t.selectionText = ""
				t.selectionStart, t.selectionEnd = 0, 0
			} else if editor.Shortcut(0, ui.KeyEscape) {
				_, end := editor.TextSelection()
				editor.SetTextSelection(end, end)
				t.selectionText = ""
				t.selectionStart, t.selectionEnd = 0, 0
			} else {
				t.selectionStart, t.selectionEnd = editor.TextSelection()
				t.selectionText = *text
			}
		}
	})
}

// The syntax highlighter normalizes trailing newlines. Only use its colors
// when all other source bytes still match, and clamp spans to the source.
func sourceRanges(path, text string, style markdown.Style) []ui.TextRange {
	spans := markdown.SourceSpans(path, text, style)
	var joined strings.Builder
	for _, span := range spans {
		joined.WriteString(span.Text)
	}
	if strings.TrimRight(joined.String(), "\n") != strings.TrimRight(text, "\n") {
		return nil
	}
	limit, at := utf8.RuneCountInString(text), 0
	var ranges []ui.TextRange
	for _, span := range spans {
		end := min(at + utf8.RuneCountInString(span.Text), limit)
		if end > at {
			ranges = append(ranges, ui.TextRange{Start: at, End: end, Color: span.Color, Weight: span.Weight})
		}
		at = end
	}
	return ranges
}

type sourceLine struct { start, end int }

// sourceLines recognizes LF, CRLF and CR while retaining native rune offsets.
func sourceLines(text []rune) []sourceLine {
	var out []sourceLine
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '\n' && text[i] != '\r' {
			continue
		}
		out = append(out, sourceLine{start, i})
		if text[i] == '\r' && i+1 < len(text) && text[i+1] == '\n' {
			i++
		}
		start = i+1
	}
	return append(out, sourceLine{start, len(text)})
}

func lineAt(lines []sourceLine, offset int) int {
	for i := 1; i < len(lines); i++ {
		if offset < lines[i].start {
			return i-1
		}
	}
	return len(lines)-1
}

func backtickFence(text string, minimum int) string {
	longest, run := 0, 0
	for _, ch := range text {
		if ch == '\x60' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat("\x60", max(minimum, longest+1))
}

// FormatSelectedLines matches Files Add to chat: complete selected source
// lines, a one-based line range and relative path, then a fenced text block.
// end is exclusive; a selection ending at the next line's start excludes it.
func FormatSelectedLines(path, text string, start, end int) (string, bool) {
	runes := []rune(text)
	start, end = max(0, start), min(len(runes), end)
	if end <= start || start >= len(runes) {
		return "", false
	}
	lines := sourceLines(runes)
	first, last := lineAt(lines, start), lineAt(lines, end-1)
	var selected []string
	for _, line := range lines[first:last+1] {
		selected = append(selected, string(runes[line.start:line.end]))
	}
	body := strings.Join(selected, "\n")
	rangeLabel := fmt.Sprintf("L%d", first+1)
	if last != first {
		rangeLabel += fmt.Sprintf(" to L%d", last+1)
	}
	path = strings.ReplaceAll(strings.ReplaceAll(path, "\r", " "), "\n", " ")
	inline := backtickFence(path, 1)
	if strings.HasPrefix(path, "\x60") || strings.HasSuffix(path, "\x60") {
		path = " " + path + " "
	}
	fence := backtickFence(body, 3)
	return rangeLabel + " " + inline + path + inline + "\n\n" + fence + "text\n" + body + "\n" + fence, true
}
