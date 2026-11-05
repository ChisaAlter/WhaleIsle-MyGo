package frame

import (
	"encoding/json"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func reviewFixture() session.ViewState {
	return session.ViewState{SelectedID: "s", TurnFiles: session.TurnFilesSnapshot{HostLabel: "https://host", Scope: session.TurnFileScope{SessionID: "s", Binding: "bound", Generation: 2}}, Messages: []session.Message{
		{Raw: json.RawMessage(`{"type":"workspace/changes","seq":3,"data":{"turn":1}}`)},
		{Raw: json.RawMessage(`{"type":"turn/end","seq":4,"data":{"turn":1}}`)},
	}}
}
func TestChangesReviewScopeSplitRestoreClose(t *testing.T) {
	snap := reviewFixture()
	target := session.TurnFileTarget{Kind: "changes", Turn: 1, Seq: 3, Index: -1}
	w := &WorkspacePanes{}
	if !w.OpenChangesReview(snap, target) {
		t.Fatal("recorded turn not opened")
	}
	first := w.Root
	if first.Kind != "changes-review" || first.ChangesReview.Split || first.ChangesReview.Target.Index != 0 {
		t.Fatal("wrong defaults")
	}
	tab := "changes-review"
	SplitWorkspacePane(Model{RightPaneOpen: true, RightPaneTab: &tab, Sizes: &LayoutSizes{Workspace: w}}, false)
	left, right := w.Root.First, w.Root.Second
	if left == nil || right == nil || left.ChangesReview == right.ChangesReview || right.ChangesReview == nil {
		t.Fatal("missing independent review split")
	}
	right.ChangesReview.Target.Index = 8
	right.ChangesReview.Wrap = true
	if left.ChangesReview.Target.Index != 0 || left.ChangesReview.Wrap {
		t.Fatal("presentation leaked")
	}
	raw := w.Export()
	if strings.Contains(string(raw), "bound") {
		t.Fatal("persisted runtime authority")
	}
	restored, err := RestoreWorkspacePanes(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := restored.Root.Second.ChangesReview
	if !r.Wrap || r.Target.Index != 8 || !r.bind(snap) {
		t.Fatal("selection did not restore through validated scope")
	}
	foreign := snap
	foreign.TurnFiles.HostLabel = "https://other"
	if r.bind(foreign) {
		t.Fatal("accepted foreign Host")
	}
	foreign = snap
	foreign.TurnFiles.Scope.Generation++
	if r.bind(foreign) {
		t.Fatal("accepted stale generation")
	}
	foreign = snap
	foreign.Messages = nil
	if r.bind(foreign) {
		t.Fatal("accepted revoked event")
	}
	if !restored.close(restored.Root.Second.ID) || restored.Root.Kind != "changes-review" {
		t.Fatal("close lost sibling")
	}
	// Reopening from the card selects the existing turn and changes only its file.
	target.Index = 1
	if !restored.OpenChangesReview(snap, target) || restored.Root.ChangesReview.Target.Index != 1 {
		t.Fatal("reopen failed")
	}
}
func TestChangesReviewPatchLimitAndSingleSided(t *testing.T) {
	content := session.TurnFileDiff{Hunks: []session.TurnDiffHunk{{OldStart: 1, NewStart: 1, Lines: []string{" same", "+new"}}}}
	patch, truncated, add, del := reviewPatch(content)
	if patch == "" || truncated || !add || del {
		t.Fatal("one-sided classification")
	}
	content.Hunks[0].Lines = make([]string, 5001)
	for i := range content.Hunks[0].Lines {
		content.Hunks[0].Lines[i] = "+x"
	}
	patch, truncated, _, _ = reviewPatch(content)
	if !truncated || strings.Count(patch, "+x\n") != 5000 {
		t.Fatal("unbounded comparison")
	}
}
