package remote

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestRemoteHeaderReservesTitleBarSafeArea(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{ThemeID: "deepseek", OnClose: func() {}})
			}, tc.width, 600)
			tt.SetTitleBar(tc.bar)
			backLabel := i18n.T(i18n.KeyRemoteBack)
			r, ok := tt.Find(backLabel)
			if !ok {
				t.Fatalf("missing %q", backLabel)
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", backLabel, r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
			titleLabel := i18n.T(i18n.KeyRemoteTitle)
			tr, ok := tt.Find(titleLabel)
			if !ok {
				t.Fatalf("missing %q", titleLabel)
			}
			if tr.X < tc.bar.Left-0.5 {
				t.Fatalf("%s overlaps left caption safe area: x=%.1f < %.1f", titleLabel, tr.X, tc.bar.Left)
			}
		})
	}
}
