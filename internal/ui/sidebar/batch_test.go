package sidebar

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestVisibleBatchTargetsRespectsCollapsedAndArchivedVisibility(t *testing.T) {
	snap := session.ViewState{WorkspaceReady: true, Items: []session.Item{{SessionID: "a", Title: "A"}, {SessionID: "b", Title: "B"}}, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", Title: "W", SessionIDs: []string{"a", "b"}}}, ArchivedSessionIDs: []string{"archived"}}
	snap.WorkspaceLayout.Collapsed = map[string]bool{"w": true}
	m := Model{Snapshot: snap, HideArchivedList: true}
	if got := VisibleBatchTargets(m); len(got) != 0 {
		t.Fatalf("collapsed rows visible: %v", got)
	}
	m.HideArchivedList = false
	if got := VisibleBatchTargets(m); len(got) != 0 {
		t.Fatalf("collapsed archive rows visible: %v", got)
	}
	archiveExpanded := true
	m.archiveExpanded = &archiveExpanded
	if got := VisibleBatchTargets(m); len(got) != 1 || got[0].ID != "archived" {
		t.Fatalf("archive rows: %v", got)
	}
	snap.WorkspaceLayout.Collapsed["w"] = false
	m.Snapshot = snap
	if got := VisibleBatchTargets(m); len(got) != 3 {
		t.Fatalf("expanded rows: %v", got)
	}
}
func TestVisibleBatchTargetsSearchExcludesArchive(t *testing.T) {
	query := "match"
	m := Model{SearchQuery: &query, Snapshot: session.ViewState{WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", SessionIDs: []string{"a", "b"}}}, Items: []session.Item{{SessionID: "a", Title: "match"}, {SessionID: "b", Title: "other"}}, ArchivedSessionIDs: []string{"archived"}}}
	if got := VisibleBatchTargets(m); len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("search selection: %v", got)
	}
}
