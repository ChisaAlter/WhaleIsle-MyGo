package chrome_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/sidebar"
	"whaleisle/internal/ui/theme"
)

// These render the dark half with the software renderer as evidence of ink and
// material, not a claim about a live Windows window: DPI, font rasterization
// and the native compositor are not involved. Set DARK_MATERIAL_SNAPSHOT_DIR to
// export them (docs/review-20261020/evidence/after holds the Wave 4 copies).
const (
	snapshotWidth        = 1400
	snapshotHeight       = 900
	snapshotSidebarWidth = 280
)

var darkWallpaperOnce struct {
	sync.Once
	url string
	err error
}

// darkWallpaperDataURL is the dark photo the snapshots stand on: a deep blue
// gradient with a soft bloom, so the wallpaper layer's mask and the glass share
// of it are both visible.
func darkWallpaperDataURL(t *testing.T) string {
	t.Helper()
	darkWallpaperOnce.Do(func() {
		const w, h = 320, 200
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				fx, fy := float64(x)/w, float64(y)/h
				v := 18 + 38*(1-fy) + 14*fx
				bloom := 60 * max(0.0, 1-2.2*((fx-.62)*(fx-.62)+(fy-.3)*(fy-.3)))
				img.SetRGBA(x, y, color.RGBA{
					R: uint8(min(255, v+bloom*.5)),
					G: uint8(min(255, v+bloom*.8)),
					B: uint8(min(255, v+bloom*1.6)),
					A: 255,
				})
			}
		}
		var buf bytes.Buffer
		darkWallpaperOnce.err = png.Encode(&buf, img)
		if darkWallpaperOnce.err == nil {
			darkWallpaperOnce.url = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	})
	if darkWallpaperOnce.err != nil {
		t.Fatal(darkWallpaperOnce.err)
	}
	return darkWallpaperOnce.url
}

// darkRoot paints what the window paints first, on the root itself as the shell
// does: the wallpaper layer with its dim mask, from the same painter. The
// returned column is the frame's content layer.
func darkRoot(c *ui.Context, t *testing.T) ui.Element {
	c.SetTheme(ui.DarkTheme())
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	paint := chrome.WallpaperBackgroundPainter(darkWallpaperDataURL(t), prefs, c.Preferences().ReduceMotion)
	return ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
}

// withWallpaper is darkRoot with the scene built inside it.
func withWallpaper(c *ui.Context, t *testing.T, scene func()) {
	darkRoot(c, t).Children(scene)
}

func darkColors() theme.Colors { return theme.Resolve(theme.IDDeepseekDark).Active(true) }

// snapshotRows is the sidebar content: one workspace with sessions.
func snapshotRows() session.ViewState {
	return session.ViewState{
		WorkspaceReady: true,
		Workspaces: []session.WorkspaceRow{{
			WorkspaceID: "w1",
			Path:        "C:/work/whale-isle",
			Title:       "Whale Isle",
			SessionIDs:  []string{"s1", "s2"},
		}},
		Items: []session.Item{
			{SessionID: "s1", Title: "Rewrite the material layer", Cwd: "C:/work/whale-isle", UpdatedAt: 1},
			{SessionID: "s2", Title: "Dark glass contrast pass", Cwd: "C:/work/whale-isle"},
		},
	}
}

// saveSnapshot writes one rendered frame as evidence.
func saveSnapshot(t *testing.T, dir, name string, image image.Image) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// finishSnapshot lets every overlay finish its entrance, so the frame shows the
// settled material rather than the first frame of the fade.
func finishSnapshot(tt *ui.Tester) {
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
}

