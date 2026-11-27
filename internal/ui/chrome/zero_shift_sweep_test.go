package chrome

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func TestChromeWorkspaceTrailingHoverFocusZeroShiftSweep(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	for _, density := range []string{"full", "cozy", "compact"} {
		t.Run("repo/"+density, func(t *testing.T) {
			m := WorkspaceTrailingModel{
				Status:            &git.RepositoryStatus{IsRepo: true, Branch: "fix/ui-systemic-bugs", Dirty: true, HasOrigin: true},
				WorkspaceRoot:     "fixture",
				GitLoaded:         true,
				Density:           density,
				OnGitAction:       func(string) {},
				RenderBranches:    func(*ui.Context, ui.Element, *bool) {},
				ShowTerminal:      true,
				TerminalAvailable: true,
				ShowSurfaces:      true,
				OnToggleTerminal:  func() {},
				OnToggleSurfaces:  func() {},
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Fill().Padding(16).AlignItems(ui.Center).Children(func() {
					WorkspaceTrailing(c, m)
				})
			}, 800, 120)
			tt.Frame()
			tt.Frame()
			ui.AssertHoverFocusZeroShift(t, tt)
		})
	}

	t.Run("non-repo", func(t *testing.T) {
		m := WorkspaceTrailingModel{
			Status:            &git.RepositoryStatus{IsRepo: false},
			WorkspaceRoot:     "fixture",
			GitLoaded:         true,
			Density:           "full",
			OnGitAction:       func(string) {},
			ShowTerminal:      true,
			TerminalAvailable: true,
			ShowSurfaces:      true,
			OnToggleTerminal:  func() {},
			OnToggleSurfaces:  func() {},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Row(c).Fill().Padding(16).AlignItems(ui.Center).Children(func() {
				WorkspaceTrailing(c, m)
			})
		}, 800, 120)
		tt.Frame()
		tt.Frame()
		ui.AssertHoverFocusZeroShift(t, tt)
	})
}
