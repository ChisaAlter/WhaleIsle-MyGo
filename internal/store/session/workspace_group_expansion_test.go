package session

import (
	"testing"

	"whaleisle/internal/config"
)

func TestWorkspaceGroupExpansionMaterializesExplicitOpenAndPreservesPresence(t *testing.T) {
	for _, id := range []string{"target", "::scratch"} {
		for _, state := range []string{"nil-map", "absent", "closed", "open"} {
			t.Run(id+"/"+state, func(t *testing.T) {
				workspace := WorkspaceRow{WorkspaceID: "target", Path: "/target"}
				key := workspacePreferenceKey(workspace)
				if id == "::scratch" {
					key = id
				}
				scope := preferenceHash("host", "expansion-host", "/scratch")
				prefs := config.WorkspaceNavigationPreferences{Collapsed: map[string]bool{"unrelated": true}}
				if state == "closed" || state == "open" {
					prefs.Collapsed[key] = state == "closed"
				}
				repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{scope: prefs}}
				s := New()
				recoveryAttach(t, s, repo, "expansion-host", workspace)
				if state == "nil-map" {
					s.mu.Lock()
					s.workspaceLayout.view.Collapsed = nil
					s.invalidateViewLocked()
					s.mu.Unlock()
				}
				before := s.Snapshot()
				generation := before.WorkspaceLayout.HostGeneration
				_, present := before.WorkspaceLayout.Collapsed[id]
				if present != (state == "open" || state == "closed") {
					t.Fatal("snapshot changed initial explicit key presence")
				}
				if s.ExpandWorkspaceGroupForHost(generation+1, id) || repo.writes != 0 {
					t.Fatal("stale Host materialized or persisted expansion")
				}
				if !s.ExpandWorkspaceGroupForHost(generation, id) {
					t.Fatal("current Host could not address a valid group")
				}
				after := s.Snapshot()
				if collapsed, explicit := after.WorkspaceLayout.Collapsed[id]; !explicit || collapsed || WorkspaceGroupCollapsed(after, id) {
					t.Fatalf("expansion did not publish an explicit open key: %+v", after.WorkspaceLayout.Collapsed)
				}
				wantWrites := 1
				if state == "open" {
					wantWrites = 0
				}
				if repo.writes != wantWrites {
					t.Fatalf("writes=%d, want %d", repo.writes, wantWrites)
				}
				if value, explicit := repo.values[scope].Collapsed[key]; !explicit || value || !repo.values[scope].Collapsed["unrelated"] {
					t.Fatal("persistence lost explicit false or unrelated state")
				}
				// Returned snapshots are independently writable. A cached snapshot
				// and its clone must retain the stored false key after mutation.
				delete(after.WorkspaceLayout.Collapsed, id)
				if value, explicit := s.Snapshot().WorkspaceLayout.Collapsed[id]; !explicit || value {
					t.Fatal("snapshot clone lost explicit false presence")
				}
				revision := s.viewRev
				if !s.ExpandWorkspaceGroupForHost(generation, id) || s.viewRev != revision || repo.writes != wantWrites {
					t.Fatal("repeated explicit-open expansion caused a write or invalidation")
				}
				restarted := New()
				recoveryAttach(t, restarted, repo, "expansion-host", workspace)
				if value, explicit := restarted.Snapshot().WorkspaceLayout.Collapsed[id]; !explicit || value {
					t.Fatal("restart did not restore explicit open presence")
				}
			})
		}
	}
}
