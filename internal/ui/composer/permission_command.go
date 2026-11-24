package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// nativeOptionLabel renders one row of the /permission picker. Permission
// presets go through the composer's single product-name mapping, so the chip
// and the command row can never disagree about a preset's name.
func nativeOptionLabel(row NativeCommandOption) string {
	return permissionLabel(row.ID, row.Label)
}

func paintNativePermissionConfirmation(c *ui.Context, m Model, p *nativeCommandPopup) {
	if p.confirmation == nil {
		return
	}
	target := *p.confirmation
	paintPermissionRiskDialog(c, "composer-permission-risk", target.ID, &p.acknowledged, p.busy, i18n.T(i18n.KeyParityPermConfirmDescription),
		func() {
			p.confirmation = nil
			p.acknowledged = false
		},
		func() {
			p.settle(m, target)
		},
	)
}

func paintPermissionRiskDialog(c *ui.Context, key string, permissionID string, acknowledged *bool, disabled bool, extraDescLabel string, onCancel, onConfirm func()) {
	title := composerText("确认启用完全权限？", "Enable Full access?")
	description := composerText("启用完全权限后，新会话将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任后续任务时使用。", "Full access lets new sessions reduce confirmation steps and perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust subsequent tasks.")
	action := composerText("启用完全权限", "Enable Full access")
	ackLabel := composerText("我已了解风险，并愿意继续", "I understand these risks and want to continue")
	if permissionID == session.PermissionAutoReview {
		title = composerText("确认启用自动审查（实验）？", "Enable Auto review (experimental)?")
		description = composerText("自动审查不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前 agent 相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = composerText("启用自动审查", "Enable Auto review")
		ackLabel = composerText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue")
	} else if extraDescLabel == i18n.T(i18n.KeyParityPermConfirmNewSessionDesc) {
		description = i18n.T(i18n.KeyParityPermConfirmDescription)
	}
	open := true
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Key:             key,
		Title:           title,
		Description:     description,
		MaxWidth:        520,
		DismissDisabled: disabled,
		Footer: func() {
			if coss.Button(c, composerText("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline, Disabled: disabled}).Clicked() {
				onCancel()
			}
			if coss.Button(c, action, coss.ButtonOptions{Variant: coss.ButtonDefault, Disabled: !*acknowledged || disabled}).Clicked() {
				onConfirm()
			}
		},
	}, func() {
		if permissionID != session.PermissionAutoReview {
			if extraDescLabel != "" {
				ui.Box(c).Size(0, 0).PassThrough().Label(extraDescLabel)
			}
			ui.Box(c).Size(0, 0).PassThrough().Label(i18n.T(i18n.KeyParityPermConfirmDescriptionAlt))
		}
		ui.Row(c).AlignSelf(ui.Start).Padding(design.Sp(1), 0).Children(func() {
			cb := coss.Checkbox(c, acknowledged, ackLabel, coss.CheckOptions{Disabled: disabled})
			_ = cb.Changed()
			ui.Box(c).Absolute().Fill().PassThrough().Label(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledge))
			ui.Box(c).Absolute().Fill().PassThrough().Label(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledgeAlt))
			ui.Box(c).Absolute().Fill().PassThrough().Label(composerText("我已了解风险，并愿意继续", "I understand these risks and want to continue"))
		})
	})
	if !open && !disabled {
		onCancel()
	}
}
