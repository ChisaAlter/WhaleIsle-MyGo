package browserpreview

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"
)

func TestBrowserChromeAndPanelReserveTitleBarSafeArea(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	cases := []struct {
		name  string
		width int
		bar   ui.TitleBar
	}{
		{name: "narrow-windows", width: 640, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "standard-windows", width: 1024, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}},
		{name: "macos-traffic-lights", width: 800, bar: ui.TitleBar{Height: 28, Left: 72}},
		{name: "fullscreen", width: 1280, bar: ui.TitleBar{}},
	}

	for _, tc := range cases {
		t.Run("controller/"+tc.name, func(t *testing.T) {
			ctrl := NewController(nil)
			ctrl.NewTab("https://example.com/a-very-long-page-title-for-narrow-window-check")
			tt := ui.NewTester(func(c *ui.Context) {
				ctrl.View(c, "deepseek", false, func() {})
			}, tc.width, 500)
			tt.SetTitleBar(tc.bar)

			maxRight := float32(tc.width) - tc.bar.Right
			for _, label := range []string{"新标签", "关闭标签"} {
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %q", label)
				}
				if r.X+r.W > maxRight+0.5 {
					t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", label, r.X+r.W, maxRight)
				}
			}
			backRect, ok := tt.Find("返回工作区")
			if !ok {
				t.Fatal("missing 返回工作区")
			}
			if backRect.X < tc.bar.Left-0.5 {
				t.Fatalf("返回工作区 overlaps left caption safe area: x=%.1f < %.1f", backRect.X, tc.bar.Left)
			}
		})

		t.Run("panel/"+tc.name, func(t *testing.T) {
			draft := "https://example.com"
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{
					ThemeID:  "deepseek",
					URLDraft: &draft,
					OnClose:  func() {},
				})
			}, tc.width, 500)
			tt.SetTitleBar(tc.bar)

			backLabel := i18n.T(i18n.KeyBrowserPreviewBack)
			r, ok := tt.Find(backLabel)
			if !ok {
				t.Fatalf("missing %q", backLabel)
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", backLabel, r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})
	}
}

func TestPanelSanitizesRawHostAndNavigateErrors(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	draft := "https://example.com"
	rawNav := "dial tcp 127.0.0.1:9999: connectex: No connection could be made because the target machine actively refused it."
	rawHost := "WebView2 E_FAIL 0x80004005 internal raw error"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:       "deepseek",
			URLDraft:      &draft,
			NavigateError: rawNav,
			HostError:     rawHost,
		})
	}, 720, 480)

	if tt.HasText(rawNav) || tt.HasText(rawHost) {
		t.Fatalf("raw error text leaked into browser panel: %v", tt.Texts())
	}
	if !tt.HasText(errcopy.SentenceText(rawNav)) {
		t.Fatalf("missing errcopy sentence for navigate error: %v", tt.Texts())
	}
	if !tt.HasText(errcopy.SentenceText(rawHost)) {
		t.Fatalf("missing errcopy sentence for host error: %v", tt.Texts())
	}
}
