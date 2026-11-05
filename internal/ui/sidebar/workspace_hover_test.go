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

func workspaceHoverFixture(t *testing.T) (*ui.Tester, *Model, time.Time) {
	t.Helper()
	locale := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(locale) })
	items := []session.Item{{SessionID: "current-a", Title: "Conversation A"}, {SessionID: "saved-b", Title: "Conversation B"}}
	m := &Model{Snapshot: session.ViewState{SelectedID: "current-a", Items: items,
		Workspaces: []session.WorkspaceRow{
			{WorkspaceID: "a", Title: "Workspace A full title", Path: "/remote/account/alpha", CreatedAt: "2026-02-03T04:05:06Z", SessionIDs: []string{"current-a"}},
			{WorkspaceID: "b", Title: "Workspace B full title", Path: `/different-host/beta`, CreatedAt: "2026-04-05T06:07:08Z", SessionIDs: []string{"saved-b"}},
		}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 11, Collapsed: map[string]bool{"a": false, "b": false}}},
		OnBeginWorkspaceRename: func(string) {}, OnBeginWorkspaceRemoval: func(string) {},
		OnCreateWorkspaceSession: func(uint64, string) {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Column(c).Width(280).FillHeight().Children(func() { workspaceRowsView(c, *m, items, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.Transparent) })
		})
	}, 760, 360)
	start := time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)
	tt.SetTime(start)
	return tt, m, start
}

func moveWorkspaceTitle(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok || r.W <= 0 || r.H <= 0 {
		t.Fatalf("missing workspace title %q: %+v", label, r)
	}
	tt.Move(r.X+40, r.Y+r.H/2)
	tt.Frame()
	return r
}

func TestWorkspaceHoverDelayFullContentCopyAndKeyboard(t *testing.T) {
	tt, m, start := workspaceHoverFixture(t)
	mutations := 0
	m.OnToggleWorkspaceGroup = func(string) { mutations++ }
	m.OnSelect = func(string) { mutations++ }
	tt.SetClipboard("unchanged")
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(799 * time.Millisecond))
	if tt.HasText("Workspace preview:") || tt.Clipboard() != "unchanged" {
		t.Fatal("workspace preview opened before 800ms")
	}
	tt.SetTime(start.Add(800 * time.Millisecond))
	preview, ok := tt.Find("Copy workspace path a")
	if !ok || preview.W != 244 {
		t.Fatalf("workspace preview missing/wrong width: %+v texts=%q", preview, tt.Texts())
	}
	for _, text := range []string{"Workspace preview: a", m.Snapshot.Workspaces[0].Path, workspaceCreatedLabel(m.Snapshot.Workspaces[0].CreatedAt)} {
		if !tt.HasText(text) {
			t.Fatalf("workspace card missing %q: %q", text, tt.Texts())
		}
	}
	// Crossing the 8 DIP gap and dwelling inside the card must preserve the
	// copy target, without navigation or a workspace disclosure side effect.
	tt.Move(284, preview.Y+preview.H/2)
	tt.SetTime(start.Add(900 * time.Millisecond))
	tt.Move(preview.X+preview.W/2, preview.Y+preview.H/2)
	tt.SetTime(start.Add(2 * time.Second))
	if err := tt.Click("Copy workspace path a"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != m.Snapshot.Workspaces[0].Path || !tt.HasText("Copied") || mutations != 0 || m.Snapshot.SelectedID != "current-a" {
		t.Fatalf("copy=%q copied=%t mutations=%d selected=%q", tt.Clipboard(), tt.HasText("Copied"), mutations, m.Snapshot.SelectedID)
	}
	feedback, _ := tt.Find("Copy workspace path a")
	if feedback.H != preview.H {
		t.Fatalf("copy feedback resized card: before=%+v after=%+v", preview, feedback)
	}
	tt.SetTime(start.Add(2999 * time.Millisecond))
	if !tt.HasText("Copied") {
		t.Fatal("copy feedback ended before one second")
	}
	tt.SetTime(start.Add(3 * time.Second))
	if tt.HasText("Copied") || !tt.HasText(m.Snapshot.Workspaces[0].Path) {
		t.Fatal("copy feedback did not restore content")
	}
	for index, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		tt.SetClipboard("keyboard guard")
		tt.Key(0, key)
		tt.Frame()
		if tt.Clipboard() != m.Snapshot.Workspaces[0].Path || mutations != 0 {
			t.Fatalf("copy key %v did not copy exact Host path", key)
		}
		tt.SetTime(start.Add(time.Duration(4+index) * time.Second))
	}
}

