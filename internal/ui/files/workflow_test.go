package files

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestDraftTabsSaveAndConflict(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(root, p), []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var selected, preview, draft, problem, status string
	editing := false
	tabs := Tabs{}
	m := Model{WorkspaceRoot: root, Tabs: &tabs, SelectedFile: &selected, Preview: &preview, PreviewErr: &problem, EditDraft: &draft, EditStatus: &status, Editing: &editing, OnSaveEdit: func(rel, text string) error { return workspacefs.WriteFileText(filepath.Join(root, rel), text) }}
	prepareTabs(m)
	selectFile(m, "a.go")
	draft = "draft"
	editing = true
	selectFile(m, "b.go")
	selectFile(m, "a.go")
	if draft != "draft" || !editing {
		t.Fatal("switch lost draft")
	}
	if closeTab(m, "a.go", false) {
		t.Fatal("dirty tab closed")
	}
	if err := saveDraft(m); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "a.go"))
	if string(data) != "draft" {
		t.Fatal("save missing")
	}
	draft = "next"
	os.WriteFile(filepath.Join(root, "a.go"), []byte("external"), 0600)
	if err := saveDraft(m); err == nil {
		t.Fatal("external edit overwritten")
	}
	if draft != "next" {
		t.Fatal("failed save lost draft")
	}
	data, _ = os.ReadFile(filepath.Join(root, "a.go"))
	if string(data) != "external" {
		t.Fatal("conflict overwrote disk")
	}
	if !closeTab(m, "a.go", true) || selected != "b.go" {
		t.Fatal("explicit discard close failed")
	}
}
