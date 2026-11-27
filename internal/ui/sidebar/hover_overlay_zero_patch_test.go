package sidebar

import (
	"fmt"
	"image"
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

func TestSidebarRowActionsWidth(t *testing.T) {
	wsGroup := session.WorkspaceGroup{
		Kind:      "workspace",
		Key:       "ws-1",
		Workspace: session.WorkspaceRow{WorkspaceID: "ws-1", Title: "Documents", Path: "/ws/docs"},
	}
	if got := workspaceRowActionsWidth(Model{}, wsGroup); got != 0 {
		t.Fatalf("0 workspace actions width = %v, want 0", got)
	}
	if got := workspaceRowActionsWidth(Model{OnCreateWorkspaceSession: func(uint64, string) {}}, wsGroup); got != 16 {
		t.Fatalf("1 workspace action (create) width = %v, want 16", got)
	}
	if got := workspaceRowActionsWidth(Model{OnBeginWorkspaceRename: func(string) {}}, wsGroup); got != 16 {
		t.Fatalf("1 workspace action (more) width = %v, want 16", got)
	}
	if got := workspaceRowActionsWidth(Model{
		OnBeginWorkspaceRename:   func(string) {},
		OnCreateWorkspaceSession: func(uint64, string) {},
	}, wsGroup); got != 47 {
		t.Fatalf("2 workspace actions width = %v, want 47", got)
	}
	pinnedGroup := session.WorkspaceGroup{Kind: "pinned", Key: "::pinned"}
	if got := workspaceRowActionsWidth(Model{
		OnBeginWorkspaceRename:   func(string) {},
		OnCreateWorkspaceSession: func(uint64, string) {},
	}, pinnedGroup); got != 0 {
		t.Fatalf("pinned group actions width = %v, want 0", got)
	}
	scratchGroup := session.WorkspaceGroup{Kind: "scratch", Key: "::scratch"}
	if got := workspaceRowActionsWidth(Model{OnNoDirectory: func() {}}, scratchGroup); got != 16 {
		t.Fatalf("scratch 1 action width = %v, want 16", got)
	}

	blank := session.Item{SessionID: "draft", Blank: true}
	if got := sessionRowActionsWidth(Model{OnArchive: func(string) {}, OnPin: func(string) {}}, blank); got != 0 {
		t.Fatalf("blank session actions width = %v, want 0", got)
	}
	item := session.Item{SessionID: "s1", Title: "Session"}
	if got := sessionRowActionsWidth(Model{}, item); got != 24 {
		t.Fatalf("1 session action width = %v, want 24", got)
	}
	if got := sessionRowActionsWidth(Model{OnArchive: func(string) {}}, item); got != 50 {
		t.Fatalf("2 session actions width = %v, want 50", got)
	}
	if got := sessionRowActionsWidth(Model{OnArchive: func(string) {}, OnPin: func(string) {}}, item); got != 76 {
		t.Fatalf("3 session actions width = %v, want 76", got)
	}
	pinnedItem := session.Item{SessionID: "s2", Title: "Pinned", Pinned: true}
	if got := sessionRowActionsWidth(Model{OnArchive: func(string) {}, OnPin: func(string) {}}, pinnedItem); got != 50 {
		t.Fatalf("pinned session without OnUnpin width = %v, want 50", got)
	}
	if got := sessionRowActionsWidth(Model{OnArchive: func(string) {}, OnUnpin: func(string) {}}, pinnedItem); got != 76 {
		t.Fatalf("pinned session with OnUnpin width = %v, want 76", got)
	}
}

func paintDualGradientBackdrop(p *ui.Painter, r ui.Rect) {
	p.FillGradient(r, ui.LinearGradient{
		From:  ui.RGB(0, 215, 255),
		To:    ui.RGB(235, 35, 195),
		Angle: 90,
	}, 0)
}

func pixelL1(a, b color.RGBA) int {
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
}

// TestSidebarHoverActionsZeroBackdropPatchAndZeroShift pins that hovering
// workspace and session rows over a vivid multi-color glass backdrop paints
// zero solid/cyan overlay patches behind the action cluster and causes zero
// layout shift.
func TestSidebarHoverActionsZeroBackdropPatchAndZeroShift(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	i18n.SetLocale(i18n.LocaleEN)
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())

	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark=%v", dark), func(t *testing.T) {
			design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
				if s == design.Dark {
					p.Sidebar = ui.RGBA(20, 24, 34, 0.40)
				} else {
					p.Sidebar = ui.RGBA(248, 250, 255, 0.40)
				}
			})
			themeID := theme.IDDeepseekLight
			if dark {
				themeID = theme.IDDeepseekDark
			}
			items := []session.Item{
				{SessionID: "s1", Title: "Short chat", Cwd: "/ws/docs", UpdatedAt: 1700000000000},
			}
			snap := session.ViewState{
				WorkspaceReady: true,
				Workspaces: []session.WorkspaceRow{
					{WorkspaceID: "ws-docs", Title: "Documents", Path: "/ws/docs", SessionIDs: []string{"s1"}},
				},
				Items: items,
			}

			renderRow := func(withActions bool) (*image.RGBA, ui.Rect, *ui.Tester) {
				m := Model{
					ThemeID:    themeID,
					SystemDark: dark,
					Snapshot:   snap,
				}
				if withActions {
					m.OnBeginWorkspaceRename = func(string) {}
					m.OnCreateWorkspaceSession = func(uint64, string) {}
					m.OnArchive = func(string) {}
					m.OnPin = func(string) {}
				}
				tt := ui.NewTester(func(c *ui.Context) {
					defer withSidebarTheme(c, m)()
					tk := tok(c)
					ui.Column(c).Fill().Draw(paintDualGradientBackdrop).Padding(8).Children(func() {
						workspaceRowsView(c, m, items, tk.Foreground, tk.MutedForeground, tk.SidebarBorder)
					})
				}, 280, 140)
				if dark {
					tt.SetDark(true)
				}
				tt.SetPreferences(ui.Preferences{ReduceMotion: true})
				tt.Frame()
				header, ok := tt.Find("Documents")
				if !ok {
					t.Fatalf("missing workspace row Documents: %v", tt.Texts())
				}
				// Hover on the left of the workspace row so action buttons appear without being individually hovered.
				tt.Move(header.X+12, header.Y+header.H/2)
				tt.Frame()
				cp := image.NewRGBA(tt.Image().Bounds())
				copy(cp.Pix, tt.Image().Pix)
				return cp, header, tt
			}

			refImg, header, _ := renderRow(false)
			actImg, _, ttAct := renderRow(true)

			// In the 15-DIP gap between the two 16px workspace action buttons
			// (Right(8), gap 15 -> button 2 at [W-24, W-8], gap at [W-39, W-24]),
			// pixels must match the bare hovered row without any cyan/opaque patch.
			gapX := int(header.X + header.W - 31)
			midY := int(header.Y + header.H/2)
			gotPx := actImg.RGBAAt(gapX, midY)
			refPx := refImg.RGBAAt(gapX, midY)
			if diff := pixelL1(gotPx, refPx); diff > 2 {
				t.Fatalf("dark=%v workspace action gap painted backdrop patch: got=%v ref=%v L1=%d", dark, gotPx, refPx, diff)
			}

			// Move pointer away and verify zero layout shift across all interactive elements.
			ttAct.Move(2, 130)
			ttAct.Frame()
			ui.AssertHoverFocusZeroShift(t, ttAct)
		})
	}
}

