package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

// This mirrors the application's wrapper: a successful read establishes its
// observed baseline, and a failed merge never acknowledges the pending value.
type recoveryWorkspacePreferences struct {
	store      *config.Store
	observed   map[string]config.WorkspaceNavigationPreferences
	reads      int
	writes     int
	beforeSave func()
}

func (r *recoveryWorkspacePreferences) WorkspaceNavigationPreferences(scope string) (config.WorkspaceNavigationPreferences, error) {
	r.reads++
	value, err := r.store.ReadWorkspaceNavigationPreferences(scope)
	if err == nil {
		r.observed[scope] = cloneWorkspacePreferences(value)
	}
	return value, err
}

func (r *recoveryWorkspacePreferences) SaveWorkspaceNavigationPreferences(scope string, next config.WorkspaceNavigationPreferences) error {
	r.writes++
	base, ok := r.observed[scope]
	if !ok {
		return errors.New("view preferences have no readable baseline for this Host")
	}
	if r.beforeSave != nil {
		f := r.beforeSave
		r.beforeSave = nil
		f()
	}
	if _, err := r.store.MergeWorkspaceNavigationPreferences(scope, base, next); err != nil {
		return err
	}
	r.observed[scope] = cloneWorkspacePreferences(next)
	return nil
}

func recoveryRepository(t *testing.T, values map[string]config.WorkspaceNavigationPreferences) (*recoveryWorkspacePreferences, string) {
	t.Helper()
	dir := t.TempDir()
	data, err := json.Marshal(map[string]any{"locale": "zh", "future": map[string]bool{"keep": true}, "goWorkspaceNavigation": values})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, config.FileName)
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &recoveryWorkspacePreferences{store: store, observed: map[string]config.WorkspaceNavigationPreferences{}}, path
}

func recoveryWrite(t *testing.T, path string, values map[string]config.WorkspaceNavigationPreferences) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"locale": "zh", "future": map[string]bool{"keep": true}, "goWorkspaceNavigation": values})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func recoveryBaseline(t *testing.T, s *Store, rows ...WorkspaceRow) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"type": "baseline", "value": map[string]any{"scratchCwd": "/scratch", "items": rows, "pinnedSessionIds": []string{}, "archivedSessionIds": []string{}}})
	if err != nil {
		t.Fatal(err)
	}
	s.applyWorkspaceFollowFrame(raw)
}

func recoveryAttach(t *testing.T, s *Store, repo WorkspacePreferenceStore, authority string, rows ...WorkspaceRow) {
	t.Helper()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	s.Attach(client, nil)
	s.BindWorkspacePreferences(client, authority, repo)
	recoveryBaseline(t, s, rows...)
}

