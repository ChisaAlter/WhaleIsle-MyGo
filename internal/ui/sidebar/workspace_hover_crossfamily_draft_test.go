// PREPARED-ONLY TEST PATCH. Install through the coordinated Go overlay; the
// frozen production build is deliberately unchanged during actual captures.
package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func crossFamilyHoverFixture(t *testing.T) (*ui.Tester, *Model, time.Time) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m := &Model{Snapshot: session.ViewState{SelectedID: "active", WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 73}}}
	start := time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)
	workspace := session.WorkspaceRow{WorkspaceID: "workspace", Title: "Full workspace title", Path: "/remote/exact/path", CreatedAt: "2026-01-02T03:04:05Z"}
	item := session.Item{SessionID: "ordinary", Title: "  Exact saved session title  ", UpdatedAt: float64(start.UnixMilli())}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Width(280).FillHeight()
		*m = beginSessionHover(root, *m)
		root.Children(func() {
			list := ui.Column(c.Key("cross-family-list")).Width(280).Gap(2)
			state := workspaceHoverGroup(c, list, *m)
			anchors := []workspaceHoverAnchor{}
			list.Children(func() {
				row := ui.ButtonBase(c.Key("workspace-family-row")).Label("Cross workspace row").FillWidth().Height(34)
				row.Children(func() { ui.Text(c, workspace.Title) })
				anchors = append(anchors, workspaceHoverAnchor{row: row, workspace: workspace, label: workspace.Title, pressed: row.Pressed()})
				sessionRow := ui.ButtonBase(c.Key("session-family-row")).Label("Cross session row").FillWidth().Height(32)
				sessionRow.Children(func() { ui.Text(c, item.Title) })
				sessionHoverCard(c, *m, sessionRow, item, item.Title, "", theme.Colors{}, sessionRow.Pressed())
			})
			registerWorkspaceHover(c, *m, list, state, anchors)
		})
		renderSessionHover(c, *m, root, m.sessionHover.dragging)
	}, 760, 480)
	tt.SetTime(start)
	return tt, m, start
}

func crossFamilyMove(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok || r.W <= 0 || r.H <= 0 {
		t.Fatalf("missing family anchor %q: %+v", label, r)
	}
	tt.Move(r.X+50, r.Y+r.H/2)
	tt.Frame()
}

func TestDraftCrossFamilyWarmTransferIsExclusiveBothDirections(t *testing.T) {
	for _, workspaceFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "workspace-to-session", false: "session-to-workspace"}[workspaceFirst], func(t *testing.T) {
			tt, _, start := crossFamilyHoverFixture(t)
			firstRow, nextRow := "Cross workspace row", "Cross session row"
			firstPreview, nextPreview := "Workspace preview: workspace", "Session preview: ordinary"
			if !workspaceFirst {
				firstRow, nextRow = nextRow, firstRow
				firstPreview, nextPreview = nextPreview, firstPreview
			}
			crossFamilyMove(t, tt, firstRow)
			tt.SetTime(start.Add(799 * time.Millisecond))
			if tt.HasText("Workspace preview:") || tt.HasText("Session preview:") {
				t.Fatal("cold cross-family group opened early")
			}
			tt.SetTime(start.Add(800 * time.Millisecond))
			if !tt.HasText(firstPreview) {
				t.Fatalf("first card missing: %q", tt.Texts())
			}
			crossFamilyMove(t, tt, nextRow)
			if !tt.HasText(nextPreview) || tt.HasText(firstPreview) {
				t.Fatalf("cross-family warm transfer missing or showed two cards: %q", tt.Texts())
			}
			crossFamilyMove(t, tt, firstRow)
			if !tt.HasText(firstPreview) || tt.HasText(nextPreview) {
				t.Fatalf("reverse transfer lost warmth or exclusivity: %q", tt.Texts())
			}
		})
	}
}

func TestDraftCrossFamilyCardPointerCopyAndColdReentry(t *testing.T) {
	tt, _, start := crossFamilyHoverFixture(t)
	crossFamilyMove(t, tt, "Cross workspace row")
	tt.SetTime(start.Add(800 * time.Millisecond))
	card, ok := tt.Find("Copy workspace path workspace")
	if !ok {
		t.Fatal("workspace card missing")
	}
	tt.Move(card.X+card.W/2, card.Y+card.H/2)
	tt.SetTime(start.Add(time.Second))
	if err := tt.Click("Copy workspace path workspace"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != "/remote/exact/path" || !tt.HasText("Copied") || tt.HasText("Session preview:") {
		t.Fatalf("copy did not stay in workspace family: clipboard=%q texts=%q", tt.Clipboard(), tt.Texts())
	}
	crossFamilyMove(t, tt, "Cross session row")
	if tt.HasText("Workspace preview:") || !tt.HasText("Session preview: ordinary") {
		t.Fatalf("workspace feedback delayed session transfer: %q", tt.Texts())
	}
	if err := tt.Click("Copy session title ordinary"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != "  Exact saved session title  " {
		t.Fatalf("session copy altered the exact title: %q", tt.Clipboard())
	}
	tt.Move(740, 460)
	tt.SetTime(start.Add(1199 * time.Millisecond))
	if _, ok := tt.Find("Copy session title ordinary"); !ok {
		t.Fatal("card lost the200ms pointer grace")
	}
	tt.SetTime(start.Add(1200 * time.Millisecond))
	if _, ok := tt.Find("Copy session title ordinary"); ok {
		t.Fatal("card survived leaving list and card")
	}
	crossFamilyMove(t, tt, "Cross workspace row")
	tt.SetTime(start.Add(1999 * time.Millisecond))
	if tt.HasText("Workspace preview:") {
		t.Fatal("cross-family warm state survived leaving the group")
	}
	tt.SetTime(start.Add(2 * time.Second))
	if !tt.HasText("Workspace preview: workspace") {
		t.Fatal("cold reentry did not wait exactly800ms")
	}
}
