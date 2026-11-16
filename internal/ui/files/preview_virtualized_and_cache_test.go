package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestSourceRangesCacheKeyedByPathDraftVersionDarkAndFontSize(t *testing.T) {
	resetSourceRangeCacheForTest()
	t.Cleanup(resetSourceRangeCacheForTest)

	code := "package main\nfunc main() { println(\"hello\") }\n"
	root, path := "/fixture/workspace", "main.go"
	tab := &FileTab{Path: path, Preview: code, Draft: code}
	tabs := &Tabs{
		Preferences: &PreviewPreferences{},
		Workspaces:  map[string]*WorkspaceTabs{filepath.Clean(root): {Open: []*FileTab{tab}, Active: path}},
	}
	m := Model{
		Tabs:          tabs,
		WorkspaceRoot: root,
		SelectedFile:  &path,
		Preview:       &tab.Preview,
		EditDraft:     &tab.Draft,
		SystemDark:    false,
	}

	style := sourceStyle(m, ui.RGB(0, 0, 0), ui.RGB(100, 100, 100))
	first := sourceRangesFor(m, path, tab.Draft, style)
	if len(first) == 0 || sourceRangeMisses.Load() != 1 {
		t.Fatalf("expected 1 miss on initial highlight, got misses=%d ranges=%d", sourceRangeMisses.Load(), len(first))
	}

	// Repeated render frames hit cache cleanly without re-running chroma.
	for range 5 {
		_ = sourceRangesFor(m, path, tab.Draft, style)
	}
	if sourceRangeMisses.Load() != 1 || sourceRangeHits.Load() != 5 {
		t.Fatalf("unexpected cache counts on stable key: misses=%d hits=%d", sourceRangeMisses.Load(), sourceRangeHits.Load())
	}

	// Editing draft advances draftVersion and triggers a single re-tokenize.
	tab.Draft += "// edited\n"
	_ = sourceRangesFor(m, path, tab.Draft, style)
	if sourceRangeMisses.Load() != 2 {
		t.Fatalf("draft change did not invalidate cache: misses=%d", sourceRangeMisses.Load())
	}

	// Dark theme toggle invalidates cache.
	m.SystemDark = true
	darkStyle := sourceStyle(m, ui.RGB(240, 240, 240), ui.RGB(140, 140, 140))
	_ = sourceRangesFor(m, path, tab.Draft, darkStyle)
	if sourceRangeMisses.Load() != 3 {
		t.Fatalf("dark mode toggle did not invalidate cache: misses=%d", sourceRangeMisses.Load())
	}

	// Font size change invalidates cache.
	prevPrefs := theme.RenderPreferences()
	nextPrefs := prevPrefs
	nextPrefs.Code.Size = prevPrefs.Code.Size + 2
	theme.SetPreferences(nextPrefs)
	t.Cleanup(func() {
		theme.SetPreferences(prevPrefs)
	})
	_ = sourceRangesFor(m, path, tab.Draft, darkStyle)
	if sourceRangeMisses.Load() != 4 {
		t.Fatalf("font size change did not invalidate cache: misses=%d", sourceRangeMisses.Load())
	}

	// Path change invalidates cache.
	_ = sourceRangesFor(m, "other.go", tab.Draft, darkStyle)
	if sourceRangeMisses.Load() != 5 {
		t.Fatalf("path change did not invalidate cache: misses=%d", sourceRangeMisses.Load())
	}
}

func TestReadOnlyPreviewListVirtualizationAndCtrlFFindNavigation(t *testing.T) {
	resetSourceRangeCacheForTest()
	t.Cleanup(resetSourceRangeCacheForTest)

	var b strings.Builder
	for i := 0; i < 1200; i++ {
		switch i {
		case 5:
			b.WriteString("func alphaMatch() { // needle_token }\n")
		case 850:
			b.WriteString("func midMatch() { // needle_token }\n")
		case 1180:
			b.WriteString("func tailMatch() { // needle_token }\n")
		default:
			fmt.Fprintf(&b, "var line_%04d = %d\n", i, i)
		}
	}
	text := b.String()
	root, path := t.TempDir(), "big.go"
	if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	tab := &FileTab{Path: path, Preview: text, Draft: text}
	tabs := &Tabs{
		Preferences: &PreviewPreferences{WordWrap: true},
		Workspaces:  map[string]*WorkspaceTabs{filepath.Clean(root): {Open: []*FileTab{tab}, Active: path}},
	}
	var previewList ui.ListState
	editing := false
	errMsg := ""
	m := Model{
		ResourceOnly:  true,
		Tabs:          tabs,
		WorkspaceRoot: root,
		SelectedFile:  &path,
		Preview:       &tab.Preview,
		EditDraft:     &tab.Draft,
		PreviewErr:    &errMsg,
		Editing:       &editing,
		PreviewList:   &previewList,
	}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 720, 360)

	first, last := previewList.Visible()
	if first != 0 || last <= first || last >= 100 {
		t.Fatalf("expected virtualized top window in [0..100), got visible=[%d..%d]", first, last)
	}
	if !tt.HasText("var line_0000 = 0") {
		t.Fatalf("top line missing from virtualized PreviewList, got texts: %v", tt.Texts())
	}
	if tt.HasText("func tailMatch() { // needle_token }") {
		t.Fatal("offscreen bottom line 1180 should not be built when scrolled at top")
	}

	// Open Ctrl+F find bar and search across virtualized file.
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	if !tab.findOpen {
		t.Fatal("Ctrl+F did not open file find bar")
	}

	tt.Type("needle_token")
	tt.Frame()
	if !tt.HasText("1/3") {
		t.Fatalf("expected 1/3 match counter, texts=%v", tt.Texts())
	}

	// Step to match 2/3 at line 850 and verify PreviewList scrolls cleanly to it.
	if err := tt.Click("下一个匹配"); err != nil {
		if err2 := tt.Click("Next match"); err2 != nil {
			t.Fatalf("failed clicking next match: %v / %v", err, err2)
		}
	}
	tt.Frame()
	if !tt.HasText("2/3") || !tt.HasText("func midMatch() { // needle_token }") {
		t.Fatalf("next match did not jump PreviewList to line 850: visible=%v texts=%v", func() [2]int { a, b := previewList.Visible(); return [2]int{a, b} }(), tt.Texts())
	}

	// Step to match 3/3 at line 1180.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !tt.HasText("3/3") || !tt.HasText("func tailMatch() { // needle_token }") {
		t.Fatalf("Enter did not advance to final match at line 1180")
	}

	// Escape closes find bar cleanly.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tab.findOpen {
		t.Fatal("Escape did not close file find bar")
	}
}
