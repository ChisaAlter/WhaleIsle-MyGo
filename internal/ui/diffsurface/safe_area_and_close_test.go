package diffsurface

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func TestGitProgressReservesTitleBarSafeArea(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	progress := git.NewProgress("git push", "C:/repo", nil)
	progress.Finish("Done", false)

	for _, tc := range []struct {
		name  string
		width int
		bar   ui.TitleBar
	}{
		{name: "narrow-windows", width: 640, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}},
		{name: "macos-traffic-lights", width: 800, bar: ui.TitleBar{Height: 28, Left: 72}},
		{name: "fullscreen", width: 1280, bar: ui.TitleBar{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DrawGitProgress(c, progress, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120), func() {})
				})
			}, tc.width, 200)
			tt.SetTitleBar(tc.bar)
			r, ok := tt.Find("关闭进度")
			if !ok {
				t.Fatal("missing dismiss button")
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("dismiss button overlaps right caption safe area: right=%.1f > %.1f", r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})
	}
}
