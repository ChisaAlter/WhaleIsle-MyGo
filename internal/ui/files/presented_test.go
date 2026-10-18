package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPresentedFileClickUsesSessionCwdAndDraftTabs(t *testing.T) {
	root := t.TempDir()
	for name, text := range map[string]string{"first.md": "saved first", "report.md": "saved report"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var selected, preview, draft, problem, status string
	editing := false
	done := make(chan struct{}, 16)
	tabs := Tabs{}
	m := Model{BindingID: "host/session", WorkspaceRoot: root, Tabs: &tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status, Editing: &editing, notify: func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}}
	tabs.Bind(m.BindingID, root)
	open := func(path string) {
		t.Helper()
		handled, err := OpenPresentedPath(m, path)
		if !handled || err != nil {
			t.Fatalf("open %q: %v, %v", path, handled, err)
		}
		awaitFileTextRead(t, m, done)
	}
	open("first.md")
	draft, editing = "unsaved first", true
	open("report.md")
	if selected != "report.md" || preview != "saved report" {
		t.Fatalf("relative path did not use session cwd: %q %q", selected, preview)
	}
	open(filepath.Join(root, "first.md"))
	if draft != "unsaved first" || !editing {
		t.Fatal("returning to file lost its dirty draft")
	}
	open("first.md")
	if draft != "unsaved first" || !editing {
		t.Fatal("reopening active file reset dirty draft")
	}
	open(root)
	if selected != "" {
		t.Fatal("workspace root should show Files directory")
	}
	open("first.md")
	if draft != "unsaved first" {
		t.Fatal("directory navigation lost dirty draft")
	}
	for _, path := range []string{"../outside.md", filepath.Join(t.TempDir(), "outside.md"), `Z:\remote\report.md`, "missing.md"} {
		handled, err := OpenPresentedPath(m, path)
		if handled || err != nil || selected != "first.md" || draft != "unsaved first" {
			t.Fatalf("fallback mutated tabs for %q: %v %v", path, handled, err)
		}
	}
	m.WorkspaceRoot = ""
	if handled, err := OpenPresentedPath(m, "report.md"); handled || err != nil {
		t.Fatal("missing cwd must defer to Host")
	}
	m.WorkspaceRoot = root
	if handled, err := OpenPresentedPath(m, "https://example.com/report.md"); handled || err == nil {
		t.Fatal("file card URL was accepted")
	}
	if closeTab(m, "first.md", false) {
		t.Fatal("dirty close protection bypassed")
	}
}
