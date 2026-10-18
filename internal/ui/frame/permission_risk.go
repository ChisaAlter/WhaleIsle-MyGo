package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func permissionRiskConfirmation(c *ui.Context, m Model) {
	request := m.Store.PermissionRiskConfirmation()
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := archiveText("确认启用完全权限？", "Enable Full access?")
	description := archiveText("启用完全权限后，智能体将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任当前任务时使用。", "Full access reduces confirmation steps and lets the agent perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust the current task.")
	action := archiveText("启用完全权限", "Enable Full access")
	if request.Value == session.PermissionAutoReview {
		title = archiveText("确认启用 Auto review（实验）？", "Enable Auto review (experimental)?")
		description = archiveText("Auto review 不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前 agent 相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = archiveText("启用 Auto review", "Enable Auto review")
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(520).MaxWidthPercent(92).Padding(20).Background(chrome.Color(colors.BG)).Radius(14).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).Bold().TextColor(chrome.Color(colors.FG))
				ui.Text(c, request.SessionID).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
				ui.Text(c, description).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
				acknowledged := request.Acknowledged
				ui.Checkbox(c, &acknowledged, archiveText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue"))
				if acknowledged != request.Acknowledged {
					m.Store.AcknowledgePermissionRisk(request.Token, acknowledged)
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, archiveText("取消", "Cancel")).Clicked() {
						m.Store.CancelPermissionRisk(request.Token)
					}
					if ui.Button(c, action).Disabled(!acknowledged).Clicked() {
						m.Store.ConfirmPermissionRisk(context.Background(), request.Token)
					}
				})
			})
		})
	})
	if !open {
		m.Store.CancelPermissionRisk(request.Token)
	}
}
