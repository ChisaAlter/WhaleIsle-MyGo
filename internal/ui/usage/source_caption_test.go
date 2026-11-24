package usage

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// TestUsageSourceCaptionKeepsThePathInDetails covers the totals caption a live
// pass read as C:\Users\48818\.config\magpie\usage.jsonl: the line says where the
// totals come from, and the file's path is a disclosure because that is how a
// person points support at the record.
func TestUsageSourceCaptionKeepsThePathInDetails(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	const path = `C:\Users\48818\.config\magpie\usage.jsonl`
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Gap(2).Children(func() {
			usageSourceLine(c, path)
		})
	}, 640, 200)

	shown := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(shown, "统计来源：本机网关用量记录") {
		t.Fatalf("the caption does not say where the totals come from: %q", shown)
	}
	if strings.Contains(shown, "usage.jsonl") || strings.Contains(shown, `C:\Users`) {
		t.Fatalf("the caption is still a filesystem path: %q", shown)
	}
	if err := tt.Click("详情"); err != nil {
		t.Fatalf("details control: %v", err)
	}
	opened := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(opened, path) {
		t.Fatalf("the path is unreachable after the disclosure opened: %q", opened)
	}
}
