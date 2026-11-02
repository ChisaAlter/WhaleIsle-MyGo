package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// petHeader is the 0.3.13 page head: a 16/24 semibold title and the one-line
// intro, which now also says that the desktop and IM share one conversation.
func petHeader(c *ui.Context, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(8, 0, 0, 0).Children(func() {
		ui.Text(c, providerCopy("鲸鱼娘", "Whale")).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
		ui.Text(c, providerCopy("你的常驻助理与桌面伙伴。桌面和 IM 共用她的同一条对话。", "Your resident assistant and desktop companion. The desktop and IM share her one conversation.")).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
	})
}

// petTabs is the segmented strip under the head: a 16 DIP rounded well with
// 180x34 tabs, the active one outlined (not filled) and semibold. It returns
// the clicked index or -1.
func petTabs(c *ui.Context, key string, labels []string, active int, fg ui.Color) int {
	dark := c.Theme().Dark
	colors := theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
	fill, outline, secondary := theme.Color(colors.SelectorFill), theme.Color(colors.BorderL3), theme.Color(colors.Secondary)
	picked := -1
	ui.Row(c.Key(key)).AlignSelf(ui.Start).Shrink(0).Padding(4).Gap(0).Radius(16).Background(fill).Margin(12, 0, 8).Children(func() {
		for index, label := range labels {
			selected := index == active
			tab := ui.Button(c.Key(key+"-"+label), "").Label(label).Width(180).Height(34).Shrink(0).Padding(0, 12).Radius(12).BorderWidth(0).
				Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).FocusRing(false)
			ink, weight := secondary, 400
			if selected {
				tab.Background(fill).Border(.667, outline)
				ink, weight = fg, 600
			}
			tab.Children(func() {
				ui.Text(c, label).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(weight).TextColor(ink).SingleLine()
			})
			if tab.Clicked() {
				picked = index
			}
		}
	})
	return picked
}

// petGroupLabel is the small group title above a run of rows ("助理").
func petGroupLabel(c *ui.Context, text string) {
	dark := c.Theme().Dark
	secondary := theme.Color(theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark).Secondary)
	ui.Text(c, text).FillWidth().Padding(16, 0, 0, 0).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).TextColor(secondary)
}
