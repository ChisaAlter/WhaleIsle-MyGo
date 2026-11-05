package usage

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func rankDialog(c *ui.Context, open *bool, title string, content func()) {
	ui.DialogBase(c, open, func(back, panel ui.Element) {
		widgets.RevealOverlay(c, back, panel, ui.RGBA(0, 0, 0, .4))
		w, h := c.Size()
		panel.Width(max(0, min(620, w-48))).MaxHeight(max(0, h-48)).Radius(16).Background(c.Theme().Background).Padding(24).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() {
				ui.Row(c).FillWidth().Gap(12).Children(func() {
					ui.Text(c, title).Grow(1).FontSize(theme.InterfaceSize(16)).FontWeight(600)
					if ui.Button(c, "×").Label(usageText("关闭", "Close")).Size(28, 28).Padding(0).BorderWidth(0).Background(ui.Transparent).Clicked() {
						*open = false
					}
				})
				ui.Scroll(c).FillWidth().MaxHeight(max(80, h-180)).Children(func() { ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(content) })
			})
		})
	})
}
func rankingSort(c *ui.Context, value *string, busy bool, changed func()) {
	label := usageText("Token", "Tokens")
	if *value == "cost" {
		label = usageText("费用", "Cost")
	}
	ui.Row(c).FillWidth().Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(func() {
		ui.Text(c, usageText("排序", "Sort by")).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		if ui.Select(c, &label, []string{usageText("Token", "Tokens"), usageText("费用", "Cost")}).Label(usageText("排序", "Sort by")).Width(100).Height(28).Disabled(busy).Changed() {
			*value = "tokens"
			if label == usageText("费用", "Cost") {
				*value = "cost"
			}
			changed()
		}
	})
}
func rankTableHeader(c *ui.Context, sessions bool, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(6, 0).Children(func() {
		ui.Text(c, "#").Width(22).TextColor(muted)
		label := usageText("工作目录", "Project")
		if sessions {
			label = usageText("会话", "Session")
		}
		ui.Text(c, label).Grow(1).MinWidth(0).TextColor(muted)
		if sessions {
			ui.Text(c, usageText("类型", "Type")).Width(52).TextColor(muted)
			ui.Text(c, usageText("活跃日期", "Last active")).Width(78).TextColor(muted)
		}
		ui.Text(c, usageText("估算费用", "Est. cost")).Width(62).TextAlign(ui.End).TextColor(muted)
		ui.Text(c, "Tokens").Width(62).TextAlign(ui.End).TextColor(muted)
	})
}
func rankMore(c *ui.Context, loading, started, hasMore bool, err string, load func(), muted ui.Color) {
	if err != "" {
		ui.Text(c, err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger)
	}
	if loading {
		ui.Text(c, usageText("正在加载…", "Loading…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		return
	}
	if !started || hasMore || err != "" {
		label := usageText("更多", "More")
		if err != "" {
			label = usageText("重试", "Retry")
		}
		if ui.Button(c, label).AlignSelf(ui.Start).Clicked() {
			load()
		}
	}
}
