package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func nativeOptionLabel(row NativeCommandOption) string {
	switch row.ID {
	case session.PermissionAutoReview:
		return composerText("自动审查", "Auto review")
	case "read-only":
		if row.Label == row.ID || row.Label == "Read Only" {
			return composerText("仅可查看", "Read Only")
		}
	case "workspace-write":
		if row.Label == row.ID || row.Label == "Workspace Write" {
			return composerText("工作区写入", "Workspace Write")
		}
	case session.PermissionFullAccess:
		if row.Label == row.ID || row.Label == "Full access" {
			return composerText("完全权限", "Full access")
		}
	}
	return row.Label
}
func paintNativePermissionConfirmation(c *ui.Context, m Model, p *nativeCommandPopup, panel, fg ui.Color) {
	title := composerText("确认启用完全权限？", "Enable Full access?")
	description := composerText("启用完全权限后，智能体将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任当前任务时使用。", "Full access reduces confirmation steps and lets the agent perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust the current task.")
	action := composerText("启用完全权限", "Enable Full access")
	if p.confirmation.ID == session.PermissionAutoReview {
		title = composerText("确认启用自动审查（实验）？", "Enable Auto review (experimental)?")
		description = composerText("自动审查不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前 agent 相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = composerText("启用自动审查", "Enable Auto review")
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, dialog ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		dialog.Width(520).MaxWidthPercent(92).Padding(20).Background(panel).Radius(14).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, title).FontSize(16).Bold().TextColor(fg)
				ui.Text(c, description).FontSize(12).TextColor(fg)
				ui.Checkbox(c, &p.acknowledged, composerText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue")).Disabled(p.busy)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, composerText("取消", "Cancel")).Disabled(p.busy).Clicked() {
						p.confirmation = nil
						p.acknowledged = false
					}
					if ui.Button(c, action).Disabled(!p.acknowledged || p.busy).Clicked() {
						p.settle(m, *p.confirmation)
					}
				})
			})
		})
	})
	if !open && !p.busy {
		p.confirmation = nil
		p.acknowledged = false
	}
}
