package frame

import (
	"context"
	"strconv"

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
		title = archiveText("确认启用自动审查（实验）？", "Enable Auto review (experimental)?")
		description = archiveText("自动审查不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前智能体相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = archiveText("启用自动审查", "Enable Auto review")
	}
	permissionRiskDialog(c, colors, request.Token, title, description, request.SessionID, action, request.Acknowledged,
		func(acknowledged bool) { m.Store.AcknowledgePermissionRisk(request.Token, acknowledged) },
		func() { m.Store.CancelPermissionRisk(request.Token) },
		func() { m.Store.ConfirmPermissionRisk(context.Background(), request.Token) },
	)
}

// permissionRiskDialog draws the confirmation and keeps its acknowledgement in
// element-local state, so a click survives the frame while the store catches
// up. Without that, the refilled request.Acknowledged resets the box to off
// every frame and the primary button can never enable.
func permissionRiskDialog(c *ui.Context, colors theme.Colors, token uint64, title, description, sessionID, action string, acknowledged bool, onAcknowledge func(bool), onCancel, onConfirm func()) {
	open := true
	ui.DialogBase(c.Key("permission-risk"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurface(c, backdrop, panel, colors, 520, 24).Children(func() {
			ack := ui.Local(panel, "permission-risk-ack-"+strconv.FormatUint(token, 10), func() bool {
				return acknowledged
			})
			ui.Column(c).Gap(14).Children(func() {
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).Bold().TextColor(chrome.Color(colors.FG))
				ui.Text(c, description).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG)).LineHeight(1.5)
				ui.Checkbox(c, ack, archiveText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue"))
				if *ack != acknowledged {
					onAcknowledge(*ack)
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, archiveText("取消", "Cancel")).Clicked() {
						onCancel()
					}
					if ui.PrimaryButton(c, action).Disabled(!*ack).Clicked() {
						onConfirm()
					}
				})
			})
		})
	})
	if !open {
		onCancel()
	}
}
