package sidebar

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The reference is an isolated text run, not another sidebar. Its typography is
// the installed WorkspaceBrowser.sectionLabel / Rows.tasksSection contract:
// inherited 14px (13px in the two skins), and 12px/500 for the archive heading.
func TestSidebarSectionTypographyPaintMatchesInstalledRoles(t *testing.T) {
	previousPreferences, previousLocale := theme.CurrentPreferences(), i18n.Locale()
	t.Cleanup(func() {
		theme.SetPreferences(previousPreferences)
		i18n.SetLocale(previousLocale)
	})
	i18n.SetLocale(i18n.LocaleEN)
	for _, tc := range []struct {
		family string
		dark   bool
	}{
		{"deepseek", false},
		{"deepseek", true},
		{"codex", false},
		{"claude", false},
	} {
		t.Run(fmt.Sprintf("%s/dark%v", tc.family, tc.dark), func(t *testing.T) {
			preferences := theme.Defaults()
			preferences.Interface.Size = 16
			preferences.Light, preferences.Dark = tc.family, tc.family
			theme.SetPreferences(preferences)
			model := Model{ThemeID: theme.IDSystem, SystemDark: tc.dark, Snapshot: session.ViewState{WorkspaceReady: true}}
			actual := ui.NewTester(func(c *ui.Context) {
				theme.ApplyInterface(c)
				View(c, model)
			}, 280, 700)
			actual.SetDark(tc.dark)
			actual.SetPreferences(ui.Preferences{TextScale: 1, ReduceMotion: true})
			painted := actual.Image()
			colors := theme.Resolve(theme.IDSystem).Active(tc.dark)
			workspaceSize := float32(14)
			if tc.family != "deepseek" {
				workspaceSize = 13
			}
			for _, text := range []struct {
				label        string
				size, height float32
				weight       int
			}{
				{i18n.T(i18n.KeySidebarWorkspaceSection), workspaceSize, 20, 400},
				{"Archive section title", 12, 18, 500},
			} {
				bounds, ok := actual.Find(text.label)
				if !ok {
					t.Fatalf("missing section text %q: %q", text.label, actual.Texts())
				}
				referenceLabel := text.label
				if referenceLabel == "Archive section title" {
					referenceLabel = i18n.T(i18n.KeySidebarArchivedSection)
				}
				reference := ui.NewTester(func(c *ui.Context) {
					theme.ApplyInterface(c)
					ui.Column(c).Fill().Background(chrome.Color(colors.Sidebar)).Children(func() {
						ui.Text(c, referenceLabel).Absolute().Left(bounds.X).Top(bounds.Y).Width(bounds.W).Height(bounds.H).
							FontSize(text.size).FontWeight(text.weight).FixedLineHeight(text.height).
							TextColor(chrome.Color(colors.Muted)).SingleLine()
					})
				}, 280, 700)
				reference.SetDark(tc.dark)
				reference.SetPreferences(ui.Preferences{TextScale: 1, ReduceMotion: true})
				want := reference.Image()
				different := 0
				ink := 0
				background := want.RGBAAt(279, 699)
				for y := int(math.Floor(float64(bounds.Y))); y < int(math.Ceil(float64(bounds.Y+bounds.H))); y++ {
					for x := int(math.Floor(float64(bounds.X))); x < int(math.Ceil(float64(bounds.X+bounds.W))); x++ {
						if painted.RGBAAt(x, y) != want.RGBAAt(x, y) {
							different++
						}
						if want.RGBAAt(x, y) != background {
							ink++
						}
					}
				}
				if ink == 0 || different != 0 {
					t.Fatalf("%q painted typography differs from installed %gpx/%gpx weight%d tertiary: differing=%d referenceInk=%d bounds=%+v", text.label, text.size, text.height, text.weight, different, ink, bounds)
				}
			}
		})
	}
}
