package files

import (
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"strings"
	"testing"
)

func previewViewModel(text string) Model {
	root, path := "/fixture/workspace", "long.txt"
	tab := &FileTab{Path: path, Preview: text, Draft: text}
	tabs := &Tabs{Preferences: &PreviewPreferences{}, Workspaces: map[string]*WorkspaceTabs{filepath.Clean(root): {Open: []*FileTab{tab}, Active: path}}}
	return Model{Tabs: tabs, WorkspaceRoot: root, SelectedFile: &path, Preview: &tab.Preview, EditDraft: &tab.Draft}
}
func TestSourceWrapNativeHorizontalScroll(t *testing.T) {
	m := previewViewModel(strings.Repeat("long words ", 80))
	var selectStart, selectEnd int
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			editor := sourceTextArea(c, m, m.EditDraft, false, "editor", ui.RGB(0, 0, 0)).Label("Source fixture")
			selectStart, selectEnd = editor.TextSelection()
		})
	}, 280, 160)
	view := sourceView(m, "editor")
	if view.scroll.MaxX < 500 {
		t.Fatalf("long unwrapped text has no horizontal range: %+v", view.scroll)
	}
	tester.ClickAt(30, 20)
	tester.Key(ui.Cmd, ui.KeyA)
	before := *m.EditDraft
	start, end := selectStart, selectEnd
	if end <= start {
		t.Fatal("selection fixture did not select text")
	}
	m.Tabs.Preferences.WordWrap = true
	tester.Frame()
	if view.scroll.MaxX > 1 {
		t.Fatalf("wrapped source still scrolls horizontally: %+v", view.scroll)
	}
	nextStart, nextEnd := selectStart, selectEnd
	if start != nextStart || end != nextEnd || *m.EditDraft != before {
		t.Fatal("wrap toggle changed selection or draft")
	}
	tester.Type("replacement")
	if *m.EditDraft != "replacement" {
		t.Fatalf("native editor no longer edits: %q", *m.EditDraft)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if *m.EditDraft != before {
		t.Fatal("wrap toggle lost native undo")
	}
	m.Tabs.Preferences.WordWrap = false
	tester.Frame()
	if view.scroll.MaxX < 500 {
		t.Fatal("toggle back did not restore horizontal range")
	}
}
func TestPreviewWrapTabsRecovery(t *testing.T) {
	m := previewViewModel("saved")
	*m.EditDraft = "unsaved"
	m.Tabs.Preferences.WordWrap = true
	raw, err := m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	var restored Tabs
	if err = restored.Restore(raw); err != nil {
		t.Fatal(err)
	}
	if restored.Preferences == nil || !restored.Preferences.WordWrap {
		t.Fatal("Files recovery lost wrap preference")
	}
	w := restored.Workspaces[filepath.Clean(m.WorkspaceRoot)]
	if w == nil || len(w.Open) != 1 || w.Open[0].Draft != "unsaved" || w.Open[0].Preview != "saved" {
		t.Fatal("preference persistence changed unsaved draft")
	}
}
