package sidebar

import (
	"fmt"
	"reflect"
	"testing"
	"time"

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
	want := []string{workspaceText("重命名", "Rename"), workspaceText("删除工作区", "Delete workspace")}
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
	if err := tt.ChooseMenuItem(want[0]); err != nil {
		t.Fatal(err)
	}
	open()
	if err := tt.ChooseMenuItem(want[1]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(renamed, []string{"b"}) || !reflect.DeepEqual(removed, []string{"b"}) || len(toggled) != 0 || m.Snapshot.SelectedID != "current-a" {
		t.Fatalf("workspace menu targets rename%v remove%v toggle%v selected%q", renamed, removed, toggled, m.Snapshot.SelectedID)
	}
}

func TestWorkspaceRemovalAnimatedExitAndSiblingGlide(t *testing.T) {
	items := []session.Item{
		{SessionID: "sa", Title: "Session A"},
		{SessionID: "sb", Title: "Session B"},
		{SessionID: "sc", Title: "Session C"},
	}
	workspaces := []session.WorkspaceRow{
		{WorkspaceID: "a", Title: "Workspace A", Path: "/host/a", SessionIDs: []string{"sa"}},
		{WorkspaceID: "b", Title: "Workspace B", Path: "/host/b", SessionIDs: []string{"sb"}},
		{WorkspaceID: "c", Title: "Workspace C", Path: "/host/c", SessionIDs: []string{"sc"}},
	}
	collapsed := map[string]bool{"a": true, "b": true, "c": true}
	var tt *ui.Tester
	tt = ui.NewTester(func(c *ui.Context) {
		m := Model{
			Snapshot: session.ViewState{
				WorkspaceReady:  true,
				Items:           items,
				Workspaces:      workspaces,
				WorkspaceLayout: session.WorkspaceLayoutState{Collapsed: collapsed},
			},
		}
		ui.Column(c).Fill().Children(func() {
			workspaceRowsView(c, m, items, ui.RGB(230, 230, 230), ui.RGB(140, 140, 140), ui.Transparent)
		})
	}, 280, 320)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	rectB, okB := tt.Find("Workspace B")
	rectCBefore, okC := tt.Find("Workspace C")
	if !okB || !okC || rectCBefore.Y <= rectB.Y {
		t.Fatalf("initial layout invalid: B=%+v C=%+v", rectB, rectCBefore)
	}
	// Remove Workspace B at t=start: it leaves the live tree immediately while
	// its inert exit ghost fades (100ms) and Workspace C smoothly glides up (200ms).
	workspaces = []session.WorkspaceRow{workspaces[0], workspaces[2]}
	tt.Frame()
	if tt.HasText("Workspace B") {
		t.Fatal("removed workspace remained in live accessibility/query tree")
	}
	tt.SetTime(start.Add(16 * time.Millisecond))
	rectCMid, ok := tt.Find("Workspace C")
	if !ok {
		t.Fatal("Workspace C missing during glide")
	}
	if !(rectCMid.Y < rectCBefore.Y && rectCMid.Y > rectB.Y) {
		t.Fatalf("Workspace C did not smoothly glide at 16ms: got Y=%v, want between %v and %v", rectCMid.Y, rectB.Y, rectCBefore.Y)
	}
	tt.SetTime(start.Add(100 * time.Millisecond))
	rectCHalf, _ := tt.Find("Workspace C")
	if !(rectCHalf.Y < rectCMid.Y && rectCHalf.Y > rectB.Y) {
		t.Fatalf("Workspace C did not continue gliding at 100ms: got Y=%v, want between %v and %v", rectCHalf.Y, rectB.Y, rectCMid.Y)
	}
	tt.SetTime(start.Add(220 * time.Millisecond))
	rectCAfter, _ := tt.Find("Workspace C")
	if rectCAfter.Y != rectB.Y {
		t.Fatalf("Workspace C final Y=%v, want %v", rectCAfter.Y, rectB.Y)
	}
}
