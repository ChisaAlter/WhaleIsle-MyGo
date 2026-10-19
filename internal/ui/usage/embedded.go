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
	_, _, muted := usageCardPalette(c)
	fg, accent, field := chrome.Color(colors.FG), chrome.Color(colors.Accent), chrome.Color(colors.Field)
	ui.Column(c.Key("usage-stats-body")).FillWidth().MinWidth(0).Gap(16).Padding(16, 20, 28).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Wrap().Children(func() {
			ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
				ui.Icon(c, usagePageIcon).Size(16, 16).Margin(4, 0, 0).TextColor(muted)
				ui.Text(c, usageText("用量统计", "Usage statistics")).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(fg).Grow(1)
			})
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
			usageEmpty(c, "", usageText("连接 Host 后即可读取用量统计。", "Connect to a Host to read usage statistics."), fg, muted)
			return
		}
		statsView(c, m, fg, muted, accent, field)
	})
}