// TestSessionTitleCrawlReservesTrailingActionsWidth verifies that a hovered long
// session title clips strictly left of the trailing action cluster and never
// leaks title foreground ink under the action buttons.
func TestSessionTitleCrawlReservesTrailingActionsWidth(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prevLocale) })
	i18n.SetLocale(i18n.LocaleEN)

	const longTitle = "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"
	m := Model{
		ThemeID:   theme.IDDeepseekLight,
		OnSelect:  func(string) {},
		OnArchive: func(string) {},
		OnPin:     func(string) {},
		Snapshot: session.ViewState{
			Items: []session.Item{
				{SessionID: "long-1", Title: longTitle, UpdatedAt: 1700000000000},
			},
		},
	}
	titleRed := ui.RGB(230, 10, 10)
	mutedGray := ui.RGB(120, 120, 120)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Padding(8).Children(func() {
			rowSession(c, m.Snapshot.Items[0], 0, m, titleRed, mutedGray, ui.Transparent)
		})
	}, 280, 60)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()

	rowRect, ok := tt.Find(longTitle)
	if !ok {
		t.Fatalf("missing session row: %v", tt.Texts())
	}
	tt.Move(rowRect.X+24, rowRect.Y+rowRect.H/2)
	tt.SetTime(time.Now().Add(10 * time.Second))
	tt.Frame()

	moreRect, ok := tt.Find(sessionActionsLabel("long-1", longTitle))
	if !ok {
		t.Fatalf("missing hovered action button: %v", tt.Texts())
	}
	img := tt.Image()
	foundLeftInk := false
	for y := int(rowRect.Y); y < int(rowRect.Y+rowRect.H); y++ {
		for x := int(rowRect.X); x < int(rowRect.X+rowRect.W); x++ {
			px := img.RGBAAt(x, y)
			isTitleInk := int(px.R) > int(px.G)+40 && int(px.R) > int(px.B)+40
			if !isTitleInk {
				continue
			}
			if x >= int(moreRect.X)-2 {
				t.Fatalf("crawling title ink leaked into action cluster at x=%d (actions start at %v)", x, moreRect.X)
			}
			foundLeftInk = true
		}
	}
	if !foundLeftInk {
		t.Fatal("expected title ink to remain visible left of the action cluster")
	}
}

