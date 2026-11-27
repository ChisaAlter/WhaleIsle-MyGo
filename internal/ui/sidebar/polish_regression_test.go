package sidebar

import (
	"fmt"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func colorL1(a, b color.RGBA) int {
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
}

func assertNoBlueLeftBar(t *testing.T, tt *ui.Tester, name string, r ui.Rect) {
	t.Helper()
	img := tt.Image()
	cy := int(r.Y + r.H/2)
	for dx := 0; dx <= 6; dx++ {
		x := int(r.X) + dx
		if x < 0 || x >= img.Bounds().Dx() {
			continue
		}
		p := img.RGBAAt(x, cy)
		// Any blue primary/sidebar-primary vertical bar would have strong blue
		// dominance (B - R >= 35). Neutral coss backgrounds keep channels balanced.
		if int(p.B)-int(p.R) >= 35 {
			t.Fatalf("%s painted a blue left accent bar at (%d,%d): %+v (rect=%+v)", name, x, cy, p, r)
		}
	}
}

func TestSidebarNoBlueActiveBarAndStateLadder(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	i18n.SetLocale(i18n.LocaleZH)
	design.SetGlass(false, nil)
	chrome.ConfigureBackdrop("", wallpapers.Defaults())

	for _, dark := range []bool{false, true} {
		themeID := theme.IDDeepseekLight
		if dark {
			themeID = theme.IDDeepseekDark
		}
		t.Run(fmt.Sprintf("dark%v", dark), func(t *testing.T) {
			// 1. Nav panel rows: no blue left bar + idle < hover < selected ladder.
			m := Model{
				ThemeID:        themeID,
				SystemDark:     dark,
				ActivePanel:    "plugins",
				OnOpenMarket:   func() {},
				OnOpenSchedule: func() {},
				OnCreate:       func() {},
				OnSelect:       func(string) {},
				Snapshot: session.ViewState{
					WorkspaceReady: true,
					SelectedID:     "sess-sel",
					Workspaces: []session.WorkspaceRow{
						{WorkspaceID: "ws1", Title: "Workspace", Path: "/ws", SessionIDs: []string{"sess-sel", "sess-idle"}},
					},
					Items: []session.Item{
						{SessionID: "sess-sel", Title: "Selected Session", Cwd: "/ws", UpdatedAt: 1700000000000},
						{SessionID: "sess-idle", Title: "Idle Session", Cwd: "/ws", UpdatedAt: 1700000000000},
					},
				},
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 800)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			if dark {
				tt.SetDark(true)
			}
			tt.Frame()

			sidebarBase := tt.Image().RGBAAt(274, 790)

			// Nav panel check.
			pluginsRect := navigationPanelRect(t, tt, i18n.T(i18n.KeySidebarMarket))
			scheduleRect := navigationPanelRect(t, tt, i18n.T(i18n.KeySidebarSchedule))
			assertNoBlueLeftBar(t, tt, "nav-panel/plugins", pluginsRect)

			tt.Move(274, 790)
			tt.Frame()
			navIdle := tt.Image().RGBAAt(int(scheduleRect.X+scheduleRect.W-10), int(scheduleRect.Y+scheduleRect.H/2))
			navSelected := tt.Image().RGBAAt(int(pluginsRect.X+pluginsRect.W-10), int(pluginsRect.Y+pluginsRect.H/2))

			tt.Move(scheduleRect.X+scheduleRect.W/2, scheduleRect.Y+scheduleRect.H/2)
			tt.Frame()
			navHover := tt.Image().RGBAAt(int(scheduleRect.X+scheduleRect.W-10), int(scheduleRect.Y+scheduleRect.H/2))

			dIdle := colorL1(navIdle, sidebarBase)
			dHover := colorL1(navHover, sidebarBase)
			dSel := colorL1(navSelected, sidebarBase)
			if !(dIdle < dHover && dHover < dSel) {
				t.Fatalf("nav state ladder broken (dark=%v): base=%v idle=%v(d=%d) hover=%v(d=%d) selected=%v(d=%d)",
					dark, sidebarBase, navIdle, dIdle, navHover, dHover, navSelected, dSel)
			}

			// 2. Session rows: no blue left bar + idle < hover < selected ladder.
			m.ActivePanel = ""
			tt.Move(274, 790)
			tt.Frame()
			selSess, ok := tt.Find("Selected Session")
			if !ok {
				t.Fatalf("missing Selected Session: %v", tt.Texts())
			}
			idleSess, ok := tt.Find("Idle Session")
			if !ok {
				t.Fatalf("missing Idle Session: %v", tt.Texts())
			}
			selRowRect := ui.Rect{X: 12, Y: selSess.Y - 4, W: 256, H: 32}
			assertNoBlueLeftBar(t, tt, "session-row/selected", selRowRect)

			sessIdlePx := tt.Image().RGBAAt(18, int(idleSess.Y+idleSess.H/2))
			sessSelPx := tt.Image().RGBAAt(18, int(selSess.Y+selSess.H/2))
			tt.Move(idleSess.X+20, idleSess.Y+idleSess.H/2)
			tt.Frame()
			sessHoverPx := tt.Image().RGBAAt(18, int(idleSess.Y+idleSess.H/2))
			dsIdle := colorL1(sessIdlePx, sidebarBase)
			dsHover := colorL1(sessHoverPx, sidebarBase)
			dsSel := colorL1(sessSelPx, sidebarBase)
			if !(dsIdle < dsHover && dsHover < dsSel) {
				t.Fatalf("session state ladder broken (dark=%v): base=%v idle=%v(d=%d) hover=%v(d=%d) selected=%v(d=%d)",
					dark, sidebarBase, sessIdlePx, dsIdle, sessHoverPx, dsHover, sessSelPx, dsSel)
			}

			// 3. Search result rows: no blue left bar + idle < hover < selected ladder.
			searchM := Model{
				ThemeID:    themeID,
				SystemDark: dark,
				OnSelect:   func(string) {},
				Snapshot: session.ViewState{
					SelectedID: "search-sel",
				},
			}
			ttSearch := ui.NewTester(func(c *ui.Context) {
				defer withSidebarTheme(c, searchM)()
				t := tok(c)
				ui.Column(c).Fill().Background(t.Sidebar).Padding(8).Gap(4).Children(func() {
					renderSearchResultRow(c, searchM, sessionSearchRow{
						Item:      session.Item{SessionID: "search-sel", Title: "Search Selected"},
						Workspace: "proj",
					}, t.Foreground, t.MutedForeground)
					renderSearchResultRow(c, searchM, sessionSearchRow{
						Item:      session.Item{SessionID: "search-idle", Title: "Search Idle"},
						Workspace: "proj",
					}, t.Foreground, t.MutedForeground)
				})
			}, 280, 160)
			ttSearch.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			if dark {
				ttSearch.SetDark(true)
			}
			ttSearch.Frame()

			searchBase := ttSearch.Image().RGBAAt(2, 150)
			rSel, ok := ttSearch.Find("Open search result search-sel")
			if !ok {
				t.Fatalf("missing selected search row: %v", ttSearch.Texts())
			}
			rIdle, ok := ttSearch.Find("Open search result search-idle")
			if !ok {
				t.Fatalf("missing idle search row: %v", ttSearch.Texts())
			}
			assertNoBlueLeftBar(t, ttSearch, "search-row/selected", rSel)

			ttSearch.Move(2, 150)
			ttSearch.Frame()
			srIdlePx := ttSearch.Image().RGBAAt(int(rIdle.X+rIdle.W-8), int(rIdle.Y+rIdle.H/2))
			srSelPx := ttSearch.Image().RGBAAt(int(rSel.X+rSel.W-8), int(rSel.Y+rSel.H/2))
			ttSearch.Move(rIdle.X+rIdle.W/2, rIdle.Y+rIdle.H/2)
			ttSearch.Frame()
			srHoverPx := ttSearch.Image().RGBAAt(int(rIdle.X+rIdle.W-8), int(rIdle.Y+rIdle.H/2))
			drIdle := colorL1(srIdlePx, searchBase)
			drHover := colorL1(srHoverPx, searchBase)
			drSel := colorL1(srSelPx, searchBase)
			if !(drIdle < drHover && drHover < drSel) {
				t.Fatalf("search row state ladder broken (dark=%v): base=%v idle=%v(d=%d) hover=%v(d=%d) selected=%v(d=%d)",
					dark, searchBase, srIdlePx, drIdle, srHoverPx, drHover, srSelPx, drSel)
			}
		})
	}
}

func TestSidebarRowsZeroHoverCompression(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prevLocale) })
	i18n.SetLocale(i18n.LocaleEN)

	// 1. rowSession keeps title rect invariant on hover (trailing actions stay Absolute).
	m := Model{
		ThemeID:   theme.IDDeepseekLight,
		OnSelect:  func(string) {},
		OnArchive: func(string) {},
		OnPin:     func(string) {},
		Snapshot: session.ViewState{
			Items: []session.Item{
				{SessionID: "s1", Title: "Long session title for hover compression check", UpdatedAt: 1700000000000},
			},
		},
	}
	ttSess := ui.NewTester(func(c *ui.Context) {
		t := tok(c)
		ui.Column(c).Fill().Padding(8).Children(func() {
			rowSession(c, m.Snapshot.Items[0], 0, m, t.Foreground, t.MutedForeground, t.SidebarBorder)
		})
	}, 280, 80)
	ttSess.SetPreferences(ui.Preferences{ReduceMotion: true})
	titleRest, ok := ttSess.Find("Long session title for hover compression check")
	if !ok {
		t.Fatalf("missing session title: %v", ttSess.Texts())
	}
	ttSess.Move(titleRest.X+10, titleRest.Y+titleRest.H/2)
	ttSess.Frame()
	titleHover, _ := ttSess.Find("Long session title for hover compression check")
	if titleHover != titleRest {
		t.Fatalf("rowSession hover compressed/shifted title: rest=%+v hover=%+v", titleRest, titleHover)
	}

	// 2. rowArchivedSession keeps title rect invariant on hover.
	archM := Model{
		ThemeID:     theme.IDDeepseekLight,
		OnUnarchive: func(string) {},
		OnDelete:    func(string) {},
		Snapshot: session.ViewState{
			Items: []session.Item{
				{SessionID: "a1", Title: "Archived session hover title", UpdatedAt: 1700000000000},
			},
		},
	}
	ttArch := ui.NewTester(func(c *ui.Context) {
		t := tok(c)
		ui.Column(c).Fill().Padding(8).Children(func() {
			rowArchivedSession(c, "a1", "Archived session hover title", 0, archM, t.MutedForeground)
		})
	}, 280, 80)
	ttArch.SetPreferences(ui.Preferences{ReduceMotion: true})
	archRest, ok := ttArch.Find("Archived session hover title")
	if !ok {
		t.Fatalf("missing archived title: %v", ttArch.Texts())
	}
	ttArch.Move(archRest.X+10, archRest.Y+archRest.H/2)
	ttArch.Frame()
	archHover, _ := ttArch.Find("Archived session hover title")
	if archHover != archRest {
		t.Fatalf("rowArchivedSession hover compressed/shifted title: rest=%+v hover=%+v", archRest, archHover)
	}

	// 3. renderSearchResultRow (archived) keeps title rect invariant on hover.
	ttSearch := ui.NewTester(func(c *ui.Context) {
		t := tok(c)
		ui.Column(c).Fill().Padding(8).Children(func() {
			renderSearchResultRow(c, archM, sessionSearchRow{
				Item:      session.Item{SessionID: "a1", Title: "Archived search result title"},
				Archived:  true,
				Workspace: "ws",
			}, t.Foreground, t.MutedForeground)
		})
	}, 280, 80)
	ttSearch.SetPreferences(ui.Preferences{ReduceMotion: true})
	srRest, ok := ttSearch.Find("Archived search result title")
	if !ok {
		t.Fatalf("missing search result title: %v", ttSearch.Texts())
	}
	ttSearch.Move(srRest.X+10, srRest.Y+srRest.H/2)
	ttSearch.Frame()
	srHover, _ := ttSearch.Find("Archived search result title")
	if srHover != srRest {
		t.Fatalf("renderSearchResultRow hover compressed/shifted title: rest=%+v hover=%+v", srRest, srHover)
	}
}

