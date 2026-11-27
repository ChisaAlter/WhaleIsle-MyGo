package frame

import (
	"context"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

func permissionRiskConfirmation(c *ui.Context, m *Model) {
	if m.Store == nil {
		return
	}
	store := m.Store
	request := store.PermissionRiskConfirmation()
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := archiveText("确认启用完全权限？", "Enable Full access?")
	description := archiveText("启用完全权限后，新会话将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任后续任务时使用。", "Full access lets new sessions reduce confirmation steps and perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust subsequent tasks.")
	action := archiveText("启用完全权限", "Enable Full access")
	if request.Value == session.PermissionAutoReview {
		title = archiveText("确认启用自动审查（实验）？", "Enable Auto review (experimental)?")
		description = archiveText("自动审查不使用沙箱。每次原生工具调用和 PTC 内层调用前，都会由与当前智能体相同的模型进行审查；审查拒绝的调用由你批准或拒绝。此功能仍属实验性，可能误放行或误拒绝，并会消耗额外 token。", "Auto review runs without a sandbox. Before every native tool call and PTC inner call, the same model as the current agent reviews whether to allow it; you approve or reject each call it denies. This feature is experimental, can falsely allow or deny actions, and uses additional tokens.")
		action = archiveText("启用自动审查", "Enable Auto review")
	}
	token := request.Token
	permissionRiskDialog(c, colors, token, title, description, request.SessionID, action, request.Acknowledged,
		func(acknowledged bool) { store.AcknowledgePermissionRisk(token, acknowledged) },
		func() { store.CancelPermissionRisk(token) },
		func() { store.ConfirmPermissionRisk(context.Background(), token) },
	)
}

// permissionRiskDialog draws the confirmation and keeps its acknowledgement in
// element-local state, so a click survives the frame while the store catches
// up. Without that, the refilled request.Acknowledged resets the box to off
// every frame and the primary button can never enable.
func permissionRiskDialog(c *ui.Context, colors theme.Colors, token uint64, title, description, sessionID, action string, acknowledged bool, onAcknowledge func(bool), onCancel, onConfirm func()) {
	_ = colors
	open := true
	anchor := ui.Box(c.Key("permission-risk-state")).Absolute().Size(0, 0).PassThrough()
	ack := ui.Local(anchor, "permission-risk-ack-"+strconv.FormatUint(token, 10), func() bool {
		return acknowledged
	})
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Title:         title,
		Description:   description,
		MaxWidth:      520,
		FooterVariant: coss.FooterBare,
		Key:           "permission-risk",
		Footer: func() {
			if coss.Button(c, archiveText("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline}).Clicked() {
				onCancel()
			}
			if coss.Button(c, action, coss.ButtonOptions{Variant: coss.ButtonDefault, Disabled: !*ack}).Clicked() {
				onConfirm()
			}
		},
	}, func() {
		ui.Column(c).FillWidth().Gap(design.Sp(2)).Children(func() {
			if coss.Checkbox(c, ack, archiveText("我已了解风险，并愿意继续", "I understand these risks and want to continue"), coss.CheckOptions{}).Changed() || *ack != acknowledged {
				onAcknowledge(*ack)
			}
		})
	})
	if !open {
		onCancel()
	}
}
