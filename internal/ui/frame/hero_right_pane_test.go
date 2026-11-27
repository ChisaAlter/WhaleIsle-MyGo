package frame

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestHeroHeadlineFitsAndCentersWhenRightPaneOpens(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		for _, viewport := range []int{840, 1024, 1200, 1280, 1440} {
			for _, reduced := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/w%d/reduced%v", locale, viewport, reduced), func(t *testing.T) {
					i18n.SetLocale(locale)
					store := session.New()
					sizes := DefaultLayout()
					sizes.Bottom = 0
					rightOpen := false
					rightTab := "files"
					draft := ""
					tt := ui.NewTester(func(c *ui.Context) {
						m := Model{
							Store:              store,
							Sizes:              &sizes,
							RightPaneOpen:      rightOpen,
							RightPaneTab:       &rightTab,
							Draft:              &draft,
							ThemeID:            theme.IDDeepseekDark,
							SystemDark:         true,
							ExternalNavigation: true,
						}
						NavigationShell(c, m, func() {
							View(c, m)
						})
					}, viewport, 800)
					tt.SetPreferences(ui.Preferences{ReduceMotion: reduced, TextScale: 1})
					now := time.Unix(1000, 0)
					tt.SetTime(now)
					tt.Frame()

					checkHero := func(stage string) {
						t.Helper()
						body, ok := tt.Find("Conversation body")
						if !ok {
							t.Fatalf("%s: missing Conversation body; texts=%q", stage, tt.Texts())
						}
						slogan, ok := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
						if !ok {
							t.Fatalf("%s: missing slogan; texts=%q", stage, tt.Texts())
						}
						badge, ok := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
						if !ok {
							t.Fatalf("%s: missing badge; texts=%q", stage, tt.Texts())
						}
						fish, ok := tt.Find("Conversation hero fish")
						if !ok {
							t.Fatalf("%s: missing fish; texts=%q", stage, tt.Texts())
						}
						head, ok := tt.Find("Conversation hero headline")
						if !ok {
							t.Fatalf("%s: missing headline row; texts=%q", stage, tt.Texts())
						}
						colLeft, colRight := body.X, body.X+body.W
						for name, r := range map[string]ui.Rect{
							"headline": head,
							"fish":     fish,
							"slogan":   slogan,
							"badge":    badge,
						} {
							if r.W <= 0 || r.H <= 0 {
								t.Errorf("%s: %s collapsed to empty rect %+v inside [%.1f..%.1f]", stage, name, r, colLeft, colRight)
							}
							if r.X < colLeft-0.5 || r.X+r.W > colRight+0.5 {
								t.Errorf("%s: %s=%+v clipped outside center column [%.1f..%.1f] (sizes=%+v)", stage, name, r, colLeft, colRight, sizes)
							}
						}
						groupLeft := min(fish.X, min(slogan.X, badge.X))
						groupRight := max(fish.X+fish.W, max(slogan.X+slogan.W, badge.X+badge.W))
						groupCenter := (groupLeft + groupRight) / 2
						colCenter := (colLeft + colRight) / 2
						if diff := groupCenter - colCenter; diff < -12 || diff > 12 {
							t.Errorf("%s: hero group [%.1f..%.1f] center=%.1f drifted from column [%.1f..%.1f] center=%.1f (head=%+v fish=%+v slogan=%+v badge=%+v)",
								stage, groupLeft, groupRight, groupCenter, colLeft, colRight, colCenter, head, fish, slogan, badge)
						}
					}

					// Open right pane and verify mid-transition and settled frames.
					rightOpen = true
					tt.SetTime(now.Add(16 * time.Millisecond))
					tt.Frame()
					tt.SetTime(now.Add(150 * time.Millisecond))
					tt.Frame()
					checkHero("open-mid")
					tt.SetTime(now.Add(500 * time.Millisecond))
					tt.Frame()
					checkHero("open-settled")

					// Close right pane and verify reverse transition and settled frames.
					rightOpen = false
					tt.SetTime(now.Add(516 * time.Millisecond))
					tt.Frame()
					tt.SetTime(now.Add(700 * time.Millisecond))
					tt.Frame()
					checkHero("close-mid")
					tt.SetTime(now.Add(1100 * time.Millisecond))
					tt.Frame()
					checkHero("close-settled")
				})
			}
		}
	}
}