func TestCollapsedRailSearchNilFocusSafeAndConnectionTriggerZeroShift(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prevLocale) })
	i18n.SetLocale(i18n.LocaleEN)

	// Clicking search in collapsedRail when m.SearchFocus == nil must not panic.
	query := ""
	toggled := 0
	m := Model{
		Collapsed:   true,
		SearchQuery: &query,
		SearchFocus: nil,
		OnToggle:    func() { toggled++ },
	}
	ttRail := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			collapsedRail(c, m, tok(c).Foreground)
		})
	}, 56, 400)
	if err := ttRail.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatalf("click collapsed rail search: %v", err)
	}
	if toggled != 1 {
		t.Fatalf("collapsed rail search did not trigger OnToggle: %d", toggled)
	}

	// Connection trigger label must not shift horizontally when status toggles.
	ready := true
	ttConn := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(8).Children(func() {
			settingsConnectionTrigger(c, Model{
				ConnectionReady: ready,
				OnOpenSettings:  func() {},
			}, false)
		})
	}, 260, 80)
	ttConn.SetPreferences(ui.Preferences{ReduceMotion: true})
	now := time.Unix(4000, 0)
	ttConn.SetTime(now)
	seatLabel := i18n.T(i18n.KeySettingsAccountSeat)
	rConnected, ok := ttConn.Find(seatLabel)
	if !ok {
		t.Fatalf("missing connection trigger label %q: %v", seatLabel, ttConn.Texts())
	}
	ready = false
	ttConn.SetTime(now.Add(100 * time.Millisecond))
	rDisconnected, ok := ttConn.Find(seatLabel)
	if !ok {
		t.Fatalf("missing connection trigger label when disconnected: %v", ttConn.Texts())
	}
	if rDisconnected != rConnected {
		t.Fatalf("connection status shifted settings label: connected=%+v disconnected=%+v", rConnected, rDisconnected)
	}
}

