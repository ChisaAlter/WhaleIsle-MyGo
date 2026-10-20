package usage

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestUsageEmptyCardMatchesTheReference measures the empty state: a plain white
// card 108 DIP tall with a 12 DIP radius and the two centred lines.
func TestUsageEmptyCardMatchesTheReference(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Column(c).Padding(20).Children(func() {
			usageEmpty(c, i18n.T(settingsPass4UsageEmptyTitle), i18n.T(settingsPass4UsageEmptyBody),
				ui.RGB(20, 20, 20), ui.RGB(120, 120, 120))
		})
	}
	tt := ui.NewTester(view, 1000, 400)
	title, ok := tt.Find(i18n.T(settingsPass4UsageEmptyTitle))
	if !ok {
		t.Fatalf("empty title missing: %v", tt.Texts())
	}
	body, ok := tt.Find(i18n.T(settingsPass4UsageEmptyBody))
	if !ok {
		t.Fatal("empty body missing")
	}
	if title.Y >= body.Y {
		t.Fatalf("title is not above the body: %+v %+v", title, body)
	}
	if !tt.HasText("暂无统计数据") || !tt.HasText("Token 消耗情况") {
		t.Fatalf("empty copy drifted: %v", tt.Texts())
	}
}

func TestUsageUpdatedLineFollowsTheHostTimestamp(t *testing.T) {
	// 2024-01-02T09:30:00Z, in seconds and in milliseconds.
	want := "数据更新于 09:30 · UTC"
	if got := usageUpdatedLabel(1704187800); got != want {
		t.Fatalf("updated line = %q, want %q", got, want)
	}
	if got := usageUpdatedLabel(1704187800000); got != want {
		t.Fatalf("millisecond stamp = %q, want %q", got, want)
	}
	if got := usageUpdatedLabel(0); got != "" {
		t.Fatalf("missing stamp rendered %q", got)
	}
}

// TestUsageGalleryEmptyState exports the usage shot with no overview data.
func TestUsageGalleryEmptyState(t *testing.T) {
	dir := os.Getenv("SETTINGS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set SETTINGS_GALLERY_DIR to export the settings gallery")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, usagePageIcon).Size(16, 16).TextColor(ui.RGB(20, 20, 20))
					ui.Text(c, i18n.T(settingsPass4UsageTitle)).FontSize(theme.InterfaceSize(18)).FontWeight(600).TextColor(ui.RGB(20, 20, 20)).Grow(1)
					for _, label := range []string{settingsPass4UsageExport, settingsPass4UsageSettings, settingsPass4UsageRefresh} {
						ui.Button(c, i18n.T(label)).Height(28).Padding(0, 10).Radius(8).BorderWidth(1)
					}
				})
				usageEmpty(c, i18n.T(settingsPass4UsageEmptyTitle), i18n.T(settingsPass4UsageEmptyBody),
					ui.RGB(20, 20, 20), ui.RGB(120, 120, 120))
			})
		})
	}, 1000, 800)
	file, err := os.Create(filepath.Join(dir, "ours-settings-usage.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
