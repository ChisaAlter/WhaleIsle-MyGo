package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSelectVisibleLeavesPluginOwnedRowsWithTheirOwner(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, scenario := range []string{"browse", "search", "archive"} {
		t.Run(scenario, func(t *testing.T) {
			query := ""
			if scenario == "search" {
				query = "conversation"
			}
			m := Model{ThemeID: "deepseek.light", SearchQuery: &query, Batch: session.SessionBatch{Enabled: true},
				Snapshot: session.ViewState{WorkspaceReady: true,
					Items: []session.Item{
						{SessionID: "ordinary", Title: "Ordinary conversation"},
						{SessionID: "owned", Title: "Owned conversation", Presentation: &session.SessionPresentation{Owner: "plugin", WorkingDirectory: "/workspace"}},
					},
					Workspaces:      []session.WorkspaceRow{{WorkspaceID: "w", Path: "/workspace", SessionIDs: []string{"ordinary", "owned"}}},
					WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: map[string]bool{"w": false}},
				},
				WorkspaceNavigation: []pluginui.WorkspaceNavigation{{SessionID: "owned", Owner: "plugin", WorkingDirectory: "/workspace", Title: "Owned conversation", Archived: scenario == "archive"}},
			}
			if scenario == "archive" {
				m.Snapshot.ArchivedSessionIDs = []string{"ordinary", "owned"}
			}
			var targets []session.SessionBatchTarget
			m.OnBatchEdit = func(action string, rows []session.SessionBatchTarget) {
				if action != "visible" {
					t.Fatalf("unexpected batch action %q", action)
				}
				targets = rows
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 300, 900)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			if scenario == "archive" {
				if err := tt.Click(i18n.T(i18n.KeySidebarArchivedSection)); err != nil {
					t.Fatal(err)
				}
			}
			if !tt.HasText("Owned conversation") {
				t.Fatal("owner navigation disappeared instead of remaining inert")
			}
			if err := tt.Click("Select visible"); err != nil {
				t.Fatal(err)
			}
			want := []session.SessionBatchTarget{{ID: "ordinary", Title: "Ordinary conversation"}}
			if !reflect.DeepEqual(targets, want) {
				t.Fatalf("generic batch received owned session: got %+v, want %+v", targets, want)
			}
			navigationEvidence(t, tt, "batch-owner-"+scenario)
		})
	}
}

func TestArchiveSectionKeepsUnprojectedManagedSessionsOutOfGenericMenus(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := Model{ThemeID: "deepseek.light", Snapshot: session.ViewState{WorkspaceReady: true,
		Items: []session.Item{
			{SessionID: "ordinary", Title: "Ordinary archive"},
			{SessionID: "project", Title: "Project archive", Presentation: &session.SessionPresentation{Owner: "project"}},
			{SessionID: "plugin", Title: "Plugin archive", Presentation: &session.SessionPresentation{Owner: "plugin"}},
		},
		Workspaces:         []session.WorkspaceRow{{WorkspaceID: "w", SessionIDs: []string{"ordinary", "project", "plugin"}}},
		ArchivedSessionIDs: []string{"ordinary", "project", "plugin", "missing"},
	}}
	var deleted string
	m.OnDelete = func(id string) { deleted = id }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 300, 900)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click(i18n.T(i18n.KeySidebarArchivedSection)); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Project archive") || tt.HasText("Plugin archive") {
		t.Fatal("managed archive exposed ordinary restore/delete controls")
	}
	for _, target := range []session.SessionBatchTarget{{ID: "ordinary", Title: "Ordinary archive"}, {ID: "missing", Title: "Missing session"}} {
		if err := tt.Click(sessionActionsLabel(target.ID, target.Title)); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarDelete)); err != nil {
			t.Fatal(err)
		}
		if deleted != target.ID {
			t.Fatalf("ordinary recovery action lost target: %q", deleted)
		}
	}
}
