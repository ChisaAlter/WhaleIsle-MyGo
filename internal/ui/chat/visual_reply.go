package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// VisualReplyView sits outside ToolCardView's disclosure. HTML is always plain
// native text here: no webview, Markdown interpretation, URL loading or scripts.
func VisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, fg, muted, border, field ui.Color) {
	id := p.File.AttachmentID
	state := m.VisualReply
	active := state.SessionID == m.Snapshot.SelectedID && state.AttachmentID == id
	ui.Column(c.Key("published-html-"+id)).FillWidth().Gap(6).Padding(10).Background(field).Border(1, border).Radius(6).Children(func() {
		ui.Text(c, p.Title).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "已发布 HTML · 仅源码 / Published HTML · source only").FontSize(11).TextColor(muted)
		ui.Row(c).Gap(6).Children(func() {
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
		if state.Ready {
			source := state.HTML
			ui.TextArea(c.Key("html-source"), &source).ReadOnly(true).FillWidth().Lines(8, 20).FontSize(12).TextColor(fg)
			ui.Text(c, "源码未执行；另存后在外部浏览器打开可能运行脚本或访问网络。 / Opening a saved page externally may run scripts or access the network.").FontSize(11).TextColor(muted)
		}
	})
}
