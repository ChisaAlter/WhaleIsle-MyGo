package projects

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func materialsView(c *ui.Context, m Model) {
	d := m.Material
	ui.Text(c, d.ProjectID+" / "+d.Path).FontSize(fonttheme.InterfaceSize(12))
	if ui.Button(c, tr("返回（保留草稿）", "Back (keep draft)")).Disabled(m.Busy).Clicked() && m.OnCloseMaterial != nil {
		m.OnCloseMaterial()
	}
	if d.Path == "notes.md" {
		ui.Text(c, tr("仅编辑用户笔记；自动生成部分只读。", "Edit user notes only; generated section is read-only.")).FontSize(fonttheme.InterfaceSize(11))
		if d.Generated != "" {
			ui.Scroll(c).Height(100).Children(func() { ui.Text(c, d.Generated).FontSize(fonttheme.InterfaceSize(11)) })
		}
	} else {
		ui.Text(c, tr("项目持久偏好；当前用户指令优先。保存不会发送消息或启动任务。", "Persistent Project preferences; current user instructions take precedence. Saving does not send a message or start work.")).FontSize(fonttheme.InterfaceSize(11))
	}
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(fonttheme.InterfaceSize(12))
	}
	if d.Notice != "" {
		ui.Text(c, d.Notice).FontSize(fonttheme.InterfaceSize(11))
	}
	if !m.MaterialMutable {
		ui.Text(c, tr("项目不可编辑；归档项目请先恢复。", "Project is not editable; restore archived Projects first.")).FontSize(fonttheme.InterfaceSize(11))
	}
	if d.Loaded {
		ui.TextArea(c, &d.Draft).FillWidth().Height(220).Disabled(m.Busy || !m.MaterialMutable)
		if d.Dirty() {
			ui.Text(c, tr("有未保存草稿（最多 256000 UTF-8 字节）", "Unsaved draft (maximum 256000 UTF-8 bytes)")).FontSize(fonttheme.InterfaceSize(11))
		}
	}
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("保存", "Save")).Disabled(m.Busy || !m.MaterialMutable || !d.Dirty()).Clicked() && m.OnSaveMaterial != nil {
			m.OnSaveMaterial()
		}
		if ui.Button(c, tr("重新读取", "Reload")).Disabled(m.Busy || d.Dirty()).Clicked() && m.OnReloadMaterial != nil {
			m.OnReloadMaterial()
		}
		if ui.Button(c, tr("放弃草稿", "Discard draft")).Disabled(m.Busy || !d.Dirty()).Clicked() && m.OnDiscardMaterial != nil {
			m.OnDiscardMaterial()
		}
	})
}
