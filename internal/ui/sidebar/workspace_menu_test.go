package sidebar

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceMenuDirectActionsAddressClickedRow(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	var renamed, removed, toggled []string
	items := []session.Item{{SessionID: "current-a", Title: "Current in A"}, {SessionID: "other-b", Title: "Other in B"}}
	m := Model{
		Snapshot: session.ViewState{SelectedID: "current-a", Items: items,
			Workspaces: []session.WorkspaceRow{
				{WorkspaceID: "a", Title: "Workspace A", Path: "/host/a", SessionIDs: []string{"current-a"}},
				{WorkspaceID: "b", Title: "Workspace B", Path: "/host/b", SessionIDs: []string{"other-b"}},
			}, WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: map[string]bool{"a": false, "b": false}}},
		OnBeginWorkspaceRename:  func(id string) { renamed = append(renamed, id) },
		OnBeginWorkspaceRemoval: func(id string) { removed = append(removed, id) },
		OnToggleWorkspaceGroup:  func(id string) { toggled = append(toggled, id) },
		OnSelectWorkspace:       func(string) { t.Fatal("workspace action selected its target") },
		OnSelect:                func(string) { t.Fatal("workspace action selected a conversation") },
		OnManageWorkspace:       func(string) { t.Fatal("workspace menu retained the intermediate manager") },
		OnMoveWorkspaceBefore:   func(string, string) { t.Fatal("workspace menu retained a move action") },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(30, 30, 30), ui.RGB(100, 100, 100), ui.Transparent) })
	}, 280, 260)
	trigger := fmt.Sprintf(workspaceText("工作区“%s”的操作", "Actions for workspace “%s”"), "Workspace B")
	want := []string{workspaceText("在 Finder 中打开工作区", "Reveal workspace in Finder"), workspaceText("重命名", "Rename"), workspaceText("删除工作区", "Delete workspace")}
	open := func() {
		t.Helper()
		if err := tt.Click(trigger); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tt.Menu(), want) {
			t.Fatalf("workspace menu=%v want=%v", tt.Menu(), want)
		}
		if len(toggled) != 0 || m.Snapshot.SelectedID != "current-a" {
			t.Fatal("opening the workspace menu changed expansion or selection")
		}
	}
	// Native menu dismissal performs no rename/removal and keeps the row intact.
	open()
	tt.CloseMenu()
	if tt.Menu() != nil || len(renamed)+len(removed)+len(toggled) != 0 {
		t.Fatalf("cancel leaked actions rename%v remove%v toggles%v", renamed, removed, toggled)
	}
	open()
	if err := tt.ChooseMenuItem(want[1]); err != nil {
		t.Fatal(err)
	}
	open()
	if err := tt.ChooseMenuItem(want[2]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(renamed, []string{"b"}) || !reflect.DeepEqual(removed, []string{"b"}) || len(toggled) != 0 || m.Snapshot.SelectedID != "current-a" {
		t.Fatalf("workspace menu targets rename%v remove%v toggle%v selected%q", renamed, removed, toggled, m.Snapshot.SelectedID)
	}
}
