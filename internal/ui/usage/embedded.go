package usage

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// embeddedView is the usage-stats settings contribution. Account and jobs retain
// their own surfaces; this body follows dsh-usage-panel's StatsSection contract.
func embeddedView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, accent, field := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Accent), chrome.Color(colors.Field)
	ui.Column(c.Key("usage-stats-body")).FillWidth().MinWidth(0).Gap(16).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Children(func() {
			ui.Text(c, usageText("用量统计", "Usage statistics")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg).Grow(1)
			if m.Store != nil {
				if ui.Button(c, usageText("刷新", "Refresh")).Disabled(m.Store.Stats().Loading).Clicked() {
					m.Store.RefreshStats(context.Background(), true)
				}
			}
		})
		if m.ActionError != "" {
			ui.Text(c, m.ActionError).TextColor(accent)
		}
		if m.Store == nil {
			ui.Text(c, usageText("连接 Host 后即可读取用量统计。", "Connect to a Host to read usage statistics.")).TextColor(muted)
			return
		}
		statsView(c, m, fg, muted, accent, field)
	})
}