func recoveryRead(t *testing.T, path, scope string) config.WorkspaceNavigationPreferences {
	t.Helper()
	store, err := config.Open(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	prefs, err := store.ReadWorkspaceNavigationPreferences(scope)
	if err != nil {
		t.Fatal(err)
	}
	return prefs
}

func TestWorkspacePreferencesRecoverFailedInitialReadWithoutOverwritingOthers(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/project/a"}
	b := WorkspaceRow{WorkspaceID: "b", Path: "/project/b"}
	removed := WorkspaceRow{WorkspaceID: "removed", Path: "/project/removed"}
	added := WorkspaceRow{WorkspaceID: "added", Path: "/project/added"}
	scopeA := preferenceHash("host", "host-a", "/scratch")
	scopeB := preferenceHash("host", "host-b", "/scratch")
	initial := config.WorkspaceNavigationPreferences{
		GroupBy: "workspace-tree", OrderBy: "updated",
		Collapsed:     map[string]bool{workspacePreferenceKey(a): false, workspacePreferenceKey(b): true, workspacePreferenceKey(removed): true},
		SessionOrders: map[string][]string{"::flat": {"f2", "f1"}, "::scratch": {"s2", "s1"}},
	}
	other := config.WorkspaceNavigationPreferences{GroupBy: "flat", OrderBy: "manual", Collapsed: map[string]bool{"other": true}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scopeA: initial, scopeB: other})
	if err := os.WriteFile(path, []byte(`{"goWorkspaceNavigation":`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	recoveryAttach(t, s, repo, "host-a", a, b, removed)
	if repo.reads != 1 || repo.writes != 0 || len(repo.observed) != 0 || s.Snapshot().WorkspaceLayout.PreferenceError == "" {
		t.Fatal("fixture did not create an initial read failure without a wrapper baseline")
	}
	s.ToggleWorkspaceGroup("a")
	if WorkspaceGroupCollapsed(s.Snapshot(), "a") {
		t.Fatal("first click did not open the default-closed workspace")
	}
	s.ToggleWorkspaceGroup("a")
	if got := s.Snapshot().WorkspaceLayout; !got.Collapsed["a"] || got.PreferenceError == "" || repo.writes != 0 {
		t.Fatalf("failed initial read lost the edit, error, or attempted a blind save: %+v", got)
	}
	s.RetryWorkspacePreferences()
	if got := s.Snapshot().WorkspaceLayout; !got.Collapsed["a"] || got.PreferenceError == "" || repo.writes != 0 || repo.reads != 2 {
		t.Fatalf("failed retry lost the pending edit or error: %+v", got)
	}
	// The Host registry can change before the preference read recovers.
	recoveryBaseline(t, s, a, b, added)
	if !s.Snapshot().WorkspaceLayout.Collapsed["a"] || repo.reads != 2 {
		t.Fatal("follow update discarded edit or implicitly retried failed read")
	}
	latest := cloneWorkspacePreferences(initial)
	latest.GroupBy = "flat"
	latest.OrderBy = "manual"
	latest.Collapsed[workspacePreferenceKey(b)] = false
	latest.Collapsed[workspacePreferenceKey(added)] = true
	latest.SessionOrders["::scratch"] = []string{"external", "s1"}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scopeA: latest, scopeB: other})
	s.RetryWorkspacePreferences()
	want := cloneWorkspacePreferences(latest)
	want.Collapsed[workspacePreferenceKey(a)] = true
	got := recoveryRead(t, path, scopeA)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(recoveryRead(t, path, scopeB), other) {
		t.Fatalf("retry overwrote unrelated data:\n got %+v\nwant %+v", got, want)
	}
	layout := s.Snapshot().WorkspaceLayout
	if layout.PreferenceError != "" || layout.GroupBy != "flat" || layout.OrderBy != "manual" || !layout.Collapsed["a"] || layout.Collapsed["b"] || !layout.Collapsed["added"] {
		t.Fatalf("UI did not adopt recovered baseline: %+v", layout)
	}
	restarted := New()
	recoveryAttach(t, restarted, repo, "host-a", a, b, added)
	if got := restarted.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || !got.Collapsed["a"] || got.Collapsed["b"] || !got.Collapsed["added"] || !reflect.DeepEqual(got.SessionOrders, want.SessionOrders) {
		t.Fatalf("restart lost recovered preferences: %+v", got)
	}
}

func TestWorkspacePreferencesFailedReadSurvivesHostRoundTripAndExplicitEdits(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a", SessionIDs: []string{"a-session"}}
	b := WorkspaceRow{WorkspaceID: "b", Path: "/b", SessionIDs: []string{"b-session"}}
	scopeA := preferenceHash("host", "host-a", "/scratch")
	scopeB := preferenceHash("host", "host-b", "/scratch")
	originalA := config.WorkspaceNavigationPreferences{GroupBy: "workspace-tree", OrderBy: "updated", Collapsed: map[string]bool{workspacePreferenceKey(b): true}, SessionOrders: map[string][]string{"::scratch": {"x", "y"}}}
	originalB := config.WorkspaceNavigationPreferences{GroupBy: "flat", OrderBy: "updated", Collapsed: map[string]bool{workspacePreferenceKey(a): true}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scopeA: originalA, scopeB: originalB})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.items = []Item{{SessionID: "a-session", UpdatedAt: 20}, {SessionID: "b-session", UpdatedAt: 10}, {SessionID: "x", Cwd: "/scratch", UpdatedAt: 8}, {SessionID: "y", Cwd: "/scratch", UpdatedAt: 5}}
	recoveryAttach(t, s, repo, "host-a", a, b)
	s.ToggleWorkspaceGroup("a") // expand the default-folded group
	s.ToggleWorkspaceGroup("a") // retain an explicit collapsed edit across recovery
	// Switching from the unread baseline's updated fallback freezes every
	// currently displayed account before any external preference can be read.
	s.SetWorkspaceGrouping("workspace")
	s.SetWorkspaceOrdering("manual")
	wantOrders := map[string][]string{"a": {"a-session"}, "b": {"b-session"}, "::scratch": {"x", "y"}, "::flat": {"a-session", "b-session", "x", "y"}}
	if got := s.Snapshot().WorkspaceLayout.SessionOrders; !reflect.DeepEqual(got, wantOrders) {
		t.Fatal("failed-read manual choice did not capture actual accounts", got)
	}
	if repo.writes != 0 {
		t.Fatal("saved before a readable baseline")
	}
	recoveryAttach(t, s, repo, "host-b", a, b)
	if repo.writes != 0 || s.Snapshot().WorkspaceLayout.Collapsed["a"] || s.Snapshot().WorkspaceLayout.PreferenceError == "" {
		t.Fatal("failed Host B borrowed Host A's pending edits or error disappeared")
	}
	latestA := cloneWorkspacePreferences(originalA)
	latestA.Collapsed[workspacePreferenceKey(b)] = false
	latestA.SessionOrders["::scratch"] = []string{"external"}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scopeA: latestA, scopeB: originalB})
	recoveryAttach(t, s, repo, "host-a", a, b)
	wantA := cloneWorkspacePreferences(latestA)
	wantA.GroupBy, wantA.OrderBy = "workspace", "manual"
	wantA.SessionOrders = cloneSessionOrders(wantOrders)
	wantA.Collapsed[workspacePreferenceKey(a)] = true
	if got := recoveryRead(t, path, scopeA); !reflect.DeepEqual(got, wantA) {
		t.Fatalf("Host round trip failed to merge explicit edits: %+v", got)
	}
	if got := recoveryRead(t, path, scopeB); !reflect.DeepEqual(got, originalB) {
		t.Fatalf("Host A save changed Host B: %+v", got)
	}
	recoveryAttach(t, s, repo, "host-b", a, b)
	if got := s.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || got.OrderBy != "updated" || !got.Collapsed["a"] || got.PreferenceError != "" {
		t.Fatalf("Host B did not independently recover: %+v", got)
	}
}

