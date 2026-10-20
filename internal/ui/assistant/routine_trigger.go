package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func routineTriggerAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIApproveObtainingPersistentHostCata)
}
const routineTriggerImpact = "此操作调用原版 bot/trigger-token：已有凭据会返回，缺失时会创建并保存持久凭据。凭据按此 Host 的 Bot 目录共享，并非单个任务专属；持有者且能访问该 Host 触发端点，可触发该目录其他已知任务 ID，包括暂停任务，可能执行工具、发送外部消息并产生模型费用。手动触发会跳过 watch 检查，仍受 Host 忙碌/次数限制。"

func routineTriggerBlock(c *ui.Context, m RoutineModel, blocked bool, fg, muted, accent ui.Color) {
	s := m.State.Trigger
	if m.OnPrepareTrigger == nil && m.TriggerUnavailable == "" {
		return
	}
	ui.Column(c.Key("routine-trigger-access")).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCurrentHostBotCatalogTriggerCredent)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		if m.TriggerUnavailable != "" {
			ui.Text(c, m.TriggerUnavailable).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			return
		}
		if !s.Open {
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewHostCatalogCredential)).Disabled(blocked || !m.State.Loaded).Clicked() {
				m.OnPrepareTrigger()
			}
			ui.Text(c, "打开页面或刷新目录不会调用凭据接口；此处没有运行任务按钮。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nRoutine: %s (%s) · revision %d", m.HostLabel, m.State.BotName, m.State.Scope.BotID, s.SessionID, s.RoutineName, s.RoutineID, s.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if s.Review {
			ui.Text(c, routineTriggerImpact).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "确认绑定当前已认证 Host 连接和目录账号；账号或连接变化会使确认失效。原版没有独立只读获取、rotate 或 revoke 接口。关闭本页只清除显示，不撤销 Host 上的凭据。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := s.Acknowledged
			if ui.Checkbox(c, &ack, routineTriggerAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledgeTrigger != nil {
				m.OnAcknowledgeTrigger(ack)
			}
			if m.OnConfirmTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmHostCatalogCredential)).Disabled(blocked || !ack || !m.State.Loaded).Clicked() {
				m.OnConfirmTrigger()
			}
		}
		if s.Busy {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWaitingForHostClosingDoesNotRevoke)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		if s.Ready && m.TriggerMaterial != nil {
			address, header := m.TriggerMaterial()
			if address != "" && header != "" {
				ui.Text(c, "POST "+address).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, header).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "敏感共享凭据，仅本页临时显示。复制鉴权头会进入系统剪贴板，可能被历史/同步记录；本应用清除显示不会清除系统剪贴板。").FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
				ui.Row(c).Gap(8).Children(func() {
					if m.OnCopyTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCopyTriggerAddress)).Disabled(blocked).Clicked() {
						m.OnCopyTrigger("address")
					}
					if m.OnCopyTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCopySecretHeader)).Disabled(blocked).Clicked() {
						m.OnCopyTrigger("header")
					}
				})
			}
		}
		if m.OnCloseTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCloseAndClearCredential)).Clicked() {
			m.OnCloseTrigger()
		}
	})
}