func TestHeroAndSkeletonMemoVersionsTrackRevisionsAndBackdrop(t *testing.T) {
	prevPrefs := theme.CurrentPreferences()
	t.Cleanup(func() {
		theme.SetPreferences(prevPrefs)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	m := Model{ThemeID: theme.IDDeepseekLight}

	h0 := heroHeadlineMemoVersion(m, 600)
	s0 := settlingSkeletonMemoVersion(m, 600)

	// Width change invalidates both memos.
	if heroHeadlineMemoVersion(m, 400) == h0 || settlingSkeletonMemoVersion(m, 400) == s0 {
		t.Fatal("width change did not invalidate hero/skeleton memo versions")
	}

	// theme.Revision() change invalidates both memos.
	nextPrefs := prevPrefs
	nextPrefs.Interface.Size = prevPrefs.Interface.Size + 1
	theme.SetPreferences(nextPrefs)
	hTheme := heroHeadlineMemoVersion(m, 600)
	sTheme := settlingSkeletonMemoVersion(m, 600)
	if hTheme == h0 || sTheme == s0 {
		t.Fatal("theme.Revision() change did not invalidate hero/skeleton memo versions")
	}

	// design.Revision() change invalidates both memos.
	design.SetGlass(true, nil)
	hDesign := heroHeadlineMemoVersion(m, 600)
	sDesign := settlingSkeletonMemoVersion(m, 600)
	if hDesign == hTheme || sDesign == sTheme {
		t.Fatal("design.Revision() change did not invalidate hero/skeleton memo versions")
	}

	// chrome.BackdropActive() change invalidates both memos.
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	hBack := heroHeadlineMemoVersion(m, 600)
	sBack := settlingSkeletonMemoVersion(m, 600)
	if hBack == hDesign || sBack == sDesign {
		t.Fatal("chrome.BackdropActive() change did not invalidate hero/skeleton memo versions")
	}
}

func solidPNGDataURL(t *testing.T, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestNavigationShellSingleLayerLightBackdropAndClaudeSkin(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	blackWall := solidPNGDataURL(t, color.RGBA{R: 0, G: 0, B: 0, A: 255})

	for _, tc := range []struct {
		name            string
		themeID         string
		hideSidebarMask bool
		wantCanvasMaxR  uint8
		wantSidebarMaxR uint8
	}{
		{"deepseek-light/hidden-mask", theme.IDDeepseekLight, true, 68, 68},
		{"deepseek-light/visible-mask", theme.IDDeepseekLight, false, 68, 148},
		{"claude-light/hidden-mask", "claude.light", true, 68, 68},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefs := wallpapers.Defaults()
			prefs.Glass = 80
			prefs.Effect = "none"
			prefs.HideSidebarMask = tc.hideSidebarMask
			chrome.ConfigureBackdrop(blackWall, prefs)

			store := session.New()
			sizes := DefaultLayout()
			sizes.Bottom = 0
			draft := ""
			tt := ui.NewTester(func(c *ui.Context) {
				paint := chrome.WallpaperBackgroundPainter(blackWall, prefs, true)
				ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) }).Children(func() {
					m := Model{
						Store:              store,
						Sizes:              &sizes,
						Draft:              &draft,
						ThemeID:            tc.themeID,
						SystemDark:         false,
						ExternalNavigation: true,
					}
					NavigationShell(c, m, func() { View(c, m) })
				})
			}, 1100, 720)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			tt.Frame()

			img := tt.Image()
			// Center conversation canvas upper background (away from centered hero/composer).
			canvasPx := img.RGBAAt(650, 120)
			if canvasPx.R > tc.wantCanvasMaxR {
				t.Fatalf("%s center canvas pixel=%+v exceeded single-layer cap %d (double-stacked or opaque fill)", tc.name, canvasPx, tc.wantCanvasMaxR)
			}
			// Sidebar empty lower area.
			sidebarPx := img.RGBAAt(140, 580)
			if sidebarPx.R > tc.wantSidebarMaxR {
				t.Fatalf("%s sidebar pixel=%+v exceeded single-layer cap %d (double-stacked white veil)", tc.name, sidebarPx, tc.wantSidebarMaxR)
			}
		})
	}
}