func TestWorkspacePreferencesFailedReadAndEarlyEditDoNotMaterializeDefaults(t *testing.T) {
	b := WorkspaceRow{WorkspaceID: "b", Path: "/b"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "workspace-tree", OrderBy: "updated", Collapsed: map[string]bool{workspacePreferenceKey(b): true}, SessionOrders: map[string][]string{"::flat": {"x", "y"}}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	s.Attach(client, nil)
	s.BindWorkspacePreferences(client, "host-a", repo)
	s.ToggleWorkspaceGroup("::scratch")
	if WorkspaceGroupCollapsed(s.Snapshot(), "::scratch") {
		t.Fatal("first early click did not explicitly open scratch")
	}
	s.ToggleWorkspaceGroup("::scratch")
	recoveryBaseline(t, s, b)
	if got := s.Snapshot().WorkspaceLayout; !got.Collapsed["::scratch"] || got.PreferenceError == "" || repo.writes != 0 {
		t.Fatalf("early edit did not survive initial preference read failure: %+v", got)
	}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s.RetryWorkspacePreferences()
	initial.Collapsed["::scratch"] = true
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, initial) {
		t.Fatalf("early recovery overwrote untouched baseline: %+v", got)
	}
}

func TestWorkspacePreferencesEarlyEditsResolveOnlyKnownWorkspace(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a"}
	b := WorkspaceRow{WorkspaceID: "b", Path: "/b"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "flat", OrderBy: "updated", Collapsed: map[string]bool{workspacePreferenceKey(b): true, "::other": true}, SessionOrders: map[string][]string{"::flat": {"f2", "f1"}}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
	s.Attach(client, nil)
	s.BindWorkspacePreferences(client, "host-a", repo)
	s.ToggleWorkspaceGroup("a")
	s.ToggleWorkspaceGroup("::scratch")
	if WorkspaceGroupCollapsed(s.Snapshot(), "a") || WorkspaceGroupCollapsed(s.Snapshot(), "::scratch") {
		t.Fatal("first early click did not explicitly open both groups")
	}
	s.ToggleWorkspaceGroup("a")
	s.ToggleWorkspaceGroup("::scratch")
	if repo.reads != 0 || repo.writes != 0 {
		t.Fatal("initial Host baseline was not pending")
	}
	// The workspace is absent in the first frame: do not guess its persistent key.
	recoveryBaseline(t, s, b)
	want := cloneWorkspacePreferences(initial)
	want.Collapsed["::scratch"] = true
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, want) {
		t.Fatalf("early edit overwrote unrelated state: %+v", got)
	}
	if !s.Snapshot().WorkspaceLayout.Collapsed["a"] {
		t.Fatal("unresolved early UI edit was discarded")
	}
	recoveryBaseline(t, s, a, b)
	want.Collapsed[workspacePreferenceKey(a)] = true
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, want) {
		t.Fatalf("later workspace discovery did not safely persist pending toggle: %+v", got)
	}
}

