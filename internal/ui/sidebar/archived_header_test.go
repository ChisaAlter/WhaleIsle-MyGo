package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchivedSectionToggleKeepsWorkspaceExpansion(t *testing.T) {
	for _, grouping := range []string{"workspace", "workspace-tree"} {
		t.Run(grouping, func(t *testing.T) {
			filter := "default"
			toggles := []string{}
			m := Model{
				BrowseArchivedFilter:   &filter,
				OnToggleWorkspaceGroup: nil,
				Snapshot: session.ViewState{
					WorkspaceReady: true,
					ScratchCwd:     "/scratch",
					SelectedID:     "active-open",
					Items: []session.Item{
						{SessionID: "active-open", Title: "Open group session"},
						{SessionID: "active-closed", Title: "Closed group session"},
						{SessionID: "archived", Title: "Archived group session"},
					},
					Workspaces: []session.WorkspaceRow{
						{WorkspaceID: "open", Title: "Open workspace", Path: "/open", SessionIDs: []string{"active-open", "archived"}},
						{WorkspaceID: "closed", Title: "Closed workspace", Path: "/closed", SessionIDs: []string{"active-closed"}},
					},
					ArchivedSessionIDs: []string{"archived"},
					WorkspaceLayout:    session.WorkspaceLayoutState{GroupBy: grouping, Collapsed: map[string]bool{"open": false, "closed": true}},
				},
			}
			m.OnToggleWorkspaceGroup = func(id string) {
				toggles = append(toggles, id)
				m.Snapshot.WorkspaceLayout.Collapsed[id] = !m.Snapshot.WorkspaceLayout.Collapsed[id]
			}
			activeBatch := false
			var batchSelected []session.SessionBatchTarget
			m.OnBatchEdit = func(action string, targets []session.SessionBatchTarget) {
				if action == "toggle" {
					activeBatch = !activeBatch
				}
				if action == "visible" {
					batchSelected = append([]session.SessionBatchTarget(nil), targets...)
				}
			}
			tt := ui.NewTester(func(c *ui.Context) { m.Batch.Enabled = activeBatch; View(c, m) }, 280, 1000)
			check := func(archiveVisible bool) {
				t.Helper()
				for _, title := range []string{"Open workspace", "Closed workspace", "Open group session"} {
					if !tt.HasText(title) {
						t.Fatalf("%s disappeared after archive toggle", title)
					}
				}
				if tt.HasText("Closed group session") || tt.HasText("Archived group session") != archiveVisible {
					t.Fatalf("closed workspace or archive visibility changed unexpectedly (archive=%v)", archiveVisible)
				}
				if filter != "default" || !reflect.DeepEqual(m.Snapshot.WorkspaceLayout.Collapsed, map[string]bool{"open": false, "closed": true}) || len(toggles) != 0 {
					t.Fatalf("archive click changed filter or workspace state: filter=%q collapsed=%v toggles=%v", filter, m.Snapshot.WorkspaceLayout.Collapsed, toggles)
				}
			}
			if !tt.HasText("Open workspace") || !tt.HasText("Closed workspace") || !tt.HasText("Open group session") || tt.HasText("Closed group session") {
				t.Fatal("fixture does not start with one open and one closed workspace")
			}
			check(false)
			label := i18n.T(i18n.KeySidebarArchivedSection)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			check(true)
			m.Snapshot.SelectedID = "active-closed"
			tt.Frame()
			check(true)
			m.Snapshot.WorkspaceLayout.HostGeneration++
			tt.Frame()
			check(true)
			m.Collapsed = true
			tt.Frame()
			m.Collapsed = false
			tt.Frame()
			check(true)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			check(false)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			check(true)
			tt = ui.NewTester(func(c *ui.Context) { m.Batch.Enabled = activeBatch; View(c, m) }, 280, 1000)
			check(false)
			if err := tt.Click(workspaceText("多选会话", "Select sessions")); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if err := tt.Click(workspaceText("全选可见", "Select visible")); err != nil {
				t.Fatal(err)
			}
			if len(batchSelected) != 1 || batchSelected[0].ID != "active-open" {
				t.Fatalf("collapsed archive batch targets=%v", batchSelected)
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(workspaceText("全选可见", "Select visible")); err != nil {
				t.Fatal(err)
			}
			if len(batchSelected) != 2 || batchSelected[0].ID != "active-open" || batchSelected[1].ID != "archived" {
				t.Fatalf("expanded archive batch targets=%v", batchSelected)
			}
		})
	}
}
