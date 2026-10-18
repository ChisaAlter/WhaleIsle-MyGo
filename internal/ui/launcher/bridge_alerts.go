package launcher

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/bridgealerts"
)

type BridgeAlertsModel struct {
	Snapshot    bridgealerts.Snapshot
	ActionError string
	OnOpen      func(bridgealerts.Ticket)
	OnContinue  func(bridgealerts.Ticket)
}

func bridgeAlertSummary(c *ui.Context, m Model, details bool) {
	b := m.BridgeAlerts
	s := b.Snapshot
	if len(s.Alerts) == 0 && s.Status == "" && b.ActionError == "" {
		return
	}
	ui.Column(c).FillWidth().Gap(7).Padding(12).Background(colWarnBG).Radius(8).Children(func() {
		ui.Text(c, releaseText("鲸桥额度与到期提醒", "WhaleBridge quota and expiration reminders")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(colPrimary)
		if s.Fallback {
			ui.Text(c, releaseText("系统通知不可用或发送失败；已收到的提醒保留在此页。", "System notifications are unavailable or failed. Received reminders remain here.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if s.Status != "" {
			ui.Text(c, s.Status).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if b.ActionError != "" {
			ui.Text(c, b.ActionError).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if details {
			for _, notice := range s.Alerts {
				ui.Column(c).FillWidth().Gap(3).Children(func() {
					ui.Text(c, notice.Title).FontSize(fonttheme.InterfaceSize(12)).FontWeight(600).TextColor(colPrimary)
					ui.Text(c, notice.Body).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
				})
			}
			if s.Dropped > 0 {
				ui.Text(c, fmt.Sprintf(releaseText("此页保留最近100条；更早的%d条请在鲸桥管理页提醒历史中查看。", "The latest 100 are retained here; view %d earlier reminders in WhaleBridge history."), s.Dropped)).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
			}
		} else if len(s.Alerts) > 0 {
			ui.Text(c, fmt.Sprintf(releaseText("已收到 %d 条提醒，可在组件页查看。", "%d reminders received. View them in Components."), len(s.Alerts))).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		ui.Row(c).Gap(8).Children(func() {
			if !details && ghostSmall(c, releaseText("查看提醒", "View reminders")).Clicked() && m.OnTab != nil {
				m.OnTab(TabComponents)
			}
			if s.Manage.Scope != "" && ghostSmall(c, releaseText("打开此鲸桥实例管理页", "Manage this WhaleBridge instance")).Clicked() && b.OnOpen != nil {
				b.OnOpen(s.Manage)
			}
			if s.Paused && ghostSmall(c, releaseText("继续接收后续提醒", "Continue receiving future reminders")).Clicked() && b.OnContinue != nil {
				b.OnContinue(s.Manage)
			}
		})
	})
}
