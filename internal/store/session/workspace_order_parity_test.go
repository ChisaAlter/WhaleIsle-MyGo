package session

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

func workspaceOrderingFixture(t *testing.T) (*Store, *fixtureWorkspacePreferences) {
	t.Helper()
	s := New()
	s.client = &dsh.Client{}
	s.eventsGeneration = 7
	s.workspaceReady = true
	s.scratchCwd = "/scratch"
	s.workspaceOrder = []string{"one", "two"}
	s.workspaceMemberships = map[string]workspaceMembership{
		"one": {WorkspaceID: "one", Path: "/one", SessionIDs: []string{"a", "hidden", "b"}},
		"two": {WorkspaceID: "two", Path: "/two", SessionIDs: []string{"c", "d"}},
	}
	s.items = []Item{
		{SessionID: "a", UpdatedAt: 20}, {SessionID: "hidden", UpdatedAt: 15}, {SessionID: "b", UpdatedAt: 10},
		{SessionID: "c", UpdatedAt: 5}, {SessionID: "d", UpdatedAt: 30},
		{SessionID: "scratch", Cwd: "/scratch", UpdatedAt: 8}, {SessionID: "foreign", Cwd: "/foreign", UpdatedAt: 9},
	}
	s.archivedSessionIDs = []string{"hidden"}
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	s.BindWorkspacePreferences(s.client, "ordering-host", repo)
	return s, repo
}

func TestWorkspaceOrderingManualFreezesAllAccountsAndUpdatedClears(t *testing.T) {
	s, repo := workspaceOrderingFixture(t)
	if s.Snapshot().WorkspaceLayout.OrderBy != "updated" {
		t.Fatal("browser default is not recently updated")
	}
	s.SetWorkspaceOrdering("manual")
	want := map[string][]string{
		"one": {"a", "hidden", "b"}, "two": {"d", "c"},
		"::scratch": {"foreign", "scratch"}, "::flat": {"d", "a", "hidden", "b", "scratch", "c"},
	}
	if got := s.Snapshot().WorkspaceLayout; got.OrderBy != "manual" || !reflect.DeepEqual(got.SessionOrders, want) {
		t.Fatalf("manual did not snapshot every complete account: %+v", got)
	}
	if repo.writes != 1 || !reflect.DeepEqual(repo.values[s.workspacePreferences.scope].SessionOrders, want) {
		t.Fatal("mode and all accounts were not persisted together")
	}
	s.items[0].UpdatedAt = 999
	s.SetWorkspaceOrdering("manual")
	if repo.writes != 1 || !reflect.DeepEqual(s.Snapshot().WorkspaceLayout.SessionOrders, want) {
		t.Fatal("same-mode selection re-snapshotted saved positions")
	}
	s.SetWorkspaceOrdering("updated")
	if got := s.Snapshot().WorkspaceLayout; got.OrderBy != "updated" || len(got.SessionOrders) != 0 || len(repo.values[s.workspacePreferences.scope].SessionOrders) != 0 || repo.writes != 2 {
		t.Fatalf("updated did not atomically clear all orders: %+v", got)
	}
}

func TestWorkspaceOrderingUpdatedDragLocalPersistenceAndHostFence(t *testing.T) {
	s, repo := workspaceOrderingFixture(t)
	hostBefore := s.Snapshot().Workspaces
	s.InsertSessionBeforeForHost(7, "one", "b", "a")
	got := s.Snapshot()
	if got.WorkspaceLayout.OrderBy != "manual" || !slices.Equal(got.WorkspaceLayout.SessionOrders["one"], []string{"b", "a", "hidden"}) || !slices.Equal(got.WorkspaceLayout.SessionOrders["two"], []string{"d", "c"}) {
		t.Fatal("updated drag lost target or untouched account order", got.WorkspaceLayout)
	}
	if !reflect.DeepEqual(got.Workspaces, hostBefore) || got.WorkspaceLayout.Reordering || repo.writes != 1 {
		t.Fatal("local drag changed Host membership, went asynchronous, or split its preference write")
	}
	s.InsertSessionBeforeForHost(6, "one", "a", "b")
	s.InsertSessionBeforeForHost(7, "two", "a", "d")
	s.InsertLocalSessionBefore(7, "::scratch", "foreign", "scratch")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.InsertSessionBefore(ctx, "one", "a", "b")
	if repo.writes != 1 || !reflect.DeepEqual(s.Snapshot().WorkspaceLayout.SessionOrders, got.WorkspaceLayout.SessionOrders) {
		t.Fatal("stale Host, cross-account, invisible source, or canceled gesture changed order")
	}
	other := New()
	other.client = &dsh.Client{}
	other.workspaceReady = true
	other.scratchCwd = "/scratch"
	other.BindWorkspacePreferences(other.client, "other-host", repo)
	if len(other.Snapshot().WorkspaceLayout.SessionOrders) != 0 {
		t.Fatal("manual order leaked into another Host")
	}
}

func TestWorkspaceOrderingPinsBlankHiddenAnchorsAndFailedSave(t *testing.T) {
	s, repo := workspaceOrderingFixture(t)
	s.workspaceMemberships["one"] = workspaceMembership{WorkspaceID: "one", Path: "/one", SessionIDs: []string{"blank", "p1", "p2", "hidden", "a", "b"}}
	s.selectedID = "blank"
	s.pinnedSessionIDs = []string{"p1", "p2"}
	s.items = append(s.items, Item{SessionID: "blank", Blank: true}, Item{SessionID: "p1", UpdatedAt: 50}, Item{SessionID: "p2", UpdatedAt: 40})
	s.InsertSessionBeforeForHost(7, "one", "blank", "p1")
	if repo.writes != 0 {
		t.Fatal("blank placeholder was draggable")
	}
	// A UI drop after p1 is represented by the next complete-account slot,
	// which can be hidden. It must not be mistaken for the visible target.
	s.InsertSessionBeforeForHost(7, "one", "p2", "p1")
	if !slices.Equal(s.Snapshot().WorkspaceLayout.SessionOrders["one"], []string{"blank", "p2", "p1", "a", "hidden", "b"}) {
		t.Fatal("pin reorder or selected blank position lost", s.Snapshot().WorkspaceLayout.SessionOrders)
	}
	repo.fail = true
	s.SetWorkspaceOrdering("updated")
	if got := s.Snapshot().WorkspaceLayout; got.OrderBy != "updated" || len(got.SessionOrders) != 0 || got.PreferenceError == "" {
		t.Fatal("failed preference write lost local choice or error", got)
	}
	repo.fail = false
	s.RetryWorkspacePreferences()
	if got := repo.values[s.workspacePreferences.scope]; got.OrderBy != "updated" || len(got.SessionOrders) != 0 || s.Snapshot().WorkspaceLayout.PreferenceError != "" {
		t.Fatal("retry resurrected cleared manual positions", got)
	}
}
