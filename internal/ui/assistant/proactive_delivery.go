package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/theme"
)

func proactiveDeliveryPreview(c *ui.Context, d proactive.Delivery, fg, muted ui.Color) {
	ui.Text(c, d.Route.Title).FontSize(theme.InterfaceSize(13)).TextColor(fg)
	ui.Text(c, whaleTaskCopy("会话：", "Conversation: ")+d.Route.ChatID+"\nBot: "+d.Route.BotID+"\n"+whaleTaskCopy("授权人：", "Authorized by: ")+d.Route.Actor).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
	ui.Text(c, d.Text).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
}
func proactiveDeliveryReceipt(c *ui.Context, d proactive.Delivery, fg, muted ui.Color) {
	labels := map[string]string{"pending": "等待批准", "ready": "已批准，等待发送", "sending": "发送中", "sent": "已发送", "failed": "未发送", "unknown": "发送结果待核对，不会自动重发", "rejected": "已拒绝发送", "cancelled": "已取消发送"}
	ui.Text(c, whaleTaskCopy(labels[d.Status], d.Status)+" · "+d.Route.ChatID).FontSize(theme.InterfaceSize(12)).TextColor(fg).Selectable()
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
	}
	for _, id := range d.MessageIDs {
		ui.Text(c, whaleTaskCopy("消息回执：", "Message receipt: ")+id).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
	}
}
func proactiveDeliveryForm(c *ui.Context, p ProactiveModel, t proactive.Task, fg, muted, border ui.Color) {
	if p.OnDelivery == nil || t.Execution == nil || t.Execution.Phase != "completed" || (t.State != "verifying" && t.State != "delivered") {
		return
	}
	for _, a := range p.State.Outbox {
		if a.TaskID == t.ID && a.Delivery != nil && (a.Delivery.Status == "pending" || a.Delivery.Status == "ready" || a.Delivery.Status == "sending") {
			return
		}
	}
	form := ui.Column(c.Key("delivery-" + t.ID)).FillWidth().MinWidth(0).Gap(8)
	open := ui.Local(form, "open", func() bool { return false })
	routeID := ui.Local(form, "route", func() string { return "" })
	text := ui.Local(form, "text", func() string { return t.Execution.Summary })
	form.Children(func() {
		ui.Row(c).Children(func() {
			if ui.Button(c, whaleTaskCopy("发送结果到飞书", "Send result to Feishu")).Disabled(!p.CanRun || p.Busy).Clicked() {
				*open = !*open
				if *open && p.OnDeliveryRefresh != nil {
					p.OnDeliveryRefresh()
				}
			}
		})
		if !*open {
			return
		}
		ui.Text(c, whaleTaskCopy("选择已授权的任务会话", "Choose an authorized task conversation")).FontSize(theme.InterfaceSize(13)).TextColor(fg)
		if p.DeliveryError != "" {
			ui.Text(c, p.DeliveryError).TextColor(muted).Selectable()
		}
		selected := false
		for _, route := range p.DeliveryRoutes {
			ui.Column(c.Key(route.ID)).FillWidth().MinWidth(0).Gap(4).Children(func() {
				label := route.Title
				if *routeID == route.ID {
					selected = true
					label = "✓ " + label
				}
				if ui.Button(c, label).Disabled(p.Busy || p.DeliveryBusy).Clicked() {
					*routeID = route.ID
				}
				ui.Text(c, route.ChatID+" · "+route.BotID).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
			})
		}
		if len(p.DeliveryRoutes) == 0 && !p.DeliveryBusy {
			ui.Text(c, whaleTaskCopy("暂无可用会话。请连接飞书机器人、授权任务用户，并在目标会话提交一次 /task。", "No conversations available. Connect a Feishu bot, authorize its task owner, and submit /task in the destination conversation.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if ui.Button(c, whaleTaskCopy("刷新会话", "Refresh conversations")).Disabled(p.DeliveryBusy || p.OnDeliveryRefresh == nil).Clicked() {
			p.OnDeliveryRefresh()
		}
		ui.Text(c, whaleTaskCopy("发送内容", "Message")).FontSize(theme.InterfaceSize(13)).TextColor(fg)
		ui.TextArea(c.Key("delivery-text"), text).Label(whaleTaskCopy("发送内容", "Message")).FillWidth().MinWidth(0).Lines(3, 10).Disabled(p.Busy)
		ui.Text(c, whaleTaskCopy("下一步核对目标和正文，批准后发送。", "Review the destination and message next; sending begins after approval.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if ui.Button(c, whaleTaskCopy("提交发送审批", "Review delivery")).Disabled(p.Busy || p.DeliveryBusy || !selected || strings.TrimSpace(*text) == "" || t.ApprovalID != "").Clicked() {
			p.OnDelivery(t.ID, *routeID, *text)
		}
	})
}
