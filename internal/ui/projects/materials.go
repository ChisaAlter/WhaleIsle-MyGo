package projects

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func materialLimit(path string) int {
	if path == "charter.md" {
		return 16000
	}
	return 256000
}

func materialLabel(path string) string {
	switch path {
	case "charter.md":
		return tr("项目规则", "Project rules")
	case "preferences.md":
		return tr("偏好", "Preferences")
	case "notes.md":
		return tr("笔记", "Notes")
	}
	return path
}

func materialsView(c *ui.Context, m Model) {
	d := m.Material
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		if ui.Button(c, tr("工作成果", "Work results")).Disabled(m.Busy).Clicked() && m.OnMaterialCollection != nil {
			m.OnMaterialCollection(d.ProjectID)
		}
		for _, path := range []string{"charter.md", "preferences.md", "notes.md"} {
			path := path
			if ui.Button(c, materialLabel(path)).Disabled(m.Busy || path == d.Path).Clicked() && m.OnMaterial != nil {
				m.OnMaterial(d.ProjectID, path)
			}
		}
	})
	ui.Text(c, d.ProjectID+" / "+d.Path).FontSize(theme.InterfaceSize(12)).Selectable()
	switch d.Path {
	case "notes.md":
		ui.Text(c, tr("编辑用户笔记；自动生成的记录为只读。", "Edit user notes; generated records are read-only.")).FontSize(theme.InterfaceSize(13))
	case "charter.md":
		ui.Text(c, tr("记录此项目必须遵守的简短规则。", "Record the short rules this Project must follow.")).FontSize(theme.InterfaceSize(13))
	default:
		ui.Text(c, tr("长期习惯与偏好，例如回复风格和常用命令。保存不会发送消息或启动工作。", "Persistent preferences such as response style and common commands. Saving does not send a message or start work.")).FontSize(theme.InterfaceSize(13))
	}
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(theme.InterfaceSize(12))
	}
	if d.Notice != "" {
		ui.Text(c, d.Notice).FontSize(theme.InterfaceSize(12))
	}
	if !m.MaterialMutable {
		ui.Text(c, tr("此项目为只读；恢复已归档项目后可继续编辑。", "This Project is read-only; restore it before editing.")).FontSize(theme.InterfaceSize(12))
	}
	if d.Loaded {
		ui.TextArea(c.Key("material/"+d.ProjectID+"/"+d.Path), &d.Draft).Label(tr("项目材料编辑器：", "Project material editor: ") + materialLabel(d.Path)).FillWidth().MinWidth(0).Height(240).FontSize(theme.InterfaceSize(14)).ReadOnly(!m.MaterialMutable).Disabled(m.Busy)
		if d.Dirty() {
			ui.Text(c, tr("有未保存的更改", "Unsaved changes")).FontSize(theme.InterfaceSize(12))
		}
		ui.Text(c, fmt.Sprintf("%d / %d UTF-8 bytes", len(d.Draft), materialLimit(d.Path))).FontSize(theme.InterfaceSize(12))
	} else if !m.Busy && d.Error == "" {
		ui.Text(c, tr("材料尚未载入，请重新读取。", "Material has not loaded; reload to retry.")).FontSize(theme.InterfaceSize(13))
	}
	if !m.modalBody {
		materialFooter(c, m)
	}
	if d.Path == "notes.md" && d.Generated != "" {
		ui.Text(c, tr("自动生成的记录", "Generated records")).FontSize(theme.InterfaceSize(13)).Bold()
		ui.Text(c, d.Generated).FontSize(theme.InterfaceSize(13)).Selectable()
	}
}

func materialFooter(c *ui.Context, m Model) {
	d := m.Material
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		if ui.PrimaryButton(c, tr("保存", "Save")).Disabled(m.Busy || !m.MaterialMutable || !d.Dirty() || len(d.Draft) > materialLimit(d.Path)).Clicked() && m.OnSaveMaterial != nil {
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
