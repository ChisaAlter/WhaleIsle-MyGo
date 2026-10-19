package frame

import (
	"testing"
	"whaleisle/internal/ui/files"
)

func TestWorkspaceNewFileDefaultAndRestoredExplicitModes(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	m := Model{Sizes: &LayoutSizes{Workspace: w}}
	created := w.fileState(w.Root, m)
	if created.Mode != files.EditViewEdit {
		t.Fatalf("new file pane mode = %v", created.Mode)
	}
	for _, explicit := range []files.EditViewMode{files.EditViewSplit, files.EditViewHighlight} {
		created.Mode, created.Draft, created.Preview = explicit, "retained draft", "baseline"
		restored, err := RestoreWorkspacePanes(w.Export())
		if err != nil {
			t.Fatal(err)
		}
		recovered := restored.fileState(restored.Root, m)
		if recovered.Mode != explicit || recovered.Draft != "retained draft" || recovered.Preview != "baseline" {
			t.Fatalf("restore changed explicit mode/draft: %+v", recovered)
		}
	}
}
