package sidebar

import (
	"reflect"
	"testing"

	"whaleisle/internal/store/session"
)

func TestFilterItemsByTitle(t *testing.T) {
	items := []session.Item{
		{SessionID: "a", Title: "Alpha"},
		{SessionID: "b", Title: "Beta", ParentID: "a"},
		{SessionID: "c", Title: "Gamma"},
	}
	got := filterItems(items, "beta")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (parent + child)", len(got))
	}
}

func TestSidebarItemsUsesRPCHits(t *testing.T) {
	snap := session.ViewState{
		Items: []session.Item{
			{SessionID: "a", Title: "Alpha"},
			{SessionID: "b", Title: "Beta", ParentID: "a"},
			{SessionID: "c", Title: "Gamma"},
		},
		Workspaces:     []session.WorkspaceRow{{WorkspaceID: "workspace", SessionIDs: []string{"a", "b", "c"}}},
		SearchRPCQuery: "q",
		SearchHitIDs:   map[string]bool{"b": true},
	}
	q := "q"
	got := sidebarItems(snap, &q)
	if len(got) != 2 {
		t.Fatalf("len = %d, want parent+child for hit b", len(got))
	}
}

func TestFilterItemsEmptyQuery(t *testing.T) {
	items := []session.Item{{SessionID: "a", Title: "A"}}
	got := filterItems(items, "  ")
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestSidebarOrdinaryVisibilityUsesAuthoritativeOwnership(t *testing.T) {
	items := []session.Item{
		{SessionID: "history", Title: "History", Cwd: "/workspace"},
		{SessionID: "fork", ParentID: "history", Cwd: "/workspace"},
		{SessionID: "scratch", Cwd: "/scratch"},
		{SessionID: "current-blank", Blank: true, Cwd: "/workspace"},
		{SessionID: "unused-blank", Blank: true, Cwd: "/workspace"},
		{SessionID: "child", Origin: "subagent", ParentID: "history", Cwd: "/workspace"},
		{SessionID: "foreign", Cwd: "/deleted"},
		{SessionID: "unaccounted-fork", ParentID: "history", Cwd: "/workspace"},
		{SessionID: "managed", Presentation: &session.SessionPresentation{Owner: "project"}, Cwd: "/workspace"},
		{SessionID: "renamed", Presentation: &session.SessionPresentation{Owner: "whaleisle", Title: "Renamed"}, Cwd: "/workspace"},
		{SessionID: "archived", Cwd: "/workspace"},
	}
	snap := session.ViewState{SelectedID: "current-blank", ScratchCwd: "/scratch", Items: items,
		Workspaces:         []session.WorkspaceRow{{WorkspaceID: "one", SessionIDs: []string{"history", "fork", "current-blank", "unused-blank", "child", "managed", "renamed", "archived", "missing"}}},
		ArchivedSessionIDs: []string{"archived", "foreign", "child", "unused-blank"},
	}
	ids := func(items []session.Item) []string {
		var out []string
		for _, item := range items {
			out = append(out, item.SessionID)
		}
		return out
	}
	before := append([]string(nil), snap.Workspaces[0].SessionIDs...)
	for _, tc := range []struct {
		filter string
		want   []string
	}{
		{"default", []string{"history", "fork", "scratch", "current-blank", "renamed"}},
		{"show", []string{"history", "fork", "scratch", "current-blank", "renamed", "archived"}},
		{"only", []string{"archived"}},
	} {
		if got := ids(sidebarItemsForArchiveFilter(snap, nil, tc.filter)); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("filter %s: %v want %v", tc.filter, got, tc.want)
		}
	}
	snap.SelectedID = "foreign"
	if got := ids(sidebarItemsForArchiveFilter(snap, nil, "show")); !reflect.DeepEqual(got, []string{"history", "fork", "scratch", "renamed", "archived"}) {
		t.Fatalf("selecting a foreign session bypassed account visibility: %v", got)
	}
	if !reflect.DeepEqual(snap.Workspaces[0].SessionIDs, before) || len(snap.ArchivedSessionIDs) != 4 {
		t.Fatal("visibility mutated account or archive membership")
	}
	snap.ScratchCwd = ""
	for _, item := range sidebarItemsForArchiveFilter(snap, nil, "show") {
		if item.SessionID == "scratch" {
			t.Fatal("scratch accepted without Host baseline cwd")
		}
	}
}