func TestBatchDarkGlassContrastAndUncheckedCheckboxVisibility(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	i18n.SetLocale(i18n.LocaleZH)
	design.SetGlass(true, nil)

	m := Model{
		ThemeID:    theme.IDDeepseekDark,
		SystemDark: true,
		Batch: session.SessionBatch{
			Enabled:  true,
			Selected: map[string]string{"s1": "One"},
		},
		OnBatchEdit:    func(string, []session.SessionBatchTarget) {},
		OnBatchRequest: func(string) {},
		Snapshot: session.ViewState{
			WorkspaceReady: true,
			SelectedID:     "s1",
			Workspaces: []session.WorkspaceRow{
				{WorkspaceID: "ws1", Title: "Workspace", Path: "/ws", SessionIDs: []string{"s1", "s2"}},
			},
			Items: []session.Item{
				{SessionID: "s1", Title: "One", Cwd: "/ws", UpdatedAt: 1700000000000},
				{SessionID: "s2", Title: "Two", Cwd: "/ws", UpdatedAt: 1700000000000},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Box(c).Fill().Background(ui.RGB(20, 22, 26)).Children(func() {
			View(c, m)
		})
	}, 280, 600)
	tt.SetDark(true)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()

	// Unchecked checkbox row ("s2") must paint a visible border outline against dark glass.
	checkLabel := i18n.WSf(i18n.KeyWSSelectionSessionAria, "name", "Two")
	checkRect, ok := tt.Find(checkLabel)
	if !ok {
		t.Fatalf("missing unchecked batch checkbox %q: %v", checkLabel, tt.Texts())
	}
	img := tt.Image()
	bgPx := img.RGBAAt(int(checkRect.X)-6, int(checkRect.Y+checkRect.H/2))
	maxDelta := 0
	for y := int(checkRect.Y) - 2; y <= int(checkRect.Y+checkRect.H)+2; y++ {
		for x := int(checkRect.X) - 2; x <= int(checkRect.X+checkRect.W)+2; x++ {
			if d := colorL1(img.RGBAAt(x, y), bgPx); d > maxDelta {
				maxDelta = d
			}
		}
	}
	if maxDelta < 45 {
		t.Fatalf("unchecked batch checkbox nearly invisible on dark glass: maxDelta=%d bg=%v rect=%+v", maxDelta, bgPx, checkRect)
	}

	// Archive and Delete batch buttons must render crisp foreground ink pixels on dark glass.
	for _, label := range []string{i18n.T(i18n.KeyWSSelectionArchive), i18n.T(i18n.KeyWSSelectionDelete)} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing batch button %q: %v", label, tt.Texts())
		}
		btnBg := img.RGBAAt(int(r.X+4), int(r.Y+4))
		bestContrast := 1.0
		for y := int(r.Y) + 3; y < int(r.Y+r.H)-3; y++ {
			for x := int(r.X) + 3; x < int(r.X+r.W)-3; x++ {
				p := img.RGBAAt(x, y)
				cr := chrome.ContrastRatio(ui.RGB(p.R, p.G, p.B), ui.RGB(btnBg.R, btnBg.G, btnBg.B))
				if cr > bestContrast {
					bestContrast = cr
				}
			}
		}
		if bestContrast < 4.5 {
			t.Fatalf("batch button %q ink contrast %.2f < 4.5 on dark glass (bg=%v rect=%+v)", label, bestContrast, btnBg, r)
		}
	}
}
