package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/store/session"
)

func workspaceViewFixture(t *testing.T) (*workspaceViewPreferences, *config.Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{"locale":"zh","goWorkspaceNavigation":{"a":{"groupBy":"workspace","orderBy":"manual","collapsed":{"local":false,"remote":false}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return newWorkspaceViewPreferences(store), store, dir
}

func TestWorkspaceViewPreferencesContinuousSavesPreserveExternalFields(t *testing.T) {
	view, store, dir := workspaceViewFixture(t)
	next, err := view.WorkspaceNavigationPreferences("a")
	if err != nil {
		t.Fatal(err)
	}
	external, _ := config.Open(dir)
	if err = external.Update(func(c *config.Config) { c.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	remote, _ := external.WorkspaceNavigationPreferences("a")
	remote.OrderBy = "updated"
	remote.Collapsed["remote"] = true
	if err = external.SaveWorkspaceNavigationPreferences("a", remote); err != nil {
		t.Fatal(err)
	}
	next.GroupBy = "flat"
	next.Collapsed["local"] = true
	if err = view.SaveWorkspaceNavigationPreferences("a", next); err != nil {
		t.Fatal(err)
	}
	// The session retains its own accepted request, including old OrderBy.
	// A later different edit must not turn that old value into new intent.
	next.GroupBy = "workspace-tree"
	if err = view.SaveWorkspaceNavigationPreferences("a", next); err != nil {
		t.Fatal(err)
	}
	other, err := view.WorkspaceNavigationPreferences("b")
	if err != nil {
		t.Fatal(err)
	}
	other.GroupBy = "flat"
	if err = view.SaveWorkspaceNavigationPreferences("b", other); err != nil {
		t.Fatal(err)
	}
	if err = view.SaveWorkspaceNavigationPreferences("a", next); err != nil {
		t.Fatal(err)
	}
	reopened, _ := config.Open(dir)
	got, _ := reopened.WorkspaceNavigationPreferences("a")
	if got.GroupBy != "workspace-tree" || got.OrderBy != "updated" || !got.Collapsed["local"] || !got.Collapsed["remote"] || store.Get().Locale != "en" {
		t.Fatalf("continuous saves overwrote concurrent fields: %+v", got)
	}
	restarted := newWorkspaceViewPreferences(reopened)
	loaded, err := restarted.WorkspaceNavigationPreferences("a")
	if err != nil || loaded.OrderBy != "updated" {
		t.Fatal("restart did not restore merged durable view")
	}
}

func TestWorkspaceViewPreferencesConflictRetainsPendingAndRetries(t *testing.T) {
	view, _, dir := workspaceViewFixture(t)
	next, err := view.WorkspaceNavigationPreferences("a")
	if err != nil {
		t.Fatal(err)
	}
	external, _ := config.Open(dir)
	remote, _ := external.WorkspaceNavigationPreferences("a")
	remote.GroupBy = "flat"
	if err = external.SaveWorkspaceNavigationPreferences("a", remote); err != nil {
		t.Fatal(err)
	}
	next.GroupBy = "workspace-tree"
	for retry := 0; retry < 2; retry++ {
		err = view.SaveWorkspaceNavigationPreferences("a", next)
		if !errors.Is(err, config.ErrConflict) || !strings.Contains(err.Error(), "groupBy") || next.GroupBy != "workspace-tree" {
			t.Fatalf("conflict lost pending request or explanation: %v", err)
		}
	}
	remote.GroupBy = "workspace"
	if err = external.SaveWorkspaceNavigationPreferences("a", remote); err != nil {
		t.Fatal(err)
	}
	if err = view.SaveWorkspaceNavigationPreferences("a", next); err != nil {
		t.Fatal("explicit retry failed", err)
	}
	reopened, _ := config.Open(dir)
	got, _ := reopened.WorkspaceNavigationPreferences("a")
	if got.GroupBy != "workspace-tree" {
		t.Fatal("pending request was not persisted")
	}
}

func TestWorkspaceViewPreferencesBindingRetainsBaselineAcrossReconnects(t *testing.T) {
	_, store, _ := workspaceViewFixture(t)
	state := &mainState{cfgStore: store}
	state.bindWorkspacePreferences(session.New(), nil)
	bound := state.viewPreferences
	if _, err := bound.WorkspaceNavigationPreferences("a"); err != nil {
		t.Fatal(err)
	}
	state.bindWorkspacePreferences(session.New(), nil)
	if state.viewPreferences != bound || len(bound.observed) != 1 {
		t.Fatal("rebind replaced acknowledged view baseline")
	}
}

func TestWorkspaceViewPreferencesFirstEditMaterializesDefaultsWithoutFalseConflict(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{"locale":"zh"}`), 0600); err != nil {
			t.Fatal(err)
		}
		store, err := config.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		view := newWorkspaceViewPreferences(store)
		next, err := view.WorkspaceNavigationPreferences("new-host")
		if err != nil {
			t.Fatal(err)
		}
		// The session converts the absent grouping/order to its visible defaults
		// when recording an unrelated collapse action.
		next.GroupBy, next.OrderBy = "workspace", "manual"
		next.Collapsed = map[string]bool{"local": true}
		if concurrent {
			external, _ := config.Open(dir)
			if err = external.SaveWorkspaceNavigationPreferences("new-host", config.WorkspaceNavigationPreferences{GroupBy: "flat", OrderBy: "manual"}); err != nil {
				t.Fatal(err)
			}
		}
		if err = view.SaveWorkspaceNavigationPreferences("new-host", next); err != nil {
			t.Fatal("default materialization was mistaken for user intent", err)
		}
		reopened, _ := config.Open(dir)
		got, _ := reopened.WorkspaceNavigationPreferences("new-host")
		want := "workspace"
		if concurrent {
			want = "flat"
		}
		if got.GroupBy != want || !got.Collapsed["local"] {
			t.Fatalf("first collapse overwrote grouping: %+v", got)
		}
	}
}
