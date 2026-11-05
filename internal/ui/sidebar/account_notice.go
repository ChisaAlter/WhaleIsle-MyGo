package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

type AccountNotice struct {
	ID, Title, Message string
	OnShown, OnDismiss func()
}

func accountNoticeStack(c *ui.Context, anchor ui.Element, notices []AccountNotice) {
	if len(notices) == 0 {
		return
	}
	n := notices[0]
	ui.Overlay(c.Key("account-notice/"+n.ID), func() {
		ui.Column(c).Width(260).MaxWidth(260).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 10, 0).Children(func() {
			if len(notices) > 2 {
				ui.Box(c).FillWidth().Height(5).Margin(0, 10).Radius(8).Background(c.Theme().Surface).Border(.5, c.Theme().Border)
			}
			if len(notices) > 1 {
				ui.Box(c).FillWidth().Height(5).Margin(0, 5).Radius(8).Background(c.Theme().Surface).Border(.5, c.Theme().Border)
			}
			ui.Column(c).FillWidth().Gap(8).Padding(16).Radius(12).Background(c.Theme().Surface).Border(.5, c.Theme().Border).Shadow(0, 6, 24, 0, ui.RGBA(0, 0, 0, .12)).Children(func() {
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, n.Title).Grow(1).FontSize(theme.InterfaceSize(14)).FontWeight(600)
					if ui.Button(c, "×").Label(workspaceText("关闭通知", "Dismiss notice")).Size(24, 24).Padding(0).BorderWidth(0).Background(ui.Transparent).Clicked() && n.OnDismiss != nil {
						n.OnDismiss()
					}
				})
				ui.Text(c, n.Message).FillWidth().FontSize(theme.InterfaceSize(13))
			})
		})
	})
	if n.OnShown != nil {
		n.OnShown()
	}
}
