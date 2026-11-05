package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceEmptyListDistinguishesReadyLoadingAndFailure(t *testing.T) {
	for _, group := range []string{"workspace", "workspace-tree", "flat"} {
		m := Model{Snapshot: session.ViewState{WorkspaceReady: true, WorkspaceLayout: session.WorkspaceLayoutState{GroupBy: group}}}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
		if !tt.HasText(workspaceText("暂无会话", "No sessions yet")) {
			t.Fatalf("%s empty view has no feedback", group)
		}
		m.Snapshot.ListLoading = true
		tt.Frame()
		if !tt.HasText(workspaceText("正在加载会话…", "Loading sessions…")) || tt.HasText(workspaceText("暂无会话", "No sessions yet")) {
			t.Fatal("loading was disguised as empty")
		}
		retried := 0
		m.Snapshot.ListErr = "fixture list failure"
		m.OnRefresh = func() { retried++ }
		tt.Frame()
		if !tt.HasText("fixture list failure") || tt.HasText(workspaceText("暂无会话", "No sessions yet")) {
			t.Fatal("failure was disguised as empty")
		}
		if err := tt.Click(workspaceText("重试加载会话", "Retry loading sessions")); err != nil {
			t.Fatal(err)
		}
		if retried != 1 {
			t.Fatal("list retry callback lost")
		}
	}
}

func TestWorkspaceArchivedOnlyEmptyHasWorkingReturn(t *testing.T) {
	filter := "only"
	m := Model{BrowseArchivedFilter: &filter, Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: map[string]bool{"::scratch": false}}, Items: []session.Item{{SessionID: "active", Title: "Visible after return", Cwd: "/scratch"}}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
	if !tt.HasText(workspaceText("暂无已归档会话", "No archived sessions")) || tt.HasText("Visible after return") {
		t.Fatal("archived-only view leaked normal rows or lacked empty state")
	}
	if err := tt.Click(workspaceText("查看其他会话", "View other sessions")); err != nil {
		t.Fatal(err)
	}
	if filter != "default" || !tt.HasText("Visible after return") || tt.HasText(workspaceText("暂无归档会话", "No archived sessions")) {
		t.Fatal("return did not restore the ordinary session list")
	}
}

func TestWorkspaceBrowseArchiveFilterMatchesRowsAndBatchTargets(t *testing.T) {
	filter := "only"
	m := Model{BrowseArchivedFilter: &filter, HideArchivedList: true, Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: map[string]bool{"::scratch": false}}, Items: []session.Item{{SessionID: "active", Title: "Active fixture", Cwd: "/scratch"}, {SessionID: "archive", Title: "Archived fixture", Cwd: "/scratch"}}, ArchivedSessionIDs: []string{"archive"}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
	if tt.HasText("Active fixture") || !tt.HasText("Archived fixture") || tt.HasText(workspaceText("暂无归档会话", "No archived sessions")) {
		t.Fatal("only archive projection differs from visible rows")
	}
	targets := VisibleBatchTargets(m)
	if len(targets) != 1 || targets[0].ID != "archive" {
		t.Fatalf("batch includes hidden rows: %+v", targets)
	}
	filter = "default"
	tt.Frame()
	if !tt.HasText("Active fixture") || tt.HasText("Archived fixture") {
		t.Fatal("hide archive projection differs from visible rows")
	}
	targets = VisibleBatchTargets(m)
	if len(targets) != 1 || targets[0].ID != "active" {
		t.Fatal("hidden archives remained in the batch target set")
	}
}

func TestWorkspaceArchiveOnlyPreservesStatusAndMenuFilter(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	filter, open := "show", false
	m := Model{BrowseArchivedFilter: &filter, WorkspaceMenuOpen: &open, Snapshot: session.ViewState{WorkspaceReady: true, WorkspaceLayout: session.WorkspaceLayoutState{PreferenceError: "fixture preference conflict"}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
	if err := tt.Click("View options"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Only archived"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if filter != "only" || !tt.HasText("No archived sessions") || !tt.HasText("fixture preference conflict") {
		t.Fatal("archive filter hid status feedback or failed to update")
	}
}
