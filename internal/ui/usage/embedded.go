package usage

import (
	"context"
	"fmt"
	"time"
	usagestore "whaleisle/internal/store/usage"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// embeddedView is the usage-stats settings contribution. Account and jobs retain
// their own surfaces; this body follows dsh-usage-panel's StatsSection contract.
//
// 0.3.13 titles the page 用量与费用: the title is followed by the time of the
// data and the basis of the totals, the export / settings / refresh actions sit
// to the right. Settings opens billing; Host preference rows remain visible.
func embeddedView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	_, _, muted := usageCardPalette(c)
	fg, accent, field := chrome.Color(colors.FG), chrome.Color(colors.Accent), chrome.Color(colors.Field)
	body := ui.Column(c.Key("usage-stats-body")).FillWidth().MinWidth(0).Gap(16).Padding(16, 20, 28).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18))

	body.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Wrap().Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Icon(c, usagePageIcon).Size(16, 16).Margin(4, 0, 0).TextColor(muted)
					ui.Text(c, usageText("用量与费用", "Usage & cost")).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(fg).Grow(1)
				})
				if m.Store != nil {
					ui.Column(c).FillWidth().MinWidth(0).Gap(2).Padding(0, 0, 0, 24).Children(func() {
						if at := m.Store.Stats().UpdatedAt; at > 0 {
							ui.Text(c, usageUpdatedLabel(at)).FillWidth().TextColor(fg)
						}
						if overview := m.Store.Stats().Overview; overview != nil {
							label, source := usageSourceLabel(overview)
							ui.Text(c, label).FillWidth().TextColor(fg).Tooltip(source)
						}
					})
				}
			})
			if m.Store != nil {
				ui.Row(c).Shrink(0).Gap(8).Children(func() {
					exportDisabled := m.OnExportStats == nil
					export := widgets.Button(c, colors, usageText("导出", "Export"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: exportDisabled})
					if exportDisabled {
						export.Tooltip(usageText("本版本还没有统计导出。", "Exporting statistics is not wired in this build"))
					} else {
						export.Menu(func(menu *ui.Menu) {
							for _, format := range []struct{ id, zh, en string }{{"json", "导出 JSON", "Export JSON"}, {"days", "导出每日 CSV", "Export daily CSV"}, {"models", "导出模型 CSV", "Export model CSV"}} {
								if menu.Item(usageText(format.zh, format.en)).Chosen() {
									m.OnExportStats(format.id)
								}
							}
						})
					}
					pricingView(c, m, fg, muted, accent)
					if widgets.Button(c, colors, usageText("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(m.Store.Stats().Loading).Clicked() {
						m.Store.RefreshStats(context.Background(), true)
					}
				})
			}
		})
		if m.Preferences != nil {
			ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() { m.Preferences(c, fg, muted) })
		}
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

func usageSourceLabel(o *usagestore.Overview) (string, string) {
	if o.Coverage.Source == "magpie" && o.Coverage.Magpie != nil {
		m := o.Coverage.Magpie
		day := func(at *int64) string {
			if at == nil {
				return "—"
			}
			return time.UnixMilli(*at).UTC().Format("2006-01-02")
		}
		return fmt.Sprintf(usageText("总计口径：Magpie · %d 次调用 · %s → %s UTC", "Totals basis: Magpie · %d calls · %s → %s UTC"), m.Calls, day(m.From), day(m.To)), m.File
	}
	return usageText("总计口径：会话日志（输入+输出，fork 继承前缀已去重）", "Totals basis: session logs (input + output, fork-inherited prefixes de-duplicated)"), ""
}