// snapshotDir skips the export unless the caller asked for it.
func snapshotDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("DARK_MATERIAL_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set DARK_MATERIAL_SNAPSHOT_DIR to export the dark material snapshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestDarkMaterialSnapshots renders the three dark scenes the Wave 4 review
// asks for: the hero with the composer card, the settings modal, and the
// workspace menu.
func TestDarkMaterialSnapshots(t *testing.T) {
	dir := snapshotDir(t)
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	t.Run("hero-composer", func(t *testing.T) {
		draft := ""
		modelPick := "DeepSeek-V3.2-Exp"
		client := composer.NewDraftClient(&draft)
		tt := ui.NewTester(func(c *ui.Context) {
			withWallpaper(c, t, func() {
				ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
					ui.Column(c).Width(snapshotSidebarWidth).Shrink(0).FillHeight().Children(func() {
						sidebar.View(c, sidebar.Model{ThemeID: theme.IDDeepseekDark, SystemDark: true, Snapshot: snapshotRows()})
					})
					ui.Column(c).Grow(1).MinHeight(0).Draw(chrome.Glass(chrome.Color(darkColors().BG), chrome.MaterialCanvas)).Children(func() {
						heroStack(c, client, &draft, &modelPick)
					})
				})
			})
		}, snapshotWidth, snapshotHeight)
		finishSnapshot(tt)
		saveSnapshot(t, dir, "dark-hero-composer", tt.Image())
	})

	t.Run("settings-modal", func(t *testing.T) {
		tt := ui.NewTester(func(c *ui.Context) {
			darkRoot(c, t)
			settings.ModalView(c, settings.Model{
				ThemeID:       theme.IDDeepseekDark,
				SystemDark:    true,
				Desktop:       &settings.DesktopAccess{},
				DesktopActive: true,
				// The body is the shell's own composition: a profile page comes
				// from the store, which a fixture does not have, so the snapshot
				// renders the contrast levels directly.
				ActiveSection:  settings.SectionGeneral,
				SectionContent: snapshotSettingsBody,
				OnClose:        func() {},
			})
		}, snapshotWidth, snapshotHeight)
		finishSnapshot(tt)
		saveSnapshot(t, dir, "dark-settings-modal", tt.Image())
	})

	t.Run("workspace-menu", func(t *testing.T) {
		open := true
		tt := ui.NewTester(func(c *ui.Context) {
			withWallpaper(c, t, func() {
				ui.Column(c).Width(snapshotSidebarWidth).Shrink(0).FillHeight().Children(func() {
					sidebar.View(c, sidebar.Model{
						ThemeID:           theme.IDDeepseekDark,
						SystemDark:        true,
						Snapshot:          snapshotRows(),
						WorkspaceMenuOpen: &open,
						OnBrowseWorkspace: func() {},
						OnRevealWorkspace: func() {},
						OnRenameWorkspace: func(string) {},
						OnDeleteWorkspace: func() {},
						OnRefresh:         func() {},
					})
				})
			})
		}, snapshotWidth, snapshotHeight)
		finishSnapshot(tt)
		saveSnapshot(t, dir, "dark-workspace-menu", tt.Image())
	})
}

// snapshotSettingsBody shows the three ink levels the material guard protects on
// the modal's glass.
func snapshotSettingsBody(c *ui.Context) {
	colors := darkColors()
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	caption := chrome.Color(colors.Caption)
	ui.Column(c).FillWidth().Gap(10).Padding(0, 4).Children(func() {
		ui.Text(c, "Appearance follows the desktop.").FontSize(14).TextColor(fg)
		ui.Text(c, "Glass opacity, wallpaper and pointer effects live here.").FontSize(13).TextColor(muted)
		ui.Text(c, "Changed a moment ago").FontSize(11).TextColor(caption)
		ui.Box(c).FillWidth().Height(0.5).Background(chrome.Color(colors.Border))
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Reduce motion").FontSize(14).TextColor(fg).Grow(1)
			ui.Box(c).Width(36).Height(20).Radius(10).Background(chrome.Color(colors.Accent))
		})
		ui.Text(c, "Whale Isle asks the desktop before it animates.").FontSize(12).TextColor(muted)
	})
}

// heroStack is ConversationRoot's empty state: the fish mark, the headline, the
// workspace chip and the composer card.
func heroStack(c *ui.Context, client *composer.DraftClient, draft, modelPick *string) {
	colors := darkColors()
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(18).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(12).Children(func() {
			if bmp := chrome.FishLogo(); bmp != nil {
				ui.Image(c, bmp).Width(68).Height(50).Fit(ui.Contain)
			}
			ui.Text(c, "New session").FontSize(26).FixedLineHeight(32).FontWeight(500).TextColor(fg)
			ui.Row(c).Height(28).Padding(0, 10).Radius(14).AlignItems(ui.Center).Gap(6).
				Border(0.5, chrome.Color(colors.Border)).Children(func() {
				ui.Text(c, "Whale Isle").FontSize(13).TextColor(muted)
				ui.Text(c, "▾").FontSize(10).TextColor(muted)
			})
		})
		ui.Column(c).Width(760).Shrink(0).Children(func() {
			composer.View(c, composer.Model{
				Draft:              draft,
				TextInput:          client,
				UseTextInputClient: true,
				Hero:               true,
				ThemeID:            theme.IDDeepseekDark,
				SystemDark:         true,
				WorkspaceRoot:      "C:/work/whale-isle",
				ModelPick:          modelPick,
				Snapshot:           session.ViewState{SelectedModel: *modelPick},
			})
		})
	})
}

// TestDarkSnapshotMenuIsVisible is the check behind the workspace-menu snapshot:
// with the menu forced open, its panel and rows are on screen in the dark half.
func TestDarkSnapshotMenuIsVisible(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		withWallpaper(c, t, func() {
			ui.Column(c).Width(snapshotSidebarWidth).Shrink(0).Children(func() {
				sidebar.View(c, sidebar.Model{
					ThemeID: theme.IDDeepseekDark, SystemDark: true, Snapshot: snapshotRows(),
					WorkspaceMenuOpen: &open, OnBrowseWorkspace: func() {}, OnRevealWorkspace: func() {},
					OnRenameWorkspace: func(string) {}, OnDeleteWorkspace: func() {}, OnRefresh: func() {},
				})
			})
		})
	}, snapshotWidth, snapshotHeight)
	finishSnapshot(tt)
	if !tt.HasText(i18n.T(i18n.KeySidebarBrowseWorkspace)) {
		t.Fatalf("workspace menu did not open; texts = %q", tt.Texts())
	}
}
