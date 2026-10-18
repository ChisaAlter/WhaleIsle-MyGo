package composer

import (
	"os"
	"path/filepath"
	"testing"
)

type syntheticFileTarget struct {
	files []string
	over  bool
}

func (f *syntheticFileTarget) DroppedFiles() []string { v := f.files; f.files = nil; return v }
func (f *syntheticFileTarget) FileDragOver() bool     { return f.over }
func TestConversationFileDropIntake(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "attachment.pdf")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	draft := NewDraftState("retained draft")
	calls := 0
	model := Model{Draft: &draft.Text, State: draft, OnAttachFiles: func(files []string) {
		calls++
		if len(files) != 1 || files[0] != path {
			t.Fatal(files)
		}
	}}
	target := &syntheticFileTarget{over: true}
	if !FileDropTarget(target, model) || calls != 0 {
		t.Fatal("hover performed intake")
	}
	target.files = []string{path}
	target.over = false
	if FileDropTarget(target, model) || calls != 1 || draft.Text != "retained draft" {
		t.Fatal("intake failed")
	}
	FileDropTarget(target, model)
	if calls != 1 {
		t.Fatal("duplicate intake")
	}
	target.files = []string{path, root}
	FileDropTarget(target, model)
	if calls != 1 || draft.Error == "" {
		t.Fatal("directory admitted")
	}
	draft.Error = ""
	draft.Preparing = 1
	target.files = []string{path}
	FileDropTarget(target, model)
	if calls != 1 || draft.Error == "" {
		t.Fatal("busy intake admitted")
	}
	if FileDropTarget(&syntheticFileTarget{over: true}, Model{}) {
		t.Fatal("missing draft registered")
	}
}
