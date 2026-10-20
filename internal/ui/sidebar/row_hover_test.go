package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestTitleCrawlOffsets pins the marquee's motion: a title that fits never
// moves, a wider one travels to its overflow and comes back, and the caller only
// keeps redrawing while the offset is actually changing.
func TestTitleCrawlOffsets(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fits := titleCrawl{cell: 200, active: true, since: start}
	if off, moving := fits.offset(start.Add(time.Second), 190); off != 0 || moving {
		t.Fatalf("fitting title moved: off=%v moving=%v", off, moving)
	}
	idle := titleCrawl{cell: 200}
	if off, moving := idle.offset(start, 900); off != 0 || moving {
		t.Fatalf("inactive crawl moved: off=%v moving=%v", off, moving)
	}
	// 400 DIP of overflow at 40 DIP/s: ten seconds to the end.
	c := titleCrawl{cell: 200, active: true, since: start}
	if off, moving := c.offset(start.Add(400*time.Millisecond), 600); off != -16 || !moving {
		t.Fatalf("400ms: off=%v moving=%v, want -16", off, moving)
	}
	if off, moving := c.offset(start.Add(10*time.Second), 600); off != -400 || !moving {
		t.Fatalf("at the end: off=%v moving=%v, want the full overflow", off, moving)
	}
	if off, _ := c.offset(start.Add(10*time.Second+600*time.Millisecond), 600); off != -400 {
		t.Fatalf("hold frame: off=%v, want to stay at the overflow", off)
	}
	if off, _ := c.offset(start.Add(15*time.Second), 600); off != -248 {
		// 15 s: 3.8 s into the 10 s return at 40 DIP/s from the 400 DIP end.
		t.Fatalf("returning: off=%v, want -248", off)
	}
}

// TestSessionTitleCrawlKeepsRestingGeometry pins the integration: the room a
// resting title gets does not change when the crawl machinery takes over.
func TestSessionTitleCrawlKeepsRestingGeometry(t *testing.T) {
	short := "short"
	tt := ui.NewTester(func(c *ui.Context) {
		row := ui.Row(c).FillWidth().Height(32).Clip()
		row.Children(func() { sessionTitleCell(c, row, "s1", short, ui.RGB(10, 10, 10)) })
	}, 200, 60)
	before, ok := tt.Find(short)
	if !ok {
		t.Fatal("title missing")
	}
	tt.Move(100, 16)
	tt.Frame()
	after, ok := tt.Find(short)
	if !ok || after != before {
		t.Fatalf("hovering a fitting title moved it: before=%+v after=%+v", before, after)
	}
	if before.W != 200 {
		t.Fatalf("title cell=%+v, want the full 200 DIP seat", before)
	}
}


// TestSessionRowHoverSwapsTimeForActions pins Rows.module.css .sessionRow: the
// trailing time cell is replaced by bare 16px glyphs while the pointer rests on
// the row, and the glyphs reach the same callbacks as the row menu.
func TestSessionRowHoverSwapsTimeForActions(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var pinned, archived, selected string
	m := Model{
		Snapshot:  session.ViewState{SelectedID: "other"},
		OnPin:     func(id string) { pinned = id },
		OnArchive: func(id string) { archived = id },
		OnSelect:  func(id string) { selected = id },
	}
	item := session.Item{SessionID: "row", Title: "A session title", UpdatedAt: 1}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowSession(c, item, 0, m, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.Transparent)
		})
	}, 280, 60)

	// At rest: the relative time cell, no action glyphs.
	if _, ok := tt.Find("Session time row"); !ok {
		t.Fatalf("resting row lost its time cell; texts=%q", tt.Texts())
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarPin)); ok {
		t.Fatal("pin glyph rendered at rest")
	}

	row, ok := tt.Find("A session title")
	if !ok {
		t.Fatal("row title missing")
	}
	tt.Move(row.X+40, row.Y+row.H/2)
	tt.Frame()

	for _, label := range []string{i18n.T(i18n.KeySidebarPin), i18n.T(i18n.KeySidebarArchive)} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("hovered row lost %q; texts=%q", label, tt.Texts())
		}
	}
	if _, ok := tt.Find("Session time row"); ok {
		t.Fatal("time cell still painted under the action glyphs")
	}

	if err := tt.Click(i18n.T(i18n.KeySidebarPin)); err != nil {
		t.Fatal(err)
	}
	if pinned != "row" {
		t.Fatalf("pin reached %q", pinned)
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarArchive)); err != nil {
		t.Fatal(err)
	}
	if archived != "row" {
		t.Fatalf("archive reached %q", archived)
	}
	if selected != "" {
		t.Fatalf("row action selected the conversation: %q", selected)
	}
}

// TestWorkspaceRowFolderBecomesChevronOnHover pins Rows.module.css .projectRow:
// the folder rests and the disclosure arrow replaces it under the pointer.
func TestWorkspaceRowFolderBecomesChevronOnHover(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var toggled []string
	items := []session.Item{{SessionID: "a", Title: "Alpha"}}
	m := Model{
		Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Whale Isle", SessionIDs: []string{"a"}}}},
		OnToggleWorkspaceGroup: func(id string) { toggled = append(toggled, id) },
	}
	folder := workspaceText("工作区文件夹 ", "Workspace folder ") + "ws"
	chevron := workspaceText("展开箭头 ", "Workspace chevron ") + "ws"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			workspaceRowsView(c, m, items, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.Transparent)
		})
	}, 300, 200)

	if _, ok := tt.Find(folder); !ok {
		t.Fatalf("resting workspace row has no folder; texts=%q", tt.Texts())
	}
	if _, ok := tt.Find(chevron); ok {
		t.Fatal("chevron rendered at rest")
	}
	row, ok := tt.Find("Whale Isle")
	if !ok {
		t.Fatal("workspace row missing")
	}
	tt.Move(row.X+40, row.Y+row.H/2)
	tt.Frame()
	if _, ok := tt.Find(chevron); !ok {
		t.Fatalf("hovered workspace row has no chevron; texts=%q", tt.Texts())
	}
	if _, ok := tt.Find(folder); ok {
		t.Fatal("folder still painted under the chevron")
	}
	toggle := workspaceText("折叠工作区 ", "Toggle workspace ") + "ws"
	if err := tt.Click(toggle); err != nil {
		t.Fatal(err)
	}
	if len(toggled) != 1 || toggled[0] != "ws" {
		t.Fatalf("toggle calls=%v", toggled)
	}
}
