package session

import (
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

func TestWorkspaceLocalOrderPersistence(t *testing.T) {
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	s := New()
	s.client = &dsh.Client{}
	s.workspaceReady = true
	s.scratchCwd = "/scratch"
	s.items = []Item{{SessionID: "a", Cwd: "/scratch"}, {SessionID: "hidden", Cwd: "/scratch"}, {SessionID: "b", Cwd: "/scratch"}}
	s.BindWorkspacePreferences(s.client, "host-a", repo)
	s.SetWorkspaceGrouping("flat")
	generation := s.Snapshot().WorkspaceLayout.HostGeneration
	s.InsertLocalSessionBefore(generation, "::flat", "b", "a")
	got := BuildWorkspaceGroups(s.Snapshot(), []Item{{SessionID: "a", Cwd: "/scratch"}, {SessionID: "b", Cwd: "/scratch"}})[0].Workspace.SessionIDs
	if len(got) != 3 || got[0] != "b" || got[2] != "hidden" {
		t.Fatalf("full order lost: %v", got)
	}
	scope := s.workspacePreferences.scope
	if len(repo.values[scope].SessionOrders["::flat"]) != 3 {
		t.Fatal("not persisted")
	}
	s.InsertLocalSessionBefore(generation+1, "::flat", "a", "")
	if s.Snapshot().WorkspaceLayout.SessionOrders["::flat"][0] != "b" {
		t.Fatal("stale Host moved")
	}
	restarted := New()
	restarted.client = &dsh.Client{}
	restarted.workspaceReady = true
	restarted.scratchCwd = "/scratch"
	restarted.BindWorkspacePreferences(restarted.client, "host-a", repo)
	if restarted.Snapshot().WorkspaceLayout.SessionOrders["::flat"][0] != "b" {
		t.Fatal("restart lost order")
	}
	other := New()
	other.client = &dsh.Client{}
	other.workspaceReady = true
	other.scratchCwd = "/scratch"
	other.BindWorkspacePreferences(other.client, "host-b", repo)
	if len(other.Snapshot().WorkspaceLayout.SessionOrders) != 0 {
		t.Fatal("leaked Host")
	}
}