func TestWorkspacePreferencesRetryPreservesConflictUntilResolved(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "workspace", OrderBy: "manual", Collapsed: map[string]bool{}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s := New()
	recoveryAttach(t, s, repo, "host-a", a)
	external := cloneWorkspacePreferences(initial)
	external.GroupBy = "flat"
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: external})
	s.SetWorkspaceGrouping("workspace-tree")
	for i := 0; i < 2; i++ {
		if got := s.Snapshot().WorkspaceLayout; got.GroupBy != "workspace-tree" || !strings.Contains(got.PreferenceError, "groupBy") {
			t.Fatalf("conflict did not retain pending edit and helpful error: %+v", got)
		}
		s.RetryWorkspacePreferences()
		if got := recoveryRead(t, path, scope); got.GroupBy != "flat" {
			t.Fatalf("retry overwrote conflicting external edit: %+v", got)
		}
	}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s.RetryWorkspacePreferences()
	if got := recoveryRead(t, path, scope); got.GroupBy != "workspace-tree" || s.Snapshot().WorkspaceLayout.PreferenceError != "" {
		t.Fatalf("resolved conflict did not persist pending edit: %+v", got)
	}
}

func TestWorkspacePreferencesFailedReadPreservesOtherSessionOrders(t *testing.T) {
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{
		GroupBy: "flat", OrderBy: "manual",
		Collapsed:     map[string]bool{"::scratch": true},
		SessionOrders: map[string][]string{"::flat": {"a", "b", "c"}, "::scratch": {"x", "y"}},
	}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.items = []Item{{SessionID: "a", Cwd: "/scratch", UpdatedAt: 30}, {SessionID: "b", Cwd: "/scratch", UpdatedAt: 20}, {SessionID: "c", Cwd: "/scratch", UpdatedAt: 10}, {SessionID: "member", UpdatedAt: 5}, {SessionID: "hidden", UpdatedAt: 4}}
	workspace := WorkspaceRow{WorkspaceID: "one", Path: "/one", SessionIDs: []string{"member", "hidden"}}
	recoveryAttach(t, s, repo, "host-a", workspace)
	s.archivedSessionIDs = []string{"hidden"}
	s.InsertLocalSessionBefore(s.Snapshot().WorkspaceLayout.HostGeneration, "::flat", "c", "a")
	wantOrders := map[string][]string{"::flat": {"c", "a", "b", "member", "hidden"}, "::scratch": {"a", "b", "c"}, "one": {"member", "hidden"}}
	if got := s.Snapshot().WorkspaceLayout; !reflect.DeepEqual(got.SessionOrders, wantOrders) || got.OrderBy != "manual" || got.PreferenceError == "" || repo.writes != 0 {
		t.Fatalf("failed read lost pending all-account snapshot: %+v", got)
	}
	latest := cloneWorkspacePreferences(initial)
	latest.Collapsed[workspacePreferenceKey(workspace)] = true
	latest.SessionOrders["::scratch"] = []string{"y", "x"}
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: latest})
	s.RetryWorkspacePreferences()
	// The explicit updated-mode drop owns all account orders. Recovery keeps
	// that captured roster while merging unrelated expansion/grouping fields.
	latest.SessionOrders = cloneSessionOrders(wantOrders)
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, latest) {
		t.Fatalf("recovering snapshot lost accounts or unrelated preferences: %+v", got)
	}
	restarted := New()
	recoveryAttach(t, restarted, repo, "host-a", workspace)
	if got := restarted.Snapshot().WorkspaceLayout; !reflect.DeepEqual(got.SessionOrders, latest.SessionOrders) || got.GroupBy != "flat" || !got.Collapsed["one"] {
		t.Fatalf("restart lost recovered accounts or expansion: %+v", got)
	}
}

