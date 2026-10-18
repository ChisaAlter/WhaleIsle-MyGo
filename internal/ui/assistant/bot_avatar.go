package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botAvatarAcknowledgement = "我理解头像替换、会话展示及目标可能执行的影响 / I understand avatar replacement, presentation and execution effects"

type BotAvatarModel struct {
	OnChooseImage, OnApplyImageCrop, OnCancelImageCrop            func()
	OnImageCropChange                                             func(avatarimage.Crop)
	State                                                         dshbotstore.AvatarSnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotAvatarValues)
	OnAcknowledge                                                 func(bool)
}

func botAvatarBlock(c *ui.Context, m BotAvatarModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-avatar")).Gap(7).Children(func() {
		ui.Text(c, "个人 Bot 头像 / Personal Bot avatar").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "原版8种轮廓×11种颜色，支持明确选择本地JPEG/PNG并裁剪。原生静态预览；动画仍在Bot Web。不自动拉取远程图片或读取文件路径。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新头像 / Refresh avatar").Disabled(blocked || s.Scope.BotID == "" || s.ImageImport.Busy || s.ImageImport.Ready).Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读回核实；未知结果保留并锁定草稿，只读回，不自动重试 / Read back before saving").TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotAvatarValue(s.Status)
		if err != nil {
			ui.Text(c, "头像记录无法无损读取，请用 Bot Web / Unsupported avatar; use Bot Web").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n当前已保存头像 / Saved avatar: %s", m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botAvatarSummary(current))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		previewErr := botAvatarPreview(c.Key("saved-avatar"), current, 64, "已保存头像 / Saved avatar preview", fg, muted)
		if previewErr != nil {
			ui.Text(c, "现有图片无法安全预览，保留原头像；请用 Bot Web 核对并编辑 / Existing image unavailable; preserved, use Bot Web").TextColor(accent)
		}
		if !s.Writable {
			ui.Text(c, "Host 写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil && previewErr == nil
		botAvatarImageImportBlock(c.Key("image-import"), m, editable, fg, muted, accent)
		editable = editable && !s.ImageImport.Busy && !s.ImageImport.Ready
		if current.Kind == "image" {
			ui.Text(c, "现有图片保持不变。选择并应用本地图片，或明确点击轮廓/颜色，才生成替换草稿；最终仍需审阅与确认保存。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		draft := s.Draft
		color, shape := "ink", "circle"
		if draft.Kind == "blob" {
			color, shape = draft.Color, draft.Shape
		}
		shapes := pluginrpc.DshbotAvatarShapes()
		for row := 0; row < len(shapes); row += 4 {
			ui.Row(c.Key(fmt.Sprintf("shapes-%d", row))).Gap(6).Children(func() {
				for _, choice := range shapes[row:min(row+4, len(shapes))] {
					v := pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: choice, Color: color}
					ui.Column(c.Key("shape-" + choice)).Gap(2).Children(func() {
						botAvatarDrawBlob(c, v, 40, "shape preview "+choice, fg, muted)
						button := ui.Button(c, "轮廓 / Shape: "+choice).FontSize(fonttheme.InterfaceSize(11)).Disabled(!editable)
						if draft.Kind == "blob" && draft.Shape == choice {
							button.Border(2, accent)
						}
						if button.Clicked() && m.OnChange != nil {
							m.OnChange(v)
						}
					})
				}
			})
		}
		colors := pluginrpc.DshbotAvatarColors()
		for row := 0; row < len(colors); row += 4 {
			ui.Row(c.Key(fmt.Sprintf("colors-%d", row))).Gap(6).Children(func() {
				for _, choice := range colors[row:min(row+4, len(colors))] {
					v := pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: shape, Color: choice}
					ui.Column(c.Key("color-" + choice)).Gap(2).Children(func() {
						botAvatarDrawBlob(c, v, 32, "color preview "+choice, fg, muted)
						button := ui.Button(c, "颜色 / Color: "+choice).FontSize(fonttheme.InterfaceSize(11)).Disabled(!editable)
						if draft.Kind == "blob" && draft.Color == choice {
							button.Border(2, accent)
						}
						if button.Clicked() && m.OnChange != nil {
							m.OnChange(v)
						}
					})
				}
			})
		}
		ui.Text(c, "草稿 / Draft: "+botAvatarSummary(draft)).TextColor(fg).Selectable()
		_ = botAvatarUncachedPreview(c.Key("draft-avatar"), draft, 64, "头像草稿 / Draft avatar preview", fg, muted)
		if draft.Kind == "image" {
			for _, crop := range []string{"circle", "square"} {
				if ui.Button(c, "草稿图片显示 / Draft image crop: "+crop).Disabled(!editable).Clicked() && m.OnChange != nil {
					next := draft
					next.Crop = crop
					m.OnChange(next)
				}
			}
		}
		if ui.Button(c, "丢弃头像草稿 / Discard avatar draft").Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅头像 / Review avatar").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, "采用匹配头像读回 / Adopt matching avatar readback").Disabled(blocked || previewErr != nil).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		before, _ := pluginrpc.DshbotAvatarValue(cmd.Expected)
		ui.Column(c.Key("bot-avatar-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认头像替换 / Confirm avatar replacement").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n原头像 / Before: %s", m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, botAvatarSummary(before))).TextColor(fg).Selectable()
			beforeErr := botAvatarPreview(c.Key("before-avatar"), before, 72, "原头像 / Before avatar preview", fg, muted)
			ui.Text(c, "新头像 / After: "+botAvatarSummary(cmd.Values)).TextColor(fg).Selectable()
			afterErr := botAvatarUncachedPreview(c.Key("after-avatar"), cmd.Values, 72, "新头像 / After avatar preview", fg, muted)
			if before.Kind == "image" && cmd.Values.Kind == "blob" {
				ui.Text(c, "将用上面的 blob 替换当前内嵌图片；保存后该 Bot 的头像记录不再保留旧图片。不会删除原始本地图片文件。").TextColor(accent)
			}
			if cmd.Values.Kind == "image" {
				ui.Text(c, "确认保存后，上方预览图片的内嵌dataURL将随完整Bot资料发送至当前Host："+m.HostLabel+"。新导入图为192px JPEG；仅修改旧图圆形/方形显示时原dataURL保持不变。不会发送源路径；本地重新编码的图片不带原图元数据。旧头像记录将被替换，原始本地文件不删除。").TextColor(accent).Selectable()
			}
			ui.Text(c, "bot/update 会写审计、更新会话 presentation，并尝试 rename 会话标题。展示/重命名或回滚可能失败。虽然仅改头像，保存仍校验原模型/能力并 requestSync：live idle 可能创建目标、重建已暂停目标或修改 active 目标；可能调用模型、执行工具、发送外部消息并产生费用。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "忙碌/冷会话可能后续 idle 同步；同步失败可能只记Host日志。资料回执不证明会话展示、重命名、目标同步或执行完成。revision CAS 不原子锁定 live 目标；不停止在途执行，不撤销已发生操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf("保留名称/标题/描述、模型、工作区及其他完整资料。workspaceId: %s\nmodel: %s\n不主动 prompt/start/resume/open。关闭不撤回已发请求；未知结果锁定，只读回，不自动重试。取消图片选择会清除本次临时源图；取消图片保存或隐藏会丢弃未发送图片草稿。已发送结果未知时保留审阅图用于读回核对。", botCapabilityProfileText(cmd.Expected, "workspaceId"), botCapabilityProfileText(cmd.Expected, "model"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botAvatarAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, "取消头像保存 / Cancel avatar save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存头像 / Confirm avatar save").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged || beforeErr != nil || afterErr != nil).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
