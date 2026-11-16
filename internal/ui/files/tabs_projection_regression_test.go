package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareTabsReconcilesTransientEmptySelectedFileWithoutDetachingTab(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "note.txt")
	if err := os.WriteFile(notePath, []byte("disk body"), 0600); err != nil {
		t.Fatal(err)
	}

	tabs := &Tabs{}
	var selected, preview, draft, problem, status string
	var editing bool
	m := Model{
		BindingID:     "host/session",
		WorkspaceRoot: root,
		Tabs:          tabs,
		SelectedFile:  &selected,
		Preview:       &preview,
		EditDraft:     &draft,
		PreviewErr:    &problem,
		EditStatus:    &status,
		Editing:       &editing,
	}

	selectFile(m, "note.txt")
	prepareTabs(m)
	draft = "unsaved note"
	editing = true
	prepareTabs(m)

	// Simulate an external mirror wipe clearing only SelectedFile while tab strip retains active note.txt.
	selected = ""
	preview = ""
	draft = ""
	prepareTabs(m)

	if selected != "note.txt" {
		t.Fatalf("selected = %q, want %q after transient empty scalar reconciliation", selected, "note.txt")
	}
	if draft != "unsaved note" || !editing {
		t.Fatalf("draft/editing lost during reconciliation: draft=%q editing=%v", draft, editing)
	}

	// Closing the last tab intentionally clears both w.Active and SelectedFile.
	if !closeTab(m, "note.txt", true) {
		t.Fatal("closeTab failed")
	}
	prepareTabs(m)
	if selected != "" {
		t.Fatalf("selected = %q after closing last tab, want empty", selected)
	}
}
