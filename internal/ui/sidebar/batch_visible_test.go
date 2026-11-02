package sidebar

import (
	"fmt"
	"testing"

	"whaleisle/internal/store/session"
)

func TestVisibleBatchTargetsUsesRenderedDefaultFoldsAndWatermark(t *testing.T) {
	snap := session.ViewState{WorkspaceReady: true, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7}, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", Title: "Workspace"}}}
	for n := 0; n < 8; n++ {
		id := fmt.Sprintf("row-%d", n)
		snap.Items = append(snap.Items, session.Item{SessionID: id, Title: id})
		snap.Workspaces[0].SessionIDs = append(snap.Workspaces[0].SessionIDs, id)
	}
	m := Model{Snapshot: snap}
	if got := VisibleBatchTargets(m); len(got) != 0 {
		t.Fatalf("default-closed workspace selected hidden rows: %v", got)
	}
	m.Snapshot.WorkspaceLayout.Collapsed = map[string]bool{"w": false}
	if got := VisibleBatchTargets(m); len(got) != 5 || got[4].ID != "row-4" {
		t.Fatalf("baseline watermark selected overflow rows: %v", got)
	}
	m.workspaceWatermarks = &workspaceWatermarkState{HostGeneration: 7, Marks: map[string]int{"w": 7}}
	if got := VisibleBatchTargets(m); len(got) != 7 || got[6].ID != "row-6" {
		t.Fatalf("expanded watermark diverged from rendered targets: %v", got)
	}
	m.workspaceWatermarks.HostGeneration = 6
	if got := VisibleBatchTargets(m); len(got) != 5 {
		t.Fatalf("prior Host watermark leaked into selection: %v", got)
	}
	m.workspaceWatermarks = nil
	m.Snapshot.SelectedID = "row-6"
	if got := VisibleBatchTargets(m); len(got) != 7 || got[6].ID != "row-6" {
		t.Fatalf("selected reveal and visible selection differ: %v", got)
	}
}
