package session

import "testing"

func TestWorkspaceGroupCollapsedSourceDefaultsAndExplicitPresence(t *testing.T) {
	snap := ViewState{SelectedID: "active", ScratchCwd: "/scratch", WorkspaceLayout: WorkspaceLayoutState{GroupBy: "workspace"},
		Workspaces: []WorkspaceRow{{WorkspaceID: "current", Path: "/repo", SessionIDs: []string{"active"}}, {WorkspaceID: "other", Path: "/other"}, {WorkspaceID: "feature", Path: "/feature", SessionIDs: []string{"owner"}}},
		Items:      []Item{{SessionID: "active"}, {SessionID: "owner", Presentation: &SessionPresentation{Owner: "whaleisle"}}, {SessionID: "scratch", Cwd: "/scratch"}}}
	for key, want := range map[string]bool{"current": false, "other": true, "feature": false, "::scratch": true} {
		if got := WorkspaceGroupCollapsed(snap, key); got != want {
			t.Fatalf("default %s = %v want %v", key, got, want)
		}
	}
	snap.WorkspaceLayout.Collapsed = map[string]bool{"current": true, "other": false, "feature": true}
	for key, want := range snap.WorkspaceLayout.Collapsed {
		if got := WorkspaceGroupCollapsed(snap, key); got != want {
			t.Fatalf("explicit %s = %v want %v", key, got, want)
		}
	}
	snap.SelectedID = "scratch"
	if WorkspaceGroupCollapsed(snap, "::scratch") {
		t.Fatal("selected real scratch group remained closed")
	}
	snap.SelectedID = "orphan"
	snap.Items = append(snap.Items, Item{SessionID: "orphan", Cwd: "/not-scratch"})
	if !WorkspaceGroupCollapsed(snap, "::scratch") {
		t.Fatal("orphan selection opened scratch group")
	}
	snap.WorkspaceLayout.GroupBy = "flat"
	if WorkspaceGroupCollapsed(snap, "current") || WorkspaceGroupCollapsed(snap, "::flat") {
		t.Fatal("flat list acquired collapsible groups")
	}
}

func TestWorkspaceGroupCollapsedTreeParentsAndCurrentAncestors(t *testing.T) {
	snap := ViewState{SelectedID: "deep-session", WorkspaceLayout: WorkspaceLayoutState{GroupBy: "workspace-tree"}, Workspaces: []WorkspaceRow{
		{WorkspaceID: "root", Path: "/repo"}, {WorkspaceID: "middle", Path: "/repo/project"},
		{WorkspaceID: "leaf", Path: "/repo/project/child", SessionIDs: []string{"deep-session"}},
		{WorkspaceID: "unselected-parent", Path: "/other"}, {WorkspaceID: "unselected-leaf", Path: "/other/child"},
	}, Items: []Item{{SessionID: "deep-session"}}}
	for _, key := range []string{"root", "middle", "leaf", "unselected-parent"} {
		if WorkspaceGroupCollapsed(snap, key) {
			t.Fatalf("source tree owner/current ancestor %s stayed closed", key)
		}
	}
	if !WorkspaceGroupCollapsed(snap, "unselected-leaf") {
		t.Fatal("unselected leaf opened without an explicit choice")
	}
	snap.WorkspaceLayout.Collapsed = map[string]bool{"root": true}
	if !WorkspaceGroupCollapsed(snap, "root") {
		t.Fatal("auto-open overrode explicit ancestor collapse")
	}
}

func TestToggleWorkspaceGroupUsesResolvedDefaultOnFirstClick(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "active"}}
	s.selectedID = "active"
	s.workspaceMemberships = map[string]workspaceMembership{"current": {WorkspaceID: "current", SessionIDs: []string{"active"}}, "other": {WorkspaceID: "other"}}
	s.ToggleWorkspaceGroup("other")
	if collapsed, present := s.workspaceLayout.view.Collapsed["other"]; !present || collapsed {
		t.Fatal("first click did not explicitly open a default-closed group")
	}
	s.ToggleWorkspaceGroup("other")
	if !s.workspaceLayout.view.Collapsed["other"] {
		t.Fatal("second click did not close the group")
	}
	s.ToggleWorkspaceGroup("current")
	if !s.workspaceLayout.view.Collapsed["current"] {
		t.Fatal("first click did not close the default-open selected group")
	}
}