// TestConnectionTriggerFadeUsesOpacityNoSolidPatch verifies that the connection
// status seat fades via real Opacity rather than painting an opaque t.Sidebar
// rectangle over translucent or colorful backdrops.
func TestConnectionTriggerFadeUsesOpacityNoSolidPatch(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prevLocale) })
	i18n.SetLocale(i18n.LocaleEN)

	ready := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for x := r.X; x < r.X+r.W; x += 4 {
				col := ui.RGB(20, 220, 250)
				if int(x/4)%2 == 1 {
					col = ui.RGB(250, 30, 170)
				}
				p.Fill(ui.Rect{X: x, Y: r.Y, W: 4, H: r.H}, col, 0)
			}
		}).Padding(8).Children(func() {
			settingsConnectionTrigger(c, Model{
				ConnectionReady: ready,
				OnOpenSettings:  func() {},
				OnReconnect:     func() {},
			}, false)
		})
	}, 260, 80)

	t0 := time.Unix(10000, 0)
	tt.SetTime(t0)
	tt.Frame()

	// Transition to disconnected and advance 45ms into the 150ms fade-in.
	ready = false
	tt.SetTime(t0.Add(45 * time.Millisecond))
	tt.Frame()

	img := tt.Image()
	// Sample inside the 36x36 seat on the right edge (row is at x=8..252, seat at x=216..252, y=12..48).
	// If DrawOver painted a solid t.Sidebar cover with alpha ~0.70, the stripe R span would collapse.
	loR, hiR := 255, 0
	for x := 220; x < 248; x++ {
		px := img.RGBAAt(x, 30)
		loR = min(loR, int(px.R))
		hiR = max(hiR, int(px.R))
	}
	if hiR-loR < 150 {
		t.Fatalf("connection seat fade painted an opaque cover over backdrop stripes: R span=%d (want >=150)", hiR-loR)
	}
}
