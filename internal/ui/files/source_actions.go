package files

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/theme"
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
	if t.readLoading || t.readResume || t.readBlocked || t.textPaged {
		t.pendingLine.begin()
	}
	if t.pendingLine.deferRequest(line) {
		if t.textPaged && !t.readResume && !t.readBlocked {
			t.pendingLine.version = t.textPage.Version
		}
		t.revealLine = 0 // The painter must not consume a target against stale text.
		t.lineDraft = strconv.Itoa(max(1, line))
		t.RenderedMarkdown = false
		if !t.readLoading && !t.readResume && !t.readBlocked {
			continueTextLine(m)
		}
		return
	}
	requestLineNow(m, line)
}

// Successful pages advance the captured target one page at a time. Failures
// never enter this path and retain the target for explicit retry.
func continueTextLine(m Model) {
	t := selectedTab(m)
	if t == nil || t.readLoading || t.readResume || t.readBlocked {
		return
	}
	if !t.pendingLine.acceptVersion(t.textPage.Version) {
		return
	}
	line := t.pendingLine.line
	if line > len(sourceLines([]rune(previewText(m)))) && t.textPaged && !t.textPage.EOF && !t.Dirty() {
		beginNextTextPage(m)
		return
	}
	if line = t.pendingLine.complete(true); line > 0 {
		requestLineNow(m, line)
	}
}

func requestLineNow(m Model, line int) {
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

func setFileStatus(m Model, text string) {
	if m.EditStatus != nil {
		*m.EditStatus = text
	}
}

// setFileError answers a preview action that failed in copy a person can act
// on; the raw error stays in the log.
func setFileError(m Model, err error) {
	if err != nil {
		log.Printf("files preview: %v", err)
	}
	setFileStatus(m, errcopy.Sentence(err))
}

const maxSourceRangeCacheEntries = 16

type sourceRangeCacheKey struct {
	path         string
	draftVersion uint64
	dark         bool
	fontSize     float32
	themeID      string
	fontFamily   string
}

type sourceRangeCacheEntry struct {
	key    sourceRangeCacheKey
	text   string
	ranges []ui.TextRange
}

var (
	sourceRangeMu     sync.Mutex
	sourceRangeOrder  []sourceRangeCacheKey
	sourceRangeMap    = map[sourceRangeCacheKey]sourceRangeCacheEntry{}
	sourceRangeHits   atomic.Int64
	sourceRangeMisses atomic.Int64
)

func resetSourceRangeCacheForTest() {
	sourceRangeMu.Lock()
	clear(sourceRangeMap)
	sourceRangeOrder = nil
	sourceRangeMu.Unlock()
	sourceRangeHits.Store(0)
	sourceRangeMisses.Store(0)
}

func fnv64aString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func (t *FileTab) sourceVersionFor(text string) uint64 {
	if t == nil {
		return fnv64aString(text)
	}
	if t.sourceTextVer == 0 || t.sourceTextLast != text {
		t.sourceTextVer++
		t.sourceTextLast = text
	}
	if t.editBuffer != nil && text == t.Draft {
		return t.sourceTextVer ^ (t.editBufferVersion << 32)
	}
	return t.sourceTextVer
}

func paintSourceArea(c *ui.Context, m Model, path string, text *string, readOnly bool, key string, fg, muted ui.Color) {
	ui.Column(c).Key(path + "/" + key).Fill().MinHeight(0).MinWidth(0).Children(func() {
		t := selectedTab(m)
		if readOnly && shouldVirtualizeReadOnlyPreview(t, *text) {
			paintVirtualizedReadOnlySource(c, m, path, *text, t, key, fg, muted)
			return
		}
		editor := sourceTextArea(c, m, text, readOnly, key, fg)
		if !isMarkdown(path) && markdown.CanSourceHighlight(path, *text) {
			editor.TextRanges(sourceRangesFor(m, path, *text, sourceStyle(m, fg, muted))...)
		}
		if t == nil {
			return
		}
		if t.revealLine > 0 {
			ranges := sourceLines([]rune(*text))
			line := min(t.revealLine, len(ranges)) - 1
			editor.SetTextSelection(ranges[line].start, ranges[line].end).Focus()
			t.revealLine = 0
		}
		if t.revealMatchPending {
			editor.SetTextSelection(t.revealMatchStart, t.revealMatchEnd)
			t.revealMatchPending = false
		}
		revealSourceCaret(c, m, editor, *text, key)
		if editor.Focused() {
			if editor.Shortcut(ui.Cmd, ui.KeyS) {
				if err := requestSaveDraft(m); err != nil {
					setFileError(m, err)
				}
			}
			if editor.Shortcut(ui.Cmd, ui.KeyF) {
				triggerPreviewFind(m)
			}
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

func sourceRangesFor(m Model, path, text string, style markdown.Style) []ui.TextRange {
	th := theme.Resolve(m.ThemeID)
	dark := th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark)
	codeFont := theme.RenderPreferences().Code
	draftVer := selectedTab(m).sourceVersionFor(text)
	key := sourceRangeCacheKey{
		path:         path,
		draftVersion: draftVer,
		dark:         dark,
		fontSize:     codeFont.Size,
		themeID:      m.ThemeID,
		fontFamily:   codeFont.Family,
	}
	return cachedSourceRanges(key, text, style)
}

// The syntax highlighter normalizes trailing newlines. Only use its colors
// when all other source bytes still match, and clamp spans to the source.
func sourceRanges(path, text string, style markdown.Style) []ui.TextRange {
	key := sourceRangeCacheKey{
		path:         path,
		draftVersion: fnv64aString(text),
		dark:         int(style.FG.R)+int(style.FG.G)+int(style.FG.B) > 384,
		fontSize:     style.FontSize,
	}
	return cachedSourceRanges(key, text, style)
}

func cachedSourceRanges(key sourceRangeCacheKey, text string, style markdown.Style) []ui.TextRange {
	sourceRangeMu.Lock()
	if entry, ok := sourceRangeMap[key]; ok && len(entry.text) == len(text) && entry.text == text {
		sourceRangeHits.Add(1)
		ranges := entry.ranges
		sourceRangeMu.Unlock()
		return ranges
	}
	sourceRangeMu.Unlock()

	sourceRangeMisses.Add(1)
	ranges := computeSourceRanges(key.path, text, style)

	sourceRangeMu.Lock()
	if _, exists := sourceRangeMap[key]; !exists {
		sourceRangeOrder = append(sourceRangeOrder, key)
		for len(sourceRangeOrder) > maxSourceRangeCacheEntries {
			evict := sourceRangeOrder[0]
			sourceRangeOrder = sourceRangeOrder[1:]
			delete(sourceRangeMap, evict)
		}
	}
	sourceRangeMap[key] = sourceRangeCacheEntry{key: key, text: text, ranges: ranges}
	sourceRangeMu.Unlock()
	return ranges
}

func computeSourceRanges(path, text string, style markdown.Style) []ui.TextRange {
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
		end := min(at+utf8.RuneCountInString(span.Text), limit)
		if end > at {
			ranges = append(ranges, ui.TextRange{Start: at, End: end, Color: span.Color, Weight: span.Weight})
		}
		at = end
	}
	return ranges
}

type sourceLine struct{ start, end int }

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
		start = i + 1
	}
	return append(out, sourceLine{start, len(text)})
}

func lineAt(lines []sourceLine, offset int) int {
	for i := 1; i < len(lines); i++ {
		if offset < lines[i].start {
			return i - 1
		}
	}
	return len(lines) - 1
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
	for _, line := range lines[first : last+1] {
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
