package usage

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestUsageHeaderExportsAndHasNoSettingsButton covers the Pass 7 audit: the
// header exports JSON through OnExportStats, and the range "settings" button the
// shell never implemented is gone instead of sitting there disabled.
func TestUsageHeaderExportsAndHasNoSettingsButton(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	formats := []string{}
	s := usageFixture(t, "empty")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(format string) { formats = append(formats, format) }})
		})
	}, 680, 1000)
	if err := tt.Click("Export"); err != nil {
		t.Fatalf("export action is missing: %v", err)
	}
	if len(formats) != 1 || formats[0] != "json" {
		t.Fatalf("export ran with %v, want one json export", formats)
	}
	text := strings.Join(tt.Texts(), "\n")
	if strings.Contains(text, "Settings\n") || strings.Contains(text, "设置\n") {
		t.Fatalf("the never-implemented settings button is still painted: %s", text)
	}
}
