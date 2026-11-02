package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestBatchSelectionScopeOnlyTracksInstalledBrowserDimensions(t *testing.T) {
	query := "  a\x00b  "
	expanded := false
	m := Model{SearchQuery: &query, archiveExpanded: &expanded, Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: map[string]bool{"group": false}}}}
	base := BatchSelectionScope(m)
	if !base.Wide || base.GroupBy != "workspace" || base.ArchivedFilter != "default" || base.Query != "ab" {
		t.Fatalf("scope=%+v", base)
	}
	expanded = true
	m.Snapshot.WorkspaceLayout.Collapsed["group"] = true
	m.Snapshot.WorkspaceLayout.OrderBy = "updated"
	m.HideArchivedList = true
	if got := BatchSelectionScope(m); got != base {
		t.Fatalf("disclosure/order cleared scope: %+v", got)
	}
	m.Collapsed = true
	if BatchSelectionScope(m) == base {
		t.Fatal("rail transition missing from scope")
	}
	m.Collapsed = false
	m.Snapshot.WorkspaceLayout.GroupBy = "flat"
	if BatchSelectionScope(m) == base {
		t.Fatal("group transition missing")
	}
	m.Snapshot.WorkspaceLayout.GroupBy = ""
	m.Snapshot.WorkspaceLayout.ArchivedFilter = "only"
	if BatchSelectionScope(m) == base {
		t.Fatal("archive visibility missing")
	}
	m.Snapshot.WorkspaceLayout.ArchivedFilter = ""
	query = "different"
	if BatchSelectionScope(m) == base {
		t.Fatal("query transition missing")
	}
}
func TestBatchDeleteWarningPreservesHostActivityProtection(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	confirmed := uint64(0)
	calls := 0
	m := Model{Batch: session.SessionBatch{Enabled: true, Confirming: true, Kind: "delete", Token: 17, Targets: []session.SessionBatchTarget{{ID: "a", Title: "A"}}}, OnBatchEdit: func(string, []session.SessionBatchTarget) {}, OnBatchConfirm: func(token uint64) { calls++; confirmed = token }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if !tt.HasText("Running turns must be stopped first") || !tt.HasText("Workspace files are unchanged") || tt.HasText("Only archived sessions can be deleted") {
		t.Fatalf("stale deletion contract %q", tt.Texts())
	}
	if calls != 0 {
		t.Fatal("paint confirmed deletion")
	}
	if err := tt.Click("Confirm batch"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || confirmed != 17 {
		t.Fatalf("confirmation token/callback %v/%v", calls, confirmed)
	}
}
