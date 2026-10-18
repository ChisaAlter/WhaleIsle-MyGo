package chrome

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestTitleBarLabels renders the custom title bar headlessly. Window minimize,
// maximize and close stay native (TitleBarHidden); the strip exposes the app
// name and optional git chrome.
func TestTitleBarLabels(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	appName := i18n.T(i18n.KeyAppName)
	colors := DefaultTitleBarColors()

	t.Run("app name", func(t *testing.T) {
		tt := ui.NewTester(func(c *ui.Context) {
			TitleBar(c, appName, colors)
		}, 800, TitleBarHeight)

		if !tt.HasText(appName) {
			t.Fatalf("missing app name %q; texts = %q", appName, tt.Texts())
		}
	})

	t.Run("git branch", func(t *testing.T) {
		tt := ui.NewTester(func(c *ui.Context) {
			TitleBarWithActions(c, appName, colors, &GitBadge{
				Branch:      "main",
				Dirty:       true,
				FetchLabel:  i18n.T(i18n.KeyGitFetch),
				PullLabel:   i18n.T(i18n.KeyGitPull),
				PushLabel:   i18n.T(i18n.KeyGitPush),
				CommitLabel: i18n.T(i18n.KeyGitCommit),
				OnFetch:     func() {},
				OnPull:      func() {},
				OnPush:      func() {},
				OnCommit:    func() {},
			}, nil)
		}, 800, TitleBarHeight)

		if !tt.HasText(appName) {
			t.Fatalf("missing app name %q; texts = %q", appName, tt.Texts())
		}
		if !tt.HasText("main •") {
			t.Fatalf("missing branch label; texts = %q", tt.Texts())
		}
		if !tt.HasText(i18n.T(i18n.KeyGitFetch)) {
			t.Fatalf("missing fetch button; texts = %q", tt.Texts())
		}
	})
}
