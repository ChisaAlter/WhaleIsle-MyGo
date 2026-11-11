package session

import (
	"encoding/json"
	"reflect"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

func TestSidebarPinnedAcrossAccounts(t *testing.T) {
	snap := ViewState{SelectedID: "blank", ScratchCwd: "/scratch",
		Workspaces: []WorkspaceRow{
			{WorkspaceID: "alpha", Path: "/alpha", SessionIDs: []string{"a", "normal", "blank", "archived", "sub"}},
			{WorkspaceID: "beta", Path: "/beta", SessionIDs: []string{"b"}},
		},
		Items: []Item{{SessionID: "a"}, {SessionID: "b"}, {SessionID: "scratch", Cwd: "/scratch"},
			{SessionID: "normal"}, {SessionID: "blank", Blank: true}, {SessionID: "archived"},
			{SessionID: "sub", Origin: "subagent"}, {SessionID: "foreign", Cwd: "/unknown"}},
		PinnedSessionIDs:   []string{"b", "scratch", "a", "a", "missing", "blank", "archived", "sub", "foreign"},
		ArchivedSessionIDs: []string{"archived"},
	}
	for _, mode := range []string{"workspace", "workspace-tree"} {
		snap.WorkspaceLayout.GroupBy = mode
		groups := BuildSidebarWorkspaceGroups(snap, snap.Items)
		if groups[0].Key != PinnedWorkspaceGroup || !reflect.DeepEqual(groups[0].Workspace.SessionIDs, []string{"b", "scratch", "a"}) {
			t.Fatalf("%s pinned order: %+v", mode, groups[0])
		}
		seen := map[string]string{}
		for _, group := range groups {
			for _, item := range group.Items {
				if prev := seen[item.SessionID]; prev != "" {
					t.Fatalf("duplicate %s in %s and %s", item.SessionID, prev, group.Key)
				}
				seen[item.SessionID] = group.Key
			}
		}
		want := map[string]string{"a": PinnedWorkspaceGroup, "b": PinnedWorkspaceGroup, "scratch": PinnedWorkspaceGroup, "normal": "alpha", "blank": "alpha", "archived": "alpha"}
		if !reflect.DeepEqual(seen, want) {
			t.Fatalf("groups=%v want %v", seen, want)
		}
		if ordinary := BuildWorkspaceGroups(snap, snap.Items); ordinary[0].Kind != "workspace" || len(ordinary[0].Items) != 4 {
			t.Fatalf("ordinary account callers changed: %+v", ordinary)
		}
	}
	snap.PinnedSessionIDs = []string{"b"}
	groups := BuildSidebarWorkspaceGroups(snap, snap.Items)
	if groups[1].Items[0].SessionID != "a" || groups[len(groups)-1].Items[0].SessionID != "scratch" {
		t.Fatal("unpin did not return rows to their original accounts", groups)
	}
	snap.ArchivedSessionIDs = append(snap.ArchivedSessionIDs, "b")
	if groups := BuildSidebarWorkspaceGroups(snap, snap.Items); groups[0].Kind == "pinned" {
		t.Fatal("archived pin remained hoisted")
	}
	snap.WorkspaceLayout.GroupBy = "flat"
	if !reflect.DeepEqual(BuildSidebarWorkspaceGroups(snap, snap.Items), BuildWorkspaceGroups(snap, snap.Items)) {
		t.Fatal("flat view changed")
	}
}

func TestPinnedGroupCollapsePersistsAcrossRestart(t *testing.T) {
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	attach := func(s *Store) {
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
		s.Attach(client, nil)
		s.BindWorkspacePreferences(client, "pinned-host", repo)
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"alpha","path":"/alpha","sessionIds":["a"]}],"pinnedSessionIds":["a"],"archivedSessionIds":[]}}`))
	}
	s := New()
	attach(s)
	if WorkspaceGroupCollapsed(s.Snapshot(), PinnedWorkspaceGroup) {
		t.Fatal("pins default collapsed")
	}
	s.ToggleWorkspaceGroup(PinnedWorkspaceGroup)
	restarted := New()
	attach(restarted)
	if !WorkspaceGroupCollapsed(restarted.Snapshot(), PinnedWorkspaceGroup) {
		t.Fatal("collapsed pins lost on restart")
	}
	restarted.ToggleWorkspaceGroup(PinnedWorkspaceGroup)
	reopened := New()
	attach(reopened)
	if WorkspaceGroupCollapsed(reopened.Snapshot(), PinnedWorkspaceGroup) {
		t.Fatal("explicit open lost on restart")
	}
}
