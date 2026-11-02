package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestGroupPresenceKeepsOutgoingRowsInertWhileTrackCollapses(t *testing.T) {
	folded, clicks := false, 0
	m := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7}}}
	group := session.WorkspaceGroup{Key: "w", Kind: "workspace", Items: []session.Item{{SessionID: "a", Title: "Alpha"}}, Workspace: session.WorkspaceRow{WorkspaceID: "w", SessionIDs: []string{"a"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		host := ui.Column(c.Key("host")).Width(240).Gap(0)
		host.Children(func() {
			ui.Box(c).Height(34)
			workspaceGroupSessionRun(c, host, m, group, folded, func(rows workspaceWatermarkRows) {
				for _, row := range rows.Rows {
					if ui.Button(c, row.Item.Title).Height(32).Clicked() {
						clicks++
					}
				}
			})
			ui.Box(c).Height(10).Label("Following group")
		})
	}, 280, 250)
	now := time.Now()
	tt.SetTime(now)
	before, _ := tt.Find("Following group")
	row, ok := tt.Find("Alpha")
	if !ok {
		t.Fatal("open session missing")
	}
	folded = true
	tt.Frame()
	tt.ClickAt(row.X+10, row.Y+10)
	if clicks != 0 {
		t.Fatal("exiting session kept action authority")
	}
	if tt.HasText("Alpha") {
		t.Fatal("folded outgoing rows retained semantic/input visibility")
	}
	tt.SetTime(now.Add(50 * time.Millisecond))
	middle, _ := tt.Find("Following group")
	if middle.Y <= 34 || middle.Y >= before.Y {
		t.Fatalf("track did not shrink: before=%+v during=%+v", before, middle)
	}
	tt.SetTime(now.Add(210 * time.Millisecond))
	if tt.HasText("Alpha") {
		t.Fatal("outgoing session remained after presence exit")
	}
	settled, _ := tt.Find("Following group")
	if settled.Y != 34 {
		t.Fatalf("collapsed run retained height: %+v", settled)
	}
	folded = false
	tt.Frame()
	tt.SetTime(now.Add(430 * time.Millisecond))
	if err := tt.Click("Alpha"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatal("reopened row did not regain action authority")
	}
}

func TestGroupPresenceInitiallyClosedAndReducedMotion(t *testing.T) {
	folded := true
	m := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7}}}
	group := session.WorkspaceGroup{Key: "w", Kind: "workspace", Items: []session.Item{{SessionID: "a", Title: "Alpha"}}, Workspace: session.WorkspaceRow{WorkspaceID: "w", SessionIDs: []string{"a"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		host := ui.Column(c.Key("host")).Width(240)
		host.Children(func() {
			workspaceGroupSessionRun(c, host, m, group, folded, func(rows workspaceWatermarkRows) {
				for _, row := range rows.Rows {
					ui.Text(c, row.Item.Title).Height(32)
				}
			})
		})
	}, 280, 250)
	if tt.HasText("Alpha") {
		t.Fatal("initially closed group mounted hidden sessions")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	folded = false
	tt.Frame()
	if !tt.HasText("Alpha") {
		t.Fatal("reduced-motion expand did not mount")
	}
	folded = true
	tt.Frame()
	if tt.HasText("Alpha") {
		t.Fatal("reduced-motion fold retained outgoing rows")
	}
}

func TestSidebarScrollbarLingerUsesColumnBoundaryAndOneDeadline(t *testing.T) {
	now := time.Unix(100, 0)
	state := sidebarScrollbarState{}
	if visible, after := state.update(now, false); visible || after != 0 {
		t.Fatal("bar visible before column entered")
	}
	if visible, after := state.update(now, true); !visible || after != 0 {
		t.Fatal("enter did not reveal bar")
	}
	if visible, after := state.update(now, false); !visible || after != 2*time.Second {
		t.Fatal("leave did not arm exact linger")
	}
	if visible, after := state.update(now.Add(time.Second), false); !visible || after != time.Second {
		t.Fatal("outside move restarted deadline")
	}
	state.update(now.Add(1500*time.Millisecond), true)
	if !state.HideAt.IsZero() {
		t.Fatal("reentry did not cancel deadline")
	}
	state.update(now.Add(1800*time.Millisecond), false)
	if visible, after := state.update(now.Add(3799*time.Millisecond), false); !visible || after != time.Millisecond {
		t.Fatal("bar hid before linger ended")
	}
	if visible, after := state.update(now.Add(3800*time.Millisecond), false); visible || after != 0 {
		t.Fatal("bar linger did not settle")
	}

	var shown bool
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c.Key("sidebar")).Width(240).Height(300)
		shown = sidebarScrollbarLinger(c, root)
		root.Children(func() { ui.Box(c).FillWidth().Height(20).Label("Header outside scroll") })
	}, 500, 400)
	tt.SetTime(now)
	tt.Move(20, 10)
	if !shown {
		t.Fatal("header inside column did not reveal scrollbars")
	}
	tt.Move(350, 10)
	tt.SetTime(now.Add(1999 * time.Millisecond))
	if !shown {
		t.Fatal("column-exit bar hid too soon")
	}
	tt.SetTime(now.Add(2 * time.Second))
	if shown {
		t.Fatal("bar ignored column geometry outside scroll")
	}
}
