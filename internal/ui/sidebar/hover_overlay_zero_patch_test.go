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
// zero solid/cyan overlay patches behind the action cluster, keeps short
// titles un-faded, smoothly fades 40+ char titles with zero ink leak under
// action buttons, and causes zero layout shift.
func TestSidebarHoverActionsZeroBackdropPatchAndZeroShift(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	i18n.SetLocale(i18n.LocaleEN)
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())

	const longTitle = "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM" // 44 chars (>40)

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

			// 1. Workspace row ("Documents"): zero backdrop patch in action area,
			//    short title stays crisp (un-faded), and zero layout shift.
			renderWorkspace := func(wsTitle string, withActions bool) (*image.RGBA, ui.Rect, *ui.Tester) {
				items := []session.Item{
					{SessionID: "s1", Title: "Short chat", Cwd: "/ws/docs", UpdatedAt: 1700000000000},
				}
				snap := session.ViewState{
					WorkspaceReady: true,
					Workspaces: []session.WorkspaceRow{
						{WorkspaceID: "ws-docs", Title: wsTitle, Path: "/ws/docs", SessionIDs: []string{"s1"}},
					},
					Items: items,
				}
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
				header, ok := tt.Find(wsTitle)
				if !ok {
					t.Fatalf("missing workspace row %q: %v", wsTitle, tt.Texts())
				}
				// Hover on the left of the workspace row so action buttons appear without being individually hovered.
				tt.Move(header.X+12, header.Y+header.H/2)
				tt.Frame()
				cp := image.NewRGBA(tt.Image().Bounds())
				copy(cp.Pix, tt.Image().Pix)
				return cp, header, tt
			}

			refShortImg, header, _ := renderWorkspace("Documents", false)
			actShortImg, _, ttShortAct := renderWorkspace("Documents", true)

			// 1a. Workspace action cluster gap & margin must match bare hovered row (L1 <= 2).
			// (Right(8), gap 15 -> button 2 at [W-24, W-8], gap at [W-39, W-24], button 1 at [W-55, W-39]).
			midY := int(header.Y + header.H/2)
			for x := int(header.X + header.W - 37); x <= int(header.X+header.W-26); x++ {
				gotPx := actShortImg.RGBAAt(x, midY)
				refPx := refShortImg.RGBAAt(x, midY)
				if diff := pixelL1(gotPx, refPx); diff > 2 {
					t.Fatalf("dark=%v workspace action gap painted backdrop patch at x=%d: got=%v ref=%v L1=%d", dark, x, gotPx, refPx, diff)
				}
			}

			// 1b. Short title "Documents" must NOT trigger fade on hover: every pixel
			//     left of the action cluster stays byte-for-byte identical to the
			//     no-actions render (full opacity preserved).
			blankShortImg, _, _ := renderWorkspace(" ", true)
			shortTitleInkPixels := 0
			for y := int(header.Y) + 4; y < int(header.Y+header.H)-4; y++ {
				for x := int(header.X) + 26; x < int(header.X+header.W-60); x++ {
					gotPx := actShortImg.RGBAAt(x, y)
					refPx := refShortImg.RGBAAt(x, y)
					if gotPx != refPx {
						t.Fatalf("dark=%v short title Documents unexpectedly faded/changed at (%d,%d): got=%v ref=%v", dark, x, y, gotPx, refPx)
					}
					if pixelL1(gotPx, blankShortImg.RGBAAt(x, y)) >= 40 {
						shortTitleInkPixels++
					}
				}
			}
			if shortTitleInkPixels < 20 {
				t.Fatalf("dark=%v expected visible short title Documents ink pixels, got %d", dark, shortTitleInkPixels)
			}

			// 2. 40+ char long workspace title over cyan/magenta gradient glass backdrop:
			//    smooth right alpha fade in fade band AND zero foreground ink leak into action area.
			plainLongImg, longHeader, _ := renderWorkspace(longTitle, false)
			fadedLongImg, _, ttLongAct := renderWorkspace(longTitle, true)
			moreLabel := fmt.Sprintf("Actions for workspace “%s”", longTitle)
			wsMoreRect, ok := ttLongAct.Find(moreLabel)
			if !ok {
				t.Fatalf("dark=%v missing workspace more button %q: %v", dark, moreLabel, ttLongAct.Texts())
			}
			// Zero foreground title pixel leakage inside the action button cluster:
			// compare fadedLongImg against blankShortImg (identical hovered row + identical action buttons, no long title).
			for y := int(longHeader.Y) + 2; y < int(longHeader.Y+longHeader.H)-2; y++ {
				for x := int(wsMoreRect.X) - 1; x < int(longHeader.X+longHeader.W)-2; x++ {
					if diff := pixelL1(fadedLongImg.RGBAAt(x, y), blankShortImg.RGBAAt(x, y)); diff > 0 {
						t.Fatalf("dark=%v long workspace title leaked ink into action area at (%d,%d) (actionsX=%.1f): got=%v ref=%v L1=%d",
							dark, x, y, wsMoreRect.X, fadedLongImg.RGBAAt(x, y), blankShortImg.RGBAAt(x, y), diff)
					}
				}
			}
			// Smooth right-edge alpha fade inside the 14 DIP band left of clipRight:
			// find pixels with strong glyph contrast in plainLongImg vs blankShortImg,
			// verify intermediate attenuated ink exists and fades down toward clipRight.
			clipRight := int(wsMoreRect.X) - 4 // text right edge (header.W-14) - (47-2) = header.W-59 = wsMoreRect.X-4
			fadeStart := clipRight - int(sidebarTitleFadeWidth)
			attenuated := 0
			leftHalfSum, leftHalfCnt := 0.0, 0
			rightHalfSum, rightHalfCnt := 0.0, 0
			fadeMid := (fadeStart + clipRight) / 2
			for y := int(longHeader.Y) + 4; y < int(longHeader.Y+longHeader.H)-4; y++ {
				for x := fadeStart; x < clipRight; x++ {
					bgPx := blankShortImg.RGBAAt(x, y)
					fullDelta := pixelL1(plainLongImg.RGBAAt(x, y), bgPx)
					if fullDelta < 60 {
						continue
					}
					curDelta := pixelL1(fadedLongImg.RGBAAt(x, y), bgPx)
					ratio := float64(curDelta) / float64(fullDelta)
					if curDelta > 0 && curDelta < fullDelta-6 {
						attenuated++
					}
					if x < fadeMid {
						leftHalfSum += ratio
						leftHalfCnt++
					} else {
						rightHalfSum += ratio
						rightHalfCnt++
					}
				}
			}
			if attenuated < 8 || leftHalfCnt == 0 || rightHalfCnt == 0 {
				t.Fatalf("dark=%v long workspace title missing smooth right alpha fade in [%d,%d): attenuated=%d leftCnt=%d rightCnt=%d",
					dark, fadeStart, clipRight, attenuated, leftHalfCnt, rightHalfCnt)
			}
			if rightHalfSum/float64(rightHalfCnt) >= leftHalfSum/float64(leftHalfCnt) {
				t.Fatalf("dark=%v long workspace title fade did not decrease toward right edge: leftAvg=%.3f rightAvg=%.3f",
					dark, leftHalfSum/float64(leftHalfCnt), rightHalfSum/float64(rightHalfCnt))
			}

			// 3. Hovered session row over cyan/magenta gradient glass backdrop:
			//    zero backdrop patch in trailing action area (L1 <= 2 vs bare hovered row)
			//    AND 40+ char crawling title smooth right fade with zero ink leak into actions.
			renderSession := func(item session.Item, withActions bool, reduceMotion bool) (*image.RGBA, ui.Rect, *ui.Tester) {
				m := Model{
					ThemeID:    themeID,
					SystemDark: dark,
					OnSelect:   func(string) {},
					Snapshot:   session.ViewState{Items: []session.Item{item}},
				}
				if withActions {
					m.OnArchive = func(string) {}
					m.OnPin = func(string) {}
				}
				tt := ui.NewTester(func(c *ui.Context) {
					defer withSidebarTheme(c, m)()
					tk := tok(c)
					ui.Column(c).Fill().Draw(paintDualGradientBackdrop).Padding(8).Children(func() {
						rowSession(c, item, 0, m, tk.Foreground, tk.MutedForeground, tk.SidebarBorder)
					})
				}, 280, 64)
				if dark {
					tt.SetDark(true)
				}
				tt.SetPreferences(ui.Preferences{ReduceMotion: reduceMotion})
				t0 := time.Unix(20000, 0)
				tt.SetTime(t0)
				tt.Frame()
				label := item.Title
				if item.Blank {
					label = i18n.T(i18n.KeySidebarNewSessionFallback)
				}
				rowRect, ok := tt.Find(label)
				if !ok {
					t.Fatalf("dark=%v missing session row %q: %v", dark, label, tt.Texts())
				}
				tt.Move(rowRect.X+16, rowRect.Y+rowRect.H/2)
				tt.SetTime(t0)
				tt.Frame()
				cp := image.NewRGBA(tt.Image().Bounds())
				copy(cp.Pix, tt.Image().Pix)
				return cp, rowRect, tt
			}

			bareSessImg, sessRect, _ := renderSession(session.Item{SessionID: "blank", Blank: true}, false, true)
			actSessImg, _, ttSessAct := renderSession(session.Item{SessionID: "s1", Title: "Short chat", UpdatedAt: 1700000000000}, true, true)
			sessMoreRect, ok := ttSessAct.Find(sessionActionsLabel("s1", "Short chat"))
			if !ok {
				t.Fatalf("dark=%v missing session action button: %v", dark, ttSessAct.Texts())
			}
			// Sample non-glyph background pixels inside the session action cluster:
			// top/bottom band (y = row.Y+2, buttons are 24px centered in 32px row)
			// and the 2px inter-button gaps at midY.
			sessTopY := int(sessRect.Y) + 2
			for x := int(sessMoreRect.X); x < int(sessRect.X+sessRect.W)-8; x++ {
				gotPx := actSessImg.RGBAAt(x, sessTopY)
				refPx := bareSessImg.RGBAAt(x, sessTopY)
				if diff := pixelL1(gotPx, refPx); diff > 2 {
					t.Fatalf("dark=%v session action band painted backdrop patch at (%d,%d): got=%v ref=%v L1=%d",
						dark, x, sessTopY, gotPx, refPx, diff)
				}
			}
			sessMidY := int(sessRect.Y + sessRect.H/2)
			for _, gapX := range []int{int(sessMoreRect.X) + 25, int(sessMoreRect.X) + 51} {
				gotPx := actSessImg.RGBAAt(gapX, sessMidY)
				refPx := bareSessImg.RGBAAt(gapX, sessMidY)
				if diff := pixelL1(gotPx, refPx); diff > 2 {
					t.Fatalf("dark=%v session action gap painted backdrop patch at (%d,%d): got=%v ref=%v L1=%d",
						dark, gapX, sessMidY, gotPx, refPx, diff)
				}
			}

			// Long session title (44 chars) at hover start (ReduceMotion=false, offset=0 -> right ClipFade=14):
			// smooth alpha attenuation on right edge AND zero ink leak into action cluster.
			longSessImg, longSessRect, ttLongSess := renderSession(session.Item{SessionID: "s-long", Title: longTitle, UpdatedAt: 1700000000000}, true, false)
			longSessMore, ok := ttLongSess.Find(sessionActionsLabel("s-long", longTitle))
			if !ok {
				t.Fatalf("dark=%v missing long session action button: %v", dark, ttLongSess.Texts())
			}
			for y := int(longSessRect.Y) + 2; y < int(longSessRect.Y+longSessRect.H)-2; y++ {
				for x := int(longSessMore.X) - 1; x < int(longSessRect.X+longSessRect.W)-8; x++ {
					if diff := pixelL1(longSessImg.RGBAAt(x, y), actSessImg.RGBAAt(x, y)); diff > 0 {
						t.Fatalf("dark=%v long session title leaked ink into action area at (%d,%d) (actionsX=%.1f): got=%v ref=%v L1=%d",
							dark, x, y, longSessMore.X, longSessImg.RGBAAt(x, y), actSessImg.RGBAAt(x, y), diff)
					}
				}
			}
			// Verify smooth right-edge alpha fade in [sessClipRight-14, sessClipRight)
			// by comparing faded glyph contrast against full-opacity left glyph peak.
			sessClipRight := int(longSessMore.X) - 4
			sessFadeStart := sessClipRight - int(sidebarTitleFadeWidth)
			sessFadeMid := (sessFadeStart + sessClipRight) / 2
			fullInkMax := 0
			for y := int(longSessRect.Y) + 4; y < int(longSessRect.Y+longSessRect.H)-4; y++ {
				for x := int(longSessRect.X) + 32; x < sessFadeStart-4; x++ {
					if d := pixelL1(longSessImg.RGBAAt(x, y), bareSessImg.RGBAAt(x, y)); d > fullInkMax {
						fullInkMax = d
					}
				}
			}
			if fullInkMax < 80 {
				t.Fatalf("dark=%v expected strong session title ink contrast left of fade, got max L1=%d", dark, fullInkMax)
			}
			sessLeftMax, sessRightMax := 0, 0
			sessMidTone := 0
			for y := int(longSessRect.Y) + 4; y < int(longSessRect.Y+longSessRect.H)-4; y++ {
				for x := sessFadeStart; x < sessClipRight; x++ {
					d := pixelL1(longSessImg.RGBAAt(x, y), bareSessImg.RGBAAt(x, y))
					if d > 12 && d < fullInkMax*3/4 {
						sessMidTone++
					}
					if x < sessFadeMid && d > sessLeftMax {
						sessLeftMax = d
					}
					if x >= sessFadeMid && d > sessRightMax {
						sessRightMax = d
					}
				}
			}
			if sessMidTone < 8 || sessRightMax >= sessLeftMax {
				t.Fatalf("dark=%v long session title missing smooth right ClipFade in [%d,%d): midTone=%d leftMax=%d rightMax=%d fullMax=%d",
					dark, sessFadeStart, sessClipRight, sessMidTone, sessLeftMax, sessRightMax, fullInkMax)
			}

			// 4. Verify zero layout shift across all interactive elements (both short and long titles).
			for _, tt := range []*ui.Tester{ttShortAct, ttLongAct} {
				tt.Move(2, 130)
				tt.Frame()
				ui.AssertHoverFocusZeroShift(t, tt)
			}
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