func TestWorkspaceHoverGroupedTransferLeaveAndMenuSuppression(t *testing.T) {
	tt, m, start := workspaceHoverFixture(t)
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(workspaceHoverDelay))
	if !tt.HasText("Workspace preview: a") {
		t.Fatal("first card missing")
	}
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[1].Title)
	if !tt.HasText("Workspace preview: b") || tt.HasText("Workspace preview: a") {
		t.Fatal("warm group did not replace its card immediately")
	}
	if err := tt.Click(fmt.Sprintf(workspaceText("工作区“%s”的操作", "Actions for workspace “%s”"), "Workspace B full title")); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tt.Menu(), []string{"Reveal workspace in Finder", "Rename", "Delete workspace"}) {
		t.Fatalf("workspace menu %v", tt.Menu())
	}
	tt.SetTime(start.Add(2 * time.Second))
	if tt.HasText("Workspace preview:") {
		t.Fatal("workspace card survived an open menu")
	}
	tt.CloseMenu()
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[1].Title)
	tt.SetTime(start.Add(2799 * time.Millisecond))
	if tt.HasText("Workspace preview:") {
		t.Fatal("menu closing retained the old warm timer")
	}
	tt.SetTime(start.Add(2800 * time.Millisecond))
	if !tt.HasText("Workspace preview: b") {
		t.Fatal("card did not reopen after a fresh 800ms")
	}
	tt.Move(740, 340)
	tt.SetTime(start.Add(2999 * time.Millisecond))
	if !tt.HasText("Workspace preview: b") {
		t.Fatal("200ms card transfer grace ended early")
	}
	tt.SetTime(start.Add(3 * time.Second))
	if tt.HasText("Workspace preview:") {
		t.Fatal("card remained after the 200ms close grace")
	}
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(3799 * time.Millisecond))
	if tt.HasText("Workspace preview:") {
		t.Fatal("leaving the list failed to cool the group")
	}
	tt.SetTime(start.Add(3800 * time.Millisecond))
	if !tt.HasText("Workspace preview: a") {
		t.Fatal("cold group did not reopen")
	}
}

func TestWorkspaceHoverEscapesAndRegistryIdentityRetiresCard(t *testing.T) {
	tt, m, start := workspaceHoverFixture(t)
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(workspaceHoverDelay))
	tt.Key(0, ui.KeyEscape)
	tt.SetTime(start.Add(2 * time.Second))
	if tt.HasText("Workspace preview:") {
		t.Fatal("Escape reopened a still-hovered card")
	}
	tt.Move(740, 340)
	tt.SetTime(start.Add(2300 * time.Millisecond))
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(3100 * time.Millisecond))
	if !tt.HasText("Workspace preview: a") {
		t.Fatal("reenter after Escape did not restore the card")
	}
	m.Snapshot.WorkspaceLayout.HostGeneration++
	tt.Frame()
	if tt.HasText("Workspace preview:") {
		t.Fatal("Host replacement retained the prior card")
	}
	m.Snapshot.Workspaces[0].CreatedAt = ""
	tt.SetTime(start.Add(5 * time.Second))
	if tt.HasText("Workspace preview: a") {
		t.Fatal("workspace without a creation fact invented a card")
	}
}

func TestWorkspaceAnchorPressClosesCardWithoutCopyOrExtraToggle(t *testing.T) {
	for _, target := range []string{"title", "folder", "padding", "other-title"} {
		t.Run(target, func(t *testing.T) {
			tt, m, start := workspaceHoverFixture(t)
			calls := 0
			m.OnToggleWorkspaceGroup = func(string) { calls++ }
			moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
			tt.SetTime(start.Add(workspaceHoverDelay))
			tt.SetClipboard("anchor guard")
			r, _ := tt.Find(m.Snapshot.Workspaces[0].Title)
			if target == "folder" {
				r, _ = tt.Find("Toggle workspace a")
			}
			if target == "other-title" {
				r, _ = tt.Find(m.Snapshot.Workspaces[1].Title)
			}
			x, y := r.X+r.W/2, r.Y+r.H/2
			if target == "padding" {
				x = 2
			}
			tt.Press(x, y)
			if tt.HasText("Workspace preview:") {
				t.Fatal("press on a workspace kept the card open")
			}
			tt.Release(x, y)
			tt.SetTime(start.Add(2 * time.Second))
			if calls != 1 || tt.Clipboard() != "anchor guard" || tt.HasText("Workspace preview:") {
				t.Fatalf("anchor press leaked calls=%d clipboard=%q texts=%q", calls, tt.Clipboard(), tt.Texts())
			}
		})
	}
}

func TestWorkspaceCreatedLabelUsesApplicationLocale(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	// Parse a timestamp in this viewer's own zone, keeping the expected clock
	// portable between test machines while changing only the app language.
	date := time.Date(2026, 2, 3, 4, 5, 0, 0, time.Local).Format(time.RFC3339)
	for _, test := range []struct{ locale, want string }{{"en", "Created 2026-2-3 04:05"}, {"zh", "创建于 2026年2月3日 04:05"}} {
		i18n.SetLocale(test.locale)
		if got := workspaceCreatedLabel(date); got != test.want {
			t.Fatalf("%s created label %q want %q", test.locale, got, test.want)
		}
	}
	if workspaceCreatedLabel("") != "" || workspaceCreatedLabel("invalid") != "" {
		t.Fatal("unknown creation time was fabricated")
	}
}
