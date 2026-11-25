package usage

import (
	"testing"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestUsageStandaloneReservesCaptionSafeBandAcrossWidths(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	store := usagestore.New()
	for _, tc := range []struct {
		name  string
		width int
		bar   ui.TitleBar
		minY  float32
	}{
		{name: "narrow-windows", width: 640, bar: ui.TitleBar{Height: 12, Right: 104}, minY: 44},
		{name: "standard-windows", width: 1024, bar: ui.TitleBar{Height: 12, Right: 104}, minY: 44},
		{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}, minY: 44},
		{name: "macos-traffic-lights", width: 800, bar: ui.TitleBar{Height: 28, Left: 72}, minY: 44},
		{name: "fullscreen", width: 1280, bar: ui.TitleBar{}, minY: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{
					Store:         store,
					ThemeID:       "deepseek",
					OnExportStats: func(string) {},
					OnClose:       func() {},
				})
			}, tc.width, 720)
			tt.SetTitleBar(tc.bar)
			for _, label := range []string{
				i18n.T(settingsPass4UsageRefresh),
				i18n.T(i18n.KeyUsageBack),
			} {
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %q", label)
				}
				if r.Y < tc.minY {
					t.Fatalf("%s intruded into top caption band: y=%.1f < %.1f", label, r.Y, tc.minY)
				}
			}
		})
	}
}
