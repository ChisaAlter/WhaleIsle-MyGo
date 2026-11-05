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

func TestArchivedRowRestTimeHoverCardAndExplicitMenuActions(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	for _, width := range []int{180, 280} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			item := session.Item{SessionID: "archived", Title: "Archived conversation", Cwd: "/fixture/archive", UpdatedAt: float64(time.Now().Add(-2 * time.Hour).UnixMilli())}
			var restored, deleted, selected []string
			m := Model{Snapshot: session.ViewState{Items: []session.Item{item}, ArchivedSessionIDs: []string{item.SessionID}, Workspaces: []session.WorkspaceRow{{WorkspaceID: "member", Path: "/fixture/archive", SessionIDs: []string{item.SessionID}}}},
				OnUnarchive: func(id string) { restored = append(restored, id) }, OnDelete: func(id string) { deleted = append(deleted, id) }, OnSelect: func(id string) { selected = append(selected, id) }}
			tt := ui.NewTester(func(c *ui.Context) {
				root := ui.Column(c).Fill().Padding(8)
				m = beginSessionHover(root, m)
				root.Children(func() { rowArchivedSession(c, item.SessionID, item.Title, 0, m, ui.RGB(100, 100, 100)) })
				renderSessionHover(c, m, root, m.sessionHover.dragging)
			}, width, 230)
			stamp, found := tt.Find("Session time archived")
			if !found || stamp.H != 16 || stamp.X+stamp.W > float32(width-16) {
				t.Fatalf("rest time width%d bounds%+v found%v texts%v", width, stamp, found, tt.Texts())
			}
			start := time.Now()
			tt.SetTime(start)
			tt.Move(20, 24)
			if _, found = tt.Find("Session time archived"); found {
				t.Fatal("hover retained archived time under the menu button")
			}
			more, found := tt.Find(sessionActionsLabel(item.SessionID, item.Title))
			if !found || more.W != 16 || more.H != 16 || more.X+more.W > float32(width-16) {
				t.Fatalf("hover action=%+v found%v", more, found)
			}
			tt.SetTime(start.Add(time.Second))
			if _, found = tt.Find("Session preview: archived"); !found || tt.HasText(item.Cwd) || !tt.HasText("2h ago") || !tt.HasText("Archived") || !tt.HasText(item.Title) {
				t.Fatalf("archived hover card differs from title/time/archive content: %v", tt.Texts())
			}
			// Dismiss the interactive preview before addressing a row in a narrow
			// viewport where the attached card may cover its anchor.
			tt.Key(0, ui.KeyEscape)
			// A casual title click remains inert; only the menu can restore/delete.
			tt.ClickAt(50, 24)
			if len(restored)+len(deleted)+len(selected) != 0 {
				t.Fatal("archived row click caused an action")
			}
			if err := tt.Click(sessionActionsLabel(item.SessionID, item.Title)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.Menu(), []string{i18n.T(i18n.KeySidebarUnarchive), i18n.T(i18n.KeySidebarDelete)}) {
				t.Fatalf("archived menu %v", tt.Menu())
			}
			if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarUnarchive)); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(sessionActionsLabel(item.SessionID, item.Title)); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarDelete)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(restored, []string{item.SessionID}) || !reflect.DeepEqual(deleted, []string{item.SessionID}) || len(selected) != 0 {
				t.Fatalf("menu actions restored%v deleted%v selected%v", restored, deleted, selected)
			}
		})
	}
}

func TestArchivedRowMissingSummaryHasTitleOnlyPreview(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	title := "Missing archived summary"
	m := Model{Snapshot: session.ViewState{ArchivedSessionIDs: []string{"missing"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Fill().Padding(8)
		m = beginSessionHover(root, m)
		root.Children(func() { rowArchivedSession(c, "missing", title, 0, m, ui.RGB(100, 100, 100)) })
		renderSessionHover(c, m, root, m.sessionHover.dragging)
	}, 280, 200)
	if _, found := tt.Find("Session time missing"); found {
		t.Fatal("missing summary fabricated a resting timestamp")
	}
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.Move(20, 24)
	tt.SetTime(start.Add(time.Second))
	if _, found := tt.Find("Session preview: missing"); !found {
		t.Fatal("missing summary lost its title preview")
	}
	if !reflect.DeepEqual(tt.Texts(), []string{title, sessionActionsLabel("missing", title), "Session preview: missing", title, "Archived"}) {
		t.Fatalf("fallback preview fabricated metadata: %v", tt.Texts())
	}
}

func TestSessionMenuInstalledActionOrderAndTargets(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	item := session.Item{SessionID: "row", Title: "Session menu"}
	calls := []string{}
	m := Model{
		OnPin:               func(id string) { calls = append(calls, "pin:"+id) },
		OnBeginRename:       func(id, _ string) { calls = append(calls, "rename:"+id) },
		OnFork:              func(id string) { calls = append(calls, "fork:"+id) },
		OnArchive:           func(id string) { calls = append(calls, "archive:"+id) },
		OnCopySessionID:     func(id string) { calls = append(calls, "copy:"+id) },
		OnMoveSessionBefore: func(string, string) { t.Fatal("installed menu exposed a move-to-top action") },
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Button(c, "Open menu").Menu(sessionRowMenu(m, item)) }, 280, 100)
	keys := []string{i18n.KeySidebarPin, i18n.KeySidebarRename, i18n.KeySidebarFork, i18n.KeySidebarArchive, i18n.KeySidebarCopyID}
	want := make([]string, len(keys))
	for index, key := range keys {
		want[index] = i18n.T(key)
	}
	for _, label := range want {
		if err := tt.Click("Open menu"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tt.Menu(), want) {
			t.Fatalf("session menu order %v want %v", tt.Menu(), want)
		}
		if err := tt.ChooseMenuItem(label); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(calls, []string{"pin:row", "rename:row", "fork:row", "archive:row", "copy:row"}) {
		t.Fatalf("menu targets %v", calls)
	}
}