func TestWorkspaceCaptionHairlineFollowsContextDarkAndGlobalPageHidesInitGit(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })
	i18n.SetLocale(i18n.LocaleEN)

	// 1. Hairline follows c.Theme().Dark even when m.SystemDark is false.
	store := session.New()
	snap := session.ViewState{
		SelectedID: "s1",
		Items:      []session.Item{{SessionID: "s1", Title: "Chat One"}},
		Messages:   []session.Message{{ID: "m1", Role: "user"}},
	}
	sizes := DefaultLayout()
	darkColors := theme.Resolve(theme.IDDeepseekDark).Active(true)
	ttHair := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Box(c).Fill().Background(ui.RGB(20, 20, 22)).Children(func() {
			m := Model{
				Store:            store,
				Sizes:            &sizes,
				ThemeID:          theme.IDDeepseekDark,
				SystemDark:       false, // intentionally opposite of c.Theme().Dark
				HideChatViewTabs: true,
			}
			workspaceSharedCaption(c, m, snap, darkColors, false)
		})
	}, 800, 80)
	ttHair.SetDark(true)
	ttHair.Frame()
	// Bottom edge pixel of the 48px caption band: dark mode draws white@2% over rgb(20,20,22) (~24),
	// whereas light tCaption.Border would be much brighter (~220) or higher delta.
	linePx := ttHair.Image().RGBAAt(400, int(workspaceCaptionHeight)-1)
	if linePx.R > 45 {
		t.Fatalf("dark caption hairline used light stroke (R=%d > 45): %+v", linePx.R, linePx)
	}

	// 2. Global pages suppress non-repo "Initialize Git", while conversation keeps it.
	lightColors := theme.Resolve(theme.IDDeepseekLight).Active(false)
	for _, tc := range []struct {
		name          string
		workspacePage bool
		activePanel   string
		wantInitGit   bool
	}{
		{"conversation", false, "", true},
		{"plugins-global", true, "plugins", false},
		{"schedule-global", true, "schedule", false},
		{"assistant-global", false, "assistant", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{
					Store:         store,
					Sizes:         &sizes,
					ThemeID:       theme.IDDeepseekLight,
					WorkspacePage: tc.workspacePage,
					ActivePanel:   tc.activePanel,
					WorkspaceTrailing: func(c *ui.Context, density string, globalPage bool) {
						chrome.WorkspaceTrailing(c, chrome.WorkspaceTrailingModel{
							Status:        &git.RepositoryStatus{IsRepo: false},
							GitLoaded:     true,
							GlobalPage:    globalPage,
							WorkspaceRoot: "fixture",
							Density:       density,
							OnGitAction:   func(string) {},
						})
					},
				}
				workspaceSharedCaption(c, m, snap, lightColors, false)
			}, 900, 80)
			tt.Frame()
			got := tt.HasText("Initialize Git")
			if got != tc.wantInitGit {
				t.Fatalf("%s: HasText(Initialize Git)=%v want=%v texts=%q", tc.name, got, tc.wantInitGit, tt.Texts())
			}
		})
	}
}

func TestWorkspaceDockTabsZeroHoverShift(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })
	i18n.SetLocale(i18n.LocaleEN)

	tabActive := &WorkspacePane{ID: "tab-1", Kind: "agents"}
	tabInactive := &WorkspacePane{ID: "tab-2", Kind: "agents"}
	tabThird := &WorkspacePane{ID: "tab-3", Kind: "host-terminal"}
	setDockTabs(tabActive, []*WorkspacePane{tabActive, tabInactive, tabThird})
	panes := &WorkspacePanes{Root: tabActive}
	sizes := DefaultLayout()
	sizes.Workspace = panes

	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{
			Sizes:         &sizes,
			workspacePane: tabActive,
			ThemeID:       theme.IDDeepseekLight,
		}
		workspaceDockTabStrip(c, m)
	}, 600, 60)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()

	// Find first and third tab chips at rest via their titles.
	r1Rest, ok1 := tt.Find(dockTabTitle(tabActive))
	r3Rest, ok3 := tt.Find(dockTabTitle(tabThird))
	if !ok1 || !ok3 {
		t.Fatalf("missing dock tabs: texts=%q", tt.Texts())
	}
	// Hover over the second (inactive closeable) tab seat right after the first tab.
	inactiveX := r1Rest.X + r1Rest.W + 20
	inactiveY := r1Rest.Y + r1Rest.H/2
	tt.Move(inactiveX, inactiveY)
	tt.Frame()

	r1Hover, _ := tt.Find(dockTabTitle(tabActive))
	r3Hover, _ := tt.Find(dockTabTitle(tabThird))
	if r1Hover != r1Rest || r3Hover != r3Rest {
		t.Fatalf("hovering inactive closeable dock tab shifted siblings: tab1 %+v->%+v tab3 %+v->%+v", r1Rest, r1Hover, r3Rest, r3Hover)
	}
}
