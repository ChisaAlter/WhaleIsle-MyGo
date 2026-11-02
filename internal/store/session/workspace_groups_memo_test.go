package session

import (
	"reflect"
	"testing"
)

func TestBuildWorkspaceGroupsReusesStoreRevision(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.items = []Item{{SessionID: "a", Title: "A", UpdatedAt: 2}, {SessionID: "b", Title: "B", UpdatedAt: 1}, {SessionID: "c", Title: "C", UpdatedAt: 3}}
	s.workspaceMemberships = map[string]workspaceMembership{"w": {WorkspaceID: "w", SessionIDs: []string{"a", "b"}}, "v": {WorkspaceID: "v", SessionIDs: []string{"c"}}}
	s.invalidateViewLocked()
	s.mu.Unlock()
	snap := s.Snapshot()
	first := BuildWorkspaceGroups(snap, snap.Items)
	if len(first) == 0 {
		t.Fatal("no groups")
	}
	if again := BuildWorkspaceGroups(snap, snap.Items); &again[0] != &first[0] {
		t.Fatal("the same revision and items regrouped")
	}
	subset := BuildWorkspaceGroups(snap, snap.Items[:1])
	if reflect.DeepEqual(subset, first) {
		t.Fatal("a different item set reused the full grouping")
	}
	if !reflect.DeepEqual(subset, buildWorkspaceGroups(snap, snap.Items[:1])) {
		t.Fatal("memoized subset differs from a direct build")
	}
	if again := BuildWorkspaceGroups(snap, snap.Items); &again[0] != &first[0] {
		t.Fatal("the full grouping was evicted by one other item set")
	}
	s.SetLastAction("changed")
	next := s.Snapshot()
	rebuilt := BuildWorkspaceGroups(next, next.Items)
	if &rebuilt[0] == &first[0] || !reflect.DeepEqual(rebuilt, first) {
		t.Fatal("a new revision did not regroup to the same result")
	}
	hand := ViewState{Items: snap.Items, Workspaces: snap.Workspaces}
	if a, b := BuildWorkspaceGroups(hand, hand.Items), BuildWorkspaceGroups(hand, hand.Items); len(a) == 0 || &a[0] == &b[0] {
		t.Fatal("a hand-built view was memoized")
	}
}