func TestWorkspacePreferencesRecoverySaveHonorsConcurrentChange(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "workspace-tree", OrderBy: "updated", Collapsed: map[string]bool{}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	recoveryAttach(t, s, repo, "host-a", a)
	s.SetWorkspaceGrouping("flat")
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	external := cloneWorkspacePreferences(initial)
	external.GroupBy = "workspace"
	repo.beforeSave = func() { recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: external}) }
	s.RetryWorkspacePreferences()
	if got := s.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || !strings.Contains(got.PreferenceError, "groupBy") {
		t.Fatalf("save-time conflict lost pending change: %+v", got)
	}
	if got := recoveryRead(t, path, scope); got.GroupBy != "workspace" {
		t.Fatal("save-time external change was overwritten")
	}
	s.RetryWorkspacePreferences()
	if got := recoveryRead(t, path, scope); got.GroupBy != "workspace" {
		t.Fatal("conflict retry silently reset the observed baseline")
	}
}

func TestWorkspacePreferencesDeletedWorkspaceDoesNotRetargetEdit(t *testing.T) {
	old := WorkspaceRow{WorkspaceID: "same-id", Path: "/old"}
	replacement := WorkspaceRow{WorkspaceID: "same-id", Path: "/new"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "workspace", OrderBy: "manual", Collapsed: map[string]bool{workspacePreferenceKey(replacement): false}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	recoveryAttach(t, s, repo, "host-a", old)
	s.ToggleWorkspaceGroup("same-id")
	if WorkspaceGroupCollapsed(s.Snapshot(), "same-id") {
		t.Fatal("first click did not open the old default-closed workspace")
	}
	s.ToggleWorkspaceGroup("same-id")
	recoveryBaseline(t, s, replacement)
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s.RetryWorkspacePreferences()
	want := cloneWorkspacePreferences(initial)
	want.Collapsed[workspacePreferenceKey(old)] = true
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, want) || s.Snapshot().WorkspaceLayout.Collapsed["same-id"] {
		t.Fatalf("removed workspace edit was applied to replacement path: %+v", got)
	}
}

func TestWorkspacePreferencesUnrelatedEditDoesNotRevertConcurrentSave(t *testing.T) {
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a"}
	b := WorkspaceRow{WorkspaceID: "b", Path: "/b"}
	scope := preferenceHash("host", "host-a", "/scratch")
	initial := config.WorkspaceNavigationPreferences{GroupBy: "workspace", OrderBy: "manual", Collapsed: map[string]bool{workspacePreferenceKey(a): false, workspacePreferenceKey(b): false}}
	repo, path := recoveryRepository(t, map[string]config.WorkspaceNavigationPreferences{scope: initial})
	s := New()
	recoveryAttach(t, s, repo, "host-a", a, b)
	external := cloneWorkspacePreferences(initial)
	external.GroupBy = "flat"
	external.Collapsed[workspacePreferenceKey(b)] = true
	recoveryWrite(t, path, map[string]config.WorkspaceNavigationPreferences{scope: external})
	s.ToggleWorkspaceGroup("a")
	s.SetWorkspaceOrdering("updated")
	external.OrderBy = "updated"
	external.Collapsed[workspacePreferenceKey(a)] = true
	if got := recoveryRead(t, path, scope); !reflect.DeepEqual(got, external) {
		t.Fatalf("later edit reverted fields merged from another window: %+v", got)
	}
	restarted := New()
	recoveryAttach(t, restarted, repo, "host-a", a, b)
	if got := restarted.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || got.OrderBy != "updated" || !got.Collapsed["a"] || !got.Collapsed["b"] {
		t.Fatalf("restart lost independent changes: %+v", got)
	}
}

func TestWorkspacePreferencesWriteFailureAcrossHostsRetainsError(t *testing.T) {
	repo := &fixtureWorkspacePreferences{values: map[string]config.WorkspaceNavigationPreferences{}}
	a := WorkspaceRow{WorkspaceID: "a", Path: "/a"}
	s := New()
	recoveryAttach(t, s, repo, "host-a", a)
	repo.fail = true
	s.SetWorkspaceGrouping("flat")
	recoveryAttach(t, s, repo, "host-b", a)
	recoveryAttach(t, s, repo, "host-a", a)
	if got := s.Snapshot().WorkspaceLayout; got.GroupBy != "flat" || !strings.Contains(got.PreferenceError, "fixture read-only disk") {
		t.Fatalf("Host return lost unsaved change or diagnostic: %+v", got)
	}
	repo.fail = false
	s.RetryWorkspacePreferences()
	if got := s.Snapshot().WorkspaceLayout; got.PreferenceError != "" || repo.values[s.workspacePreferences.scope].GroupBy != "flat" {
		t.Fatalf("retry did not recover write failure: %+v", got)
	}
}
