package frame

import (
	"path/filepath"
	"testing"

	"whaleisle/internal/ui/files"
)

func TestWorkspaceNonFilePaneProjectionPreservesFileSelectionAndCapture(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	w := &WorkspacePanes{}
	w.syncRoute("files")
	filePane := w.Root
	fileID := filePane.ID
	filePane.Files = &WorkspaceFileState{
		Selected: "main.go",
		Preview:  "saved main",
		Draft:    "edited main",
		Editing:  true,
		Tabs: files.Tabs{
			Root: root,
			Workspaces: map[string]*files.WorkspaceTabs{
				root: {
					Active: "main.go",
					Open: []*files.FileTab{
						{Path: "main.go", Preview: "saved main", Draft: "edited main", Editing: true},
					},
				},
			},
		},
	}

	// Split in a non-file pane (diff) and activate it.
	diffPane := w.leaf("diff")
	diffPane.Diff = &WorkspaceDiffState{Root: root, Path: "main.go"}
	w.placeDockResource(diffPane, filePane, true)
	w.activateDockTab(diffPane.ID)

	var adapter WorkspaceFileState
	var diffAdapter WorkspaceDiffState
	sizes := DefaultLayout()
	sizes.Workspace = w
	m := diffAdapter.model(adapter.model(Model{
		Sizes:         &sizes,
		WorkspaceRoot: root,
		FileBindingID: "binding",
	}))

	w.BeginContentFrame("binding", root)
	t.Cleanup(w.DisposeContent)

	projected := w.beginFileProjection(m)
	w.finishFileProjection(projected)

	if *m.SelectedFile != "main.go" || *m.FileEditDraft != "edited main" {
		t.Fatalf("non-file pane activation cleared file mirror: selected=%q draft=%q", *m.SelectedFile, *m.FileEditDraft)
	}
	fileLeaf := w.Root.find(fileID)
	if fileLeaf == nil || fileLeaf.Files == nil {
		t.Fatalf("missing file leaf %q after split", fileID)
	}
	if fileLeaf.Files.Selected != "main.go" || w.legacyFileID != fileID {
		t.Fatalf("legacy file owner drifted to %q with fileLeaf.Files.Selected=%q (want %q)", w.legacyFileID, fileLeaf.Files.Selected, fileID)
	}
	if *m.DiffPath != "main.go" || w.legacyDiffID != diffPane.ID {
		t.Fatalf("diff projection failed to track active diff pane: path=%q legacyDiffID=%q", *m.DiffPath, w.legacyDiffID)
	}

	// Simulate an empty scalar mirror write while tab state still owns main.go.
	*m.SelectedFile = ""
	fileLeaf.Files.capture(m)
	if fileLeaf.Files.Selected != "main.go" {
		t.Fatalf("WorkspaceFileState.capture cleared Selected to %q while active tab remained main.go", fileLeaf.Files.Selected)
	}
}
