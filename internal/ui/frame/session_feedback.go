package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func sessionFeedbackDialog(c *ui.Context, m Model) {
	v := m.Store.SessionFeedback()
	if !v.Open || v.SessionID != m.Store.Snapshot().SelectedID {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurface(c, backdrop, panel, colors, 520, 24).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				ui.Text(c, archiveText("会话反馈", "Session feedback")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(chrome.Color(colors.FG))
				ui.Text(c, archiveText("提交后，Host 会记录这条反馈，并可能发送相关会话日志供审阅。", "Submitting records your feedback and may send related session logs for review.")).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
				text := v.Text
				ui.TextArea(c, &text).FillWidth().Height(100).Disabled(v.Busy).OnChange(func() { m.Store.EditSessionFeedback(text, v.Category) })
				ui.Row(c).Wrap().Gap(4).Children(func() {
					for _, category := range []string{"", "task-result", "instruction-following", "product-interaction", "service-stability", "resource-cost", "security-privacy-permission", "other"} {
						category := category
						label := category
						if label == "" {
							label = archiveText("无分类", "No category")
						}
						if category == v.Category {
							label = "● " + label
						}
						if ui.Button(c, label).Disabled(v.Busy).Clicked() {
							m.Store.EditSessionFeedback(text, category)
						}
					}
				})
				if v.Error != "" {
					ui.Text(c, v.Error).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, archiveText("取消", "Cancel")).Disabled(v.Busy).Clicked() {
						m.Store.CancelSessionFeedback()
					}
					if ui.Button(c, archiveText("提交会话反馈", "Submit session feedback")).Disabled(v.Busy).Clicked() {
						m.Store.SubmitSessionFeedback(context.Background())
					}
				})
			})
		})
	})
	if !open && !v.Busy {
		m.Store.CancelSessionFeedback()
	}
}
