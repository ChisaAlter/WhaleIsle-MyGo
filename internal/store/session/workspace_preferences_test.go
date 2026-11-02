package session

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

type fixtureWorkspacePreferences struct {
	values   map[string]config.WorkspaceNavigationPreferences
	fail     bool
	readFail bool
	writes   int
}

func (f *fixtureWorkspacePreferences) WorkspaceNavigationPreferences(key string) (config.WorkspaceNavigationPreferences, error) {
	if f.readFail {
		return config.WorkspaceNavigationPreferences{}, errors.New("fixture read failure")
	}
	return cloneWorkspacePreferences(f.values[key]), nil
}
func (f *fixtureWorkspacePreferences) SaveWorkspaceNavigationPreferences(key string, p config.WorkspaceNavigationPreferences) error {
	f.writes++
	if f.fail {
		return errors.New("fixture read-only disk")
	}
	f.values[key] = cloneWorkspacePreferences(p)
	return nil
}
func TestWorkspaceExpansionSurvivesArchiveFollowRestartAndWorkspaceSwitch(t *testing.T) {
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	baseline := json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"open","path":"/open","sessionIds":["active","archived"]},{"workspaceId":"closed","path":"/closed","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":["archived"]}}`)
	attach := func(s *Store) {
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
		s.Attach(client, nil)
		s.BindWorkspacePreferences(client, "fixture-host", repo)
		s.applyWorkspaceFollowFrame(baseline)
	}
	s := New()
	attach(s)
	s.ToggleWorkspaceGroup("open")
	s.ToggleWorkspaceGroup("closed")
	if WorkspaceGroupCollapsed(s.Snapshot(), "open") || WorkspaceGroupCollapsed(s.Snapshot(), "closed") {
		t.Fatal("first click must open both default-closed groups")
	}
	s.ToggleWorkspaceGroup("closed")
	want := map[string]bool{"open": false, "closed": true}
	check := func(store *Store, when string) {
		t.Helper()
		if got := store.Snapshot().WorkspaceLayout.Collapsed; !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: collapsed=%v want %v", when, got, want)
		}
	}
	check(s, "before archive follow")
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"archived","archivedSessionIds":[]}`))
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"archived","archivedSessionIds":["archived"]}`))
	check(s, "after archive follow")
	s.SetWorkspaceCwd("/open")
	s.SetWorkspaceCwd("/closed")
	check(s, "after workspace selection changes")
	restarted := New()
	attach(restarted)
	check(restarted, "after store restart")
}

func TestWorkspacePreferencesHostIsolationFixture(t *testing.T) {
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	baseline := func(s *Store, path string) {
		raw, _ := json.Marshal(map[string]any{"type": "baseline", "value": map[string]any{"scratchCwd": "/same/scratch", "items": []any{map[string]any{"workspaceId": "same-id", "path": path, "sessionIds": []string{}}}, "pinnedSessionIds": []string{}, "archivedSessionIds": []string{}}})
		s.applyWorkspaceFollowFrame(raw)
	}
	attach := func(s *Store, authority, path string) *dsh.Client {
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:3080", Cookie: "do-not-persist-this"}}
		s.Attach(client, nil)
		s.BindWorkspacePreferences(client, authority, repo)
		baseline(s, path)
		return client
	}
	s := New()
	old := attach(s, "host-a", "/project/a")
	s.SetWorkspaceGrouping("workspace-tree")
	s.SetWorkspaceOrdering("updated")
	s.ToggleWorkspaceGroup("same-id")
	if WorkspaceGroupCollapsed(s.Snapshot(), "same-id") {
		t.Fatal("first click did not open the default-closed workspace")
	}
	s.ToggleWorkspaceGroup("same-id")
	scopeA := s.workspacePreferences.scope
	if len(repo.values) != 1 || repo.values[scopeA].GroupBy != "workspace-tree" || !repo.values[scopeA].Collapsed[workspacePreferenceKey(s.Snapshot().Workspaces[0])] {
		t.Fatal("view state not persisted")
	}
	restarted := New()
	attach(restarted, "host-a", "/project/a")
	layout := restarted.Snapshot().WorkspaceLayout
	if layout.GroupBy != "workspace-tree" || layout.OrderBy != "updated" || !layout.Collapsed["same-id"] {
		t.Fatal("new Store did not restore preferences", layout)
	}
	attach(s, "host-b", "/project/a")
	if s.Snapshot().WorkspaceLayout.GroupBy != "workspace" || s.Snapshot().WorkspaceLayout.Collapsed["same-id"] {
		t.Fatal("other Host borrowed preferences")
	}
	s.SetWorkspaceGrouping("flat")
	if len(repo.values) != 2 {
		t.Fatal("Host scopes collided")
	}
	s.BindWorkspacePreferences(old, "host-a", repo)
	if s.workspacePreferences.scope == scopeA {
		t.Fatal("late old Host bind changed current preference scope")
	}
	attach(s, "host-a", "/project/b")
	if s.Snapshot().WorkspaceLayout.Collapsed["same-id"] {
		t.Fatal("reused workspace id borrowed another path's fold")
	}
	attach(s, "host-a", "/project/a")
	repo.fail = true
	s.ToggleWorkspaceGroup("same-id")
	s.SetWorkspaceGrouping("flat")
	layout = s.Snapshot().WorkspaceLayout
	if layout.GroupBy != "flat" || layout.Collapsed["same-id"] || layout.PreferenceError == "" {
		t.Fatal("save failure lost active state or error")
	}
	if repo.values[scopeA].GroupBy != "workspace-tree" {
		t.Fatal("failed persistence changed durable value")
	}
	// A follow update does not repeatedly write the same failed preferences.
	n := repo.writes
	baseline(s, "/project/a")
	if repo.writes != n || s.Snapshot().WorkspaceLayout.GroupBy != "flat" {
		t.Fatal("follow retried disk write or reverted dirty state")
	}
	attach(s, "host-b", "/project/a")
	attach(s, "host-a", "/project/a")
	if s.Snapshot().WorkspaceLayout.GroupBy != "flat" {
		t.Fatal("Host round trip discarded unsaved preferences")
	}
	repo.fail = false
	s.RetryWorkspacePreferences()
	if s.Snapshot().WorkspaceLayout.PreferenceError != "" || repo.values[scopeA].GroupBy != "flat" {
		t.Fatal("explicit save retry failed")
	}
	// Edits made while the initial Host baseline is loading survive restore.
	early := New()
	earlyClient := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	early.Attach(earlyClient, nil)
	early.BindWorkspacePreferences(earlyClient, "host-early", repo)
	early.ToggleWorkspaceGroup("::scratch")
	if WorkspaceGroupCollapsed(early.Snapshot(), "::scratch") {
		t.Fatal("first early click did not explicitly open scratch")
	}
	early.ToggleWorkspaceGroup("::scratch")
	early.SetWorkspaceGrouping("flat")
	baseline(early, "/project/a")
	if got := early.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || !got.Collapsed["::scratch"] {
		t.Fatal("initial follow discarded early view edits", got)
	}
	// Failed reads can be explicitly retried, without writing defaults.
	unread := New()
	repo.readFail = true
	writes := repo.writes
	attach(unread, "host-a", "/project/a")
	if unread.Snapshot().WorkspaceLayout.PreferenceError == "" || repo.writes != writes {
		t.Fatal("read error missing or wrote defaults")
	}
	repo.readFail = false
	unread.RetryWorkspacePreferences()
	if got := unread.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || got.PreferenceError != "" || repo.writes != writes {
		t.Fatal("read retry did not restore saved view", got)
	}
	groups := []WorkspaceGroup{{Key: "a", Kind: "workspace"}, {Key: "b", Kind: "workspace"}, {Key: "c", Kind: "workspace"}, {Key: "child", Kind: "workspace", ParentID: "a"}}
	if anchor, ok := WorkspaceDropAnchor(groups, "c", "a", false); !ok || anchor != "a" {
		t.Fatal("before drop")
	}
	if anchor, ok := WorkspaceDropAnchor(groups, "a", "c", true); !ok || anchor != "" {
		t.Fatal("append drop")
	}
	if _, ok := WorkspaceDropAnchor(groups, "a", "child", false); ok {
		t.Fatal("cross-parent drop")
	}
	if _, ok := WorkspaceDropAnchor(groups, "a", "b", false); ok {
		t.Fatal("no-op drop")
	}
	// An old drag cannot reach the new Host even when it reuses workspace ids.
	s.InsertWorkspaceBeforeForHost(context.Background(), s.eventsGeneration-1, "same-id", "")
	if s.Snapshot().WorkspaceLayout.Reordering {
		t.Fatal("old Host drag was accepted")
	}
}
