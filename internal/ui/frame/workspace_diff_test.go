package frame

import (
	"testing"
	"whaleisle/internal/ui/diffsurface"
)

func TestDiffPaneIndependentState(t *testing.T) {
	kind := "diff"
	w := &WorkspacePanes{}
	w.syncRoute(kind)
	path, text, message := "a.go", "patch A", ""
	workbench := diffsurface.Workbench{Root: "repo", Loaded: true, BaseRef: "main", Message: "private commit draft", ReviewToken: "review-token"}
	m := Model{RightPaneOpen: true, RightPaneTab: &kind, Sizes: &LayoutSizes{Workspace: w}, WorkspaceRoot: "repo", DiffPath: &path, DiffText: &text, DiffErr: &message, GitWorkbench: &workbench}
	w.beginDiffProjection(m)
	firstID := w.Active
	SplitWorkspacePane(m, false)
	left, right := w.Root.First, w.Root.Second
	if left.ID != firstID || right.Kind != "diff" || right.Diff == nil {
		t.Fatal("missing diff split")
	}
	if left.Diff == right.Diff || right.Diff.Path != "a.go" || right.Diff.Text != "patch A" {
		t.Fatal("diff presentation not copied independently")
	}
	if right.Diff.Workbench.BaseRef != "main" || right.Diff.Workbench.Message != "" || right.Diff.Workbench.ReviewToken != "" {
		t.Fatal("copied private operation state")
	}
	local := right.Diff.model(m)
	*local.DiffPath = "b.go"
	*local.DiffText = "patch B"
	local.GitWorkbench.BaseRef = "other"
	if left.Diff.Path != "a.go" || left.Diff.Text != "patch A" || left.Diff.Workbench.BaseRef != "main" {
		t.Fatal("second pane changed first")
	}
	w.finishDiffProjection(m)
	if path != "b.go" || text != "patch B" || workbench.BaseRef != "other" {
		t.Fatal("focused projection not published")
	}
	otherRoot := m
	otherRoot.WorkspaceRoot = "other-repository"
	right.Diff.model(otherRoot)
	if right.Diff.Path != "" || right.Diff.Text != "" {
		t.Fatal("old repository leaked into new scope")
	}
	right.Diff.Path = "b.go"
	restored, err := RestoreWorkspacePanes(w.Export())
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root.First.Diff.Path != "a.go" || restored.Root.Second.Diff.Path != "b.go" || restored.Root.First.Diff.Text != "" || restored.Root.First.Diff.Workbench.ReviewToken != "" {
		t.Fatal("layout mixed selectors or persisted patch/operation")
	}
}
