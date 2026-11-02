package session

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

func TestWorkspaceArchivedFilterRestartsAndSeparatesHostAndScratchScopes(t *testing.T) {
	a, b := preferenceHash("host", "host-a", "/scratch"), preferenceHash("host", "host-b", "/scratch")
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{})
	s := New()
	recoveryAttach(t, s, repo, "host-a")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "default" || repo.writes != 0 {
		t.Fatal("missing preference did not default without a save")
	}
	s.SetWorkspaceArchivedFilter("show")
	if got := recoveryRead(t, path, a); got.ArchivedFilter != "show" {
		t.Fatal("filter not persisted", got)
	}
	s.SetWorkspaceCwd("/another-selected-directory")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "show" {
		t.Fatal("selected directory reset Host-wide filter")
	}
	restarted := New()
	recoveryAttach(t, restarted, repo, "host-a")
	if restarted.Snapshot().WorkspaceLayout.ArchivedFilter != "show" {
		t.Fatal("restart lost filter")
	}
	recoveryAttach(t, s, repo, "host-b")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "default" {
		t.Fatal("new Host borrowed archive preference")
	}
	s.SetWorkspaceArchivedFilter("only")
	recoveryAttach(t, s, repo, "host-a")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "show" || recoveryRead(t, path, b).ArchivedFilter != "only" {
		t.Fatal("Host round trip crossed filters")
	}
	s.applyWorkspaceFollowFrame([]byte(`{"type":"baseline","value":{"scratchCwd":"/other/scratch","items":[],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "default" {
		t.Fatal("other scratch namespace borrowed filter")
	}
	s.SetWorkspaceArchivedFilter("only")
	recoveryBaseline(t, s)
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "show" {
		t.Fatal("scratch scope return lost filter")
	}
}

func TestWorkspaceArchivedFilterFailedReadEditsOnlyChosenField(t *testing.T) {
	for _, chosen := range []string{"default", "show", "only"} {
		t.Run(chosen, func(t *testing.T) {
			scope, otherScope := preferenceHash("host", "host-a", "/scratch"), preferenceHash("host", "host-b", "/scratch")
			initial := config.WorkspaceNavigationPreferences{ArchivedFilter: "only", GroupBy: "workspace-tree", OrderBy: "updated", Collapsed: map[string]bool{"::scratch": true}, SessionOrders: map[string][]string{"::flat": {"b", "a"}}}
			other := config.WorkspaceNavigationPreferences{ArchivedFilter: "show", GroupBy: "flat"}
			repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial, otherScope: other})
			if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
				t.Fatal(err)
			}
			s := New()
			recoveryAttach(t, s, repo, "host-a")
			s.SetWorkspaceArchivedFilter(chosen)
			if got := s.Snapshot().WorkspaceLayout; got.ArchivedFilter != chosen || got.PreferenceError == "" || repo.writes != 0 {
				t.Fatal("read failure lost edit or blindly saved defaults", got)
			}
			s.RetryWorkspacePreferences()
			if repo.writes != 0 || s.Snapshot().WorkspaceLayout.ArchivedFilter != chosen {
				t.Fatal("unreadable retry changed edit")
			}
			latest := cloneWorkspacePreferences(initial)
			latest.GroupBy, latest.OrderBy, latest.ArchivedFilter = "flat", "manual", "show"
			latest.Collapsed["::other"] = true
			latest.SessionOrders["::flat"] = []string{"external", "b", "a"}
			recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: latest, otherScope: other})
			s.RetryWorkspacePreferences()
			want := cloneWorkspacePreferences(latest)
			want.ArchivedFilter = chosen
			if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, want) {
				t.Fatalf("recovery overwrote other fields: got=%+v want=%+v", got, want)
			}
			if !reflect.DeepEqual(recoveryRead(t, path, otherScope), other) {
				t.Fatal("recovery overwrote other Host")
			}
			if got := s.Snapshot().WorkspaceLayout; got.ArchivedFilter != chosen || got.GroupBy != latest.GroupBy || got.OrderBy != latest.OrderBy || got.PreferenceError != "" {
				t.Fatal("recovered view disagrees with saved preferences", got)
			}
			restarted := New()
			recoveryAttach(t, restarted, repo, "host-a")
			if restarted.Snapshot().WorkspaceLayout.ArchivedFilter != chosen {
				t.Fatal("restart lost recovered filter")
			}
		})
	}
}

func TestWorkspaceArchivedFilterEarlyAndFailedReadEditsSurviveScopeReturn(t *testing.T) {
	scopeA, scopeB := preferenceHash("host", "host-a", "/scratch"), preferenceHash("host", "host-b", "/scratch")
	a := config.WorkspaceNavigationPreferences{ArchivedFilter: "only", GroupBy: "flat", OrderBy: "updated"}
	b := config.WorkspaceNavigationPreferences{ArchivedFilter: "show", GroupBy: "workspace-tree"}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scopeA: a, scopeB: b})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	s.Attach(client, nil)
	s.BindWorkspacePreferences(client, "host-a", repo)
	s.SetWorkspaceArchivedFilter("default")
	if repo.reads != 0 || repo.writes != 0 {
		t.Fatal("unready Host accessed preferences")
	}
	recoveryBaseline(t, s)
	if got := s.Snapshot().WorkspaceLayout; got.ArchivedFilter != "default" || got.PreferenceError == "" {
		t.Fatal("early explicit default lost", got)
	}
	recoveryAttach(t, s, repo, "host-b")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "default" {
		t.Fatal("unread Host inherited pending archive choice")
	}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scopeA: a, scopeB: b})
	recoveryAttach(t, s, repo, "host-a")
	if got := recoveryRead(t, path, scopeA); got.ArchivedFilter != "default" || got.GroupBy != "flat" || got.OrderBy != "updated" {
		t.Fatal("scope return lost explicit default or untouched fields", got)
	}
	recoveryAttach(t, s, repo, "host-b")
	if s.Snapshot().WorkspaceLayout.ArchivedFilter != "show" || !reflect.DeepEqual(recoveryRead(t, path, scopeB), b) {
		t.Fatal("scope A changed scope B")
	}
}

func TestWorkspaceArchivedFilterDoesNotMaterializeFallbackDuringOtherEdit(t *testing.T) {
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{ArchivedFilter: "only", GroupBy: "flat", OrderBy: "updated"}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	recoveryAttach(t, s, repo, "host-a")
	s.SetWorkspaceOrdering("manual")
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s.RetryWorkspacePreferences()
	if got := recoveryRead(t, path, scope); got.ArchivedFilter != "only" || got.OrderBy != "manual" || got.GroupBy != "flat" {
		t.Fatal("untouched archive preference replaced by fallback", got)
	}
	// Invalid future values remain untouched on disk until an explicit choice.
	initial.ArchivedFilter = "future-filter"
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	restarted := New()
	recoveryAttach(t, restarted, repo, "host-a")
	if restarted.Snapshot().WorkspaceLayout.ArchivedFilter != "default" {
		t.Fatal("invalid archive preference leaked into view")
	}
	before := repo.writes
	restarted.SetWorkspaceArchivedFilter("bad-filter")
	if repo.writes != before || restarted.Snapshot().WorkspaceLayout.ArchivedFilter != "default" {
		t.Fatal("invalid explicit choice accepted")
	}
}

func TestWorkspaceArchivedFilterConcurrentFieldsMergeAndConflictStaysVisible(t *testing.T) {
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{ArchivedFilter: "default", GroupBy: "workspace", OrderBy: "manual"}
	repoA, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	otherStore, err := config.Open(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	repoB := &recoveryWorkspacePreferences{store: otherStore, observed: map[string]config.WorkspaceNavigationPreferences{}}
	a, b := New(), New()
	recoveryAttach(t, a, repoA, "host-a")
	recoveryAttach(t, b, repoB, "host-a")
	a.SetWorkspaceArchivedFilter("show")
	b.SetWorkspaceGrouping("flat")
	b.SetWorkspaceOrdering("updated")
	if got := recoveryRead(t, path, scope); got.ArchivedFilter != "show" || got.GroupBy != "flat" || got.OrderBy != "updated" {
		t.Fatal("independent windows lost changes", got)
	}
	// A fresh window observes show, while A still has its acknowledged show
	// baseline. A distinct local choice must conflict with B's latest choice.
	b = New()
	recoveryAttach(t, b, repoB, "host-a")
	b.SetWorkspaceArchivedFilter("only")
	a.SetWorkspaceArchivedFilter("default")
	for i := 0; i < 2; i++ {
		if got := a.Snapshot().WorkspaceLayout; got.ArchivedFilter != "default" || !strings.Contains(got.PreferenceError, "archivedFilter") {
			t.Fatal("pending choice or field conflict hidden", got)
		}
		a.RetryWorkspacePreferences()
		if recoveryRead(t, path, scope).ArchivedFilter != "only" {
			t.Fatal("retry overwrote conflicting filter")
		}
	}
	latest := recoveryRead(t, path, scope)
	latest.ArchivedFilter = "show"
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: latest})
	a.RetryWorkspacePreferences()
	if got := recoveryRead(t, path, scope); got.ArchivedFilter != "default" || got.GroupBy != "flat" || got.OrderBy != "updated" || a.Snapshot().WorkspaceLayout.PreferenceError != "" {
		t.Fatal("resolved conflict lost filter or independent fields", got)
	}
}
