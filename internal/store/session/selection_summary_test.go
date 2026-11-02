package session

import (
	"errors"
	"testing"
)

func TestSelectionSummaryMatchesSnapshot(t *testing.T) {
	s := benchmarkStore(30)
	check := func(name string) {
		t.Helper()
		snap, got := s.Snapshot(), s.SelectionSummary()
		want := SelectionSummary{SelectedID: snap.SelectedID, SelectedCwd: snap.SelectedCwd, ComposerHostGeneration: snap.ComposerHostGeneration, ClientReady: snap.ClientReady, ListLoading: snap.ListLoading, ListErr: snap.ListErr, HasWorkspaces: len(snap.Workspaces) != 0}
		for _, it := range snap.Items {
			if it.SessionID == snap.SelectedID {
				if it.Presentation != nil {
					want.SelectedComposer = it.Presentation.Composer
				}
				if it.SessionID != "" {
					want.SelectedListed, want.SelectedTitle, want.SelectedRunning = true, it.Title, it.Running
				}
				break
			}
		}
		if got != want {
			t.Fatalf("%s: summary %+v, snapshot %+v", name, got, want)
		}
		if s.WorkspaceNavigationPending() != (snap.WorkspaceNavigation != nil) {
			t.Fatalf("%s: navigation pending %v with snapshot request %v", name, s.WorkspaceNavigationPending(), snap.WorkspaceNavigation)
		}
	}
	check("initial")
	s.mu.Lock()
	s.items = []Item{{SessionID: "a", Title: "A", Cwd: `C:\a`}, {SessionID: "b", Title: "B", Cwd: `C:\b`, Running: true, Presentation: &SessionPresentation{Composer: "managed"}}}
	s.selectedID = "b"
	s.composerHostGeneration = 7
	s.listLoading = true
	s.listErr = errors.New("list failed")
	s.workspaceMemberships = map[string]workspaceMembership{"w": {}}
	s.invalidateViewLocked()
	s.mu.Unlock()
	check("listed")
	s.mu.Lock()
	s.selectedID = "missing"
	s.listLoading, s.listErr = false, nil
	s.workspaceMemberships = nil
	s.invalidateViewLocked()
	s.mu.Unlock()
	check("unlisted")
}

func TestSelectionSummaryDoesNotProject(t *testing.T) {
	s := benchmarkStore(2000)
	builds, _ := s.SnapshotStats()
	for i := 0; i < 10; i++ {
		_ = s.SelectionSummary()
		_ = s.WorkspaceNavigationPending()
	}
	if after, _ := s.SnapshotStats(); after != builds {
		t.Fatalf("summary projected the view: builds %d -> %d", builds, after)
	}
	if allocs := testing.AllocsPerRun(20, func() { _ = s.SelectionSummary() }); allocs > 0 {
		t.Fatalf("summary allocated %.0f times per call", allocs)
	}
}
