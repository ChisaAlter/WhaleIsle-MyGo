package sidebar

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestUpdateSeatSpeaksTheActiveLocale pins the sidebar's update entry: in a
// Chinese interface it reads Chinese. The seat is 36 DIP with icon only, so the
// label is what a screen reader and a tooltip say — the string a person quoted
// from the sidebar when it read "Checking for updates…".
func TestUpdateSeatSpeaksTheActiveLocale(t *testing.T) {
	phases := map[string]string{
		"checking":    "正在检查更新…",
		"available":   "有可用更新",
		"downloading": "正在下载更新…",
		"verifying":   "正在校验更新…",
		"installing":  "正在准备重启…",
		"ready":       "安装并重启",
		"error":       "重试更新",
	}
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for phase, want := range phases {
		tt := ui.NewTester(func(c *ui.Context) {
			desktopUpdateSeat(c, Model{}, phase, theme.Resolve("deepseek.light").Light)
		}, 60, 60)
		painted := strings.Join(tt.Texts(), "\n")
		if !strings.Contains(painted, want) {
			t.Fatalf("phase %q = %q, want %q", phase, painted, want)
		}
		for _, english := range []string{"Checking for updates", "Update available", "Downloading update", "Verifying update", "Preparing to restart", "Install and Restart", "Retry update"} {
			if strings.Contains(painted, english) {
				t.Fatalf("phase %q painted English %q in a Chinese interface", phase, english)
			}
		}
	}
}

// TestConnectionSeatSpeaksTheActiveLocale covers the seat beside it, which
// shared the same helper and therefore the same dead zh branch.
func TestConnectionSeatSpeaksTheActiveLocale(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, tc := range []struct{ status, want string }{
		{"disconnected", "连接异常"},
		{"connecting", "连接中断，正在重试"},
		{"recovered", "连接成功"},
	} {
		tt := ui.NewTester(func(c *ui.Context) {
			connectionStatusSeat(c, Model{OnReconnect: func() {}}, tc.status, theme.Resolve("deepseek.light").Light, true)
		}, 60, 60)
		painted := strings.Join(tt.Texts(), "\n")
		if !strings.Contains(painted, tc.want) {
			t.Fatalf("status %q = %q, want %q", tc.status, painted, tc.want)
		}
		for _, english := range []string{"Disconnected", "Reconnecting", "Connected"} {
			if strings.Contains(painted, english) {
				t.Fatalf("status %q painted English %q in a Chinese interface", tc.status, english)
			}
		}
	}
}
