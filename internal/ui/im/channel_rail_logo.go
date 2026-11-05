package im

import "github.com/egoist/mygo/ui"

func channelRailLogo(c *ui.Context, id string) {
	background, ink, size := c.Theme().Surface, ui.RGB(255, 255, 255), float32(21)
	border := false
	switch id {
	case "weixin":
		background, size = ui.RGB(7, 193, 96), 19
	case "feishu":
		border, size = true, 28
	case "dingtalk":
		background, size = ui.RGB(22, 119, 255), 24
	case "qq":
		background = ui.RGB(22, 119, 255)
	case "wecom":
		border, size = true, 22
	case "telegram":
		background = ui.RGB(34, 158, 217)
	case "discord":
		background = ui.RGB(88, 101, 242)
	case "slack":
		background = ui.RGB(74, 21, 75)
	case "whatsapp":
		background = ui.RGB(37, 211, 102)
	case "office":
		background, size = c.Theme().Accent, 23
	}
	box := ui.Box(c).Size(30, 30).Shrink(0).Radius(8).Background(background).Center()
	if border {
		box.Border(1, c.Theme().Border)
	}
	box.Children(func() { ui.Icon(c, channelLogos[id]).Size(size, size).TextColor(ink) })
}
