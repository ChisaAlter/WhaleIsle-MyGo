package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// VisualReplyView sits outside ToolCardView's disclosure. HTML is always plain
// native text here. Preview pixels come only from the gated isolated renderer;
// the chat card itself never loads a webview, URL or script.
func VisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, fg, muted, border, field ui.Color) {
	id := p.File.AttachmentID
	state := m.VisualReply
	active := state.SessionID == m.Snapshot.SelectedID && state.AttachmentID == id
	ui.Column(c.Key("published-html-"+id)).FillWidth().Gap(6).Padding(10).Background(field).Border(1, border).Radius(6).Children(func() {
		ui.Text(c, p.Title).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "已发布 HTML / Published HTML").FontSize(11).TextColor(muted)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, "安全图片预览 / Safe image preview").Disabled(m.OnPreviewVisualReply == nil || (active && (state.Loading || state.PreviewLoading))).Clicked() {
				m.OnPreviewVisualReply(id)
			}
			if ui.Button(c, "查看完整源码 / View source").Disabled(m.OnReadVisualReply == nil || (active && state.Loading)).Clicked() {
				m.OnReadVisualReply(id)
			}
			if active && state.Ready {
				if ui.Button(c, "另存 HTML… / Save as…").Disabled(m.OnSaveVisualReply == nil).Clicked() {
					m.OnSaveVisualReply(id)
				}
			}
			if active && m.OnCloseVisualReply != nil {
				if ui.Button(c, "关闭 / Close").Clicked() {
					m.OnCloseVisualReply()
				}
			}
		})
		if !active {
			return
		}
		if state.Loading {
			ui.Text(c, "正在授权读取… / Reading published page…").TextColor(muted)
		}
		if state.Error != "" {
			ui.Text(c, state.Error).TextColor(muted)
		}
		if state.PreviewLoading {
			ui.Text(c, "正在检查隔离环境并生成静态预览… / Checking isolation and rendering a static preview…").TextColor(muted)
		}
		if state.PreviewError != "" {
			ui.Text(c, "安全预览不可用，仍可查看源码或另存 / Safe preview unavailable: "+state.PreviewError).TextColor(muted)
		}
		if m.VisualReplyBitmap != nil && state.PreviewID != 0 {
			ui.Image(c, m.VisualReplyBitmap).FillWidth().AspectRatio(float32(state.PreviewWidth) / float32(state.PreviewCapturedHeight)).Fit(ui.Contain)
			ui.Text(c, "静态 PNG，交互未启用 / Static PNG; interactions are not enabled").FontSize(11).TextColor(muted)
			if state.PreviewCapturedHeight < state.PreviewContentHeight {
				ui.Text(c, "页面超过4000像素，预览未覆盖底部 / Preview is capped at4000 pixels; the bottom is not shown.").FontSize(11).TextColor(muted)
			}
		}
		if state.Ready {
			source := state.HTML
			ui.TextArea(c.Key("html-source"), &source).ReadOnly(true).FillWidth().Lines(8, 20).FontSize(12).TextColor(fg)
			ui.Text(c, "源码视图为纯文本；安全预览仅显示 PNG。另存后在外部浏览器打开可能运行脚本或访问网络。 / Source is plain text; safe preview is PNG only. Opening a saved page externally may run scripts or access the network.").FontSize(11).TextColor(muted)
		}
	})
}
