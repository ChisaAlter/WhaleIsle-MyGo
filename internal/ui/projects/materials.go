package projects

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	case "research-hosts.txt":
		return tr("研究站点", "Research sites")
	}
	return path
}

func materialsView(c *ui.Context, m Model) {
	d := m.Material
	root := ui.Column(c.Key("project-material:" + d.ProjectID + "/" + d.Path)).FillWidth().MinWidth(0).Gap(12)
	root.Children(func() { materialBody(c, m, root) })
}

func materialBody(c *ui.Context, m Model, root ui.Element) {
	d := m.Material
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		if ui.Button(c, tr("工作成果", "Work results")).Disabled(m.Busy).Clicked() && m.OnMaterialCollection != nil {
			m.OnMaterialCollection(d.ProjectID)
		}
		for _, path := range []string{"charter.md", "preferences.md", "notes.md", "research-hosts.txt"} {
			path := path
			if ui.Button(c, materialLabel(path)).Disabled(m.Busy || path == d.Path).Clicked() && m.OnMaterial != nil {
				m.OnMaterial(d.ProjectID, path)
			}
		}
	})
	switch d.Path {
	case "notes.md":
		ui.Text(c, tr("编辑用户笔记；自动生成的记录为只读。", "Edit user notes; generated records are read-only.")).FontSize(theme.InterfaceSize(13))
	case "charter.md":
		ui.Text(c, tr("记录此项目必须遵守的简短规则。", "Record the short rules this Project must follow.")).FontSize(theme.InterfaceSize(13))
	case "research-hosts.txt":
		ui.Text(c, tr("限制研究成员只能读取这些站点：每行一个域名（如 example.com，同时包含它的子域），用 # 写注释。留空表示不限制。", "Limit research members to these sites: one domain per line (example.com also covers its subdomains), # starts a comment. Leave it empty for no limit.")).FontSize(theme.InterfaceSize(13))
	default:
		ui.Text(c, tr("长期习惯与偏好，例如回复风格和常用命令。保存不会发送消息或启动工作。", "Persistent preferences such as response style and common commands. Saving does not send a message or start work.")).FontSize(theme.InterfaceSize(13))
	}
	notice := ui.Local(root, "material-error:"+d.ProjectID+"/"+d.Path, func() string { return "" })
	if *notice != d.Error {
		*notice = d.Error
		if d.Error == "" {
			c.CloseToast("project-material-error")
		} else {
			c.AddToast(ui.Toast{ID: "project-material-error", Description: d.Error, Type: "error", Timeout: -1})
		}
	}
	if !m.MaterialMutable {
		ui.Text(c, tr("此项目为只读；恢复已归档项目后可继续编辑。", "This Project is read-only; restore it before editing.")).FontSize(theme.InterfaceSize(12))
	}
	if d.Loaded && d.Path == "charter.md" {
		ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
			examples := []struct{ label, line string }{
				{tr("测试", "Tests"), tr("- 新增或修改的功能必须附带测试", "- New or changed features must include tests")},
				{tr("范围", "Scope"), tr("- 只修改与需求相关的文件，不顺手重构无关代码", "- Change only files related to the requirement; no unrelated refactoring")},
				{tr("验证", "Verify"), tr("- 交付前运行项目测试，并在交付中报告结果", "- Run the project tests before delivery and report the result")},
				{tr("语言", "Language"), tr("- 界面文案使用中文，代码注释使用英文", "- Keep user-facing text in the product language; write code comments in English")},
			}
			for _, ex := range examples {
				present := false
				for _, line := range strings.Split(d.Draft, "\n") {
					if strings.TrimSpace(line) == ex.line {
						present = true
					}
				}
				label := ex.label
				if present {
					label = "✓ " + label
				}
				if ui.Button(c, label).Disabled(m.Busy || !m.MaterialMutable).Clicked() {
					if present {
						lines := []string{}
						for _, line := range strings.Split(d.Draft, "\n") {
							if strings.TrimSpace(line) != ex.line {
								lines = append(lines, line)
							}
						}
						d.Draft = strings.Join(lines, "\n")
					} else {
						d.Draft = strings.TrimRight(d.Draft, " \t\r\n")
						if d.Draft != "" {
							d.Draft += "\n"
						}
						d.Draft += ex.line
					}
					d.Notice = ""
				}
			}
		})
	}
	if d.Loaded {
		editor := ui.TextArea(c.Key("material/"+d.ProjectID+"/"+d.Path), &d.Draft).Label(tr("项目材料编辑器：", "Project material editor: ") + materialLabel(d.Path)).FillWidth().MinWidth(0).Height(240).FontSize(theme.InterfaceSize(14)).ReadOnly(!m.MaterialMutable).Disabled(m.Busy)
		if d.Path == "research-hosts.txt" {
			editor.Placeholder(tr("example.com\ndocs.example.org  # 注释", "example.com\ndocs.example.org  # comment"))
		}
		if d.Path == "charter.md" {
			ui.Progress(c, float64(len(d.Draft))/16000).FillWidth().Height(3)
			ui.Text(c, fmt.Sprintf(tr("已使用 %.1f / 16 KB", "%.1f / 16 KB used"), float64(len(d.Draft))/1000)).FontSize(theme.InterfaceSize(12))
		} else if len(d.Draft) > materialLimit(d.Path) {
			ui.Text(c, tr("内容超出长度限制，请缩短后保存。", "This material exceeds the size limit. Shorten it before saving.")).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger)
		}
	} else if !m.Busy && d.Error == "" {
		ui.Text(c, tr("材料尚未载入，请重新读取。", "Material has not loaded; reload to retry.")).FontSize(theme.InterfaceSize(13))
	}
	if !m.modalBody {
		materialFooter(c, m)
	}
	if d.Path == "notes.md" {
		generated := ui.Local(root, "generated-notes:"+d.ProjectID, func() bool { return false })
		widgets.DisclosureRow(c, widgets.DisclosureRowOptions{Title: tr("自动生成的记录", "Generated records"), Open: generated, Body: func() { projectMarkdown(c, m, d.Generated) }})
		decisions := ui.Local(root, "decision-log:"+d.ProjectID, func() bool { return false })
		widgets.DisclosureRow(c, widgets.DisclosureRowOptions{Title: tr("决策日志", "Decision log"), Open: decisions, Body: func() {
			if d.DecisionError != "" {
				ui.Text(c, d.DecisionError).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger)
			}
			text := strings.TrimSpace(d.Decisions)
			if at := strings.Index(text, "\n"); strings.HasPrefix(text, "# ") && at >= 0 {
				text = strings.TrimSpace(text[at+1:])
			}
			lines := strings.Split(text, "\n")
			clean := lines[:0]
			for _, line := range lines {
				if !strings.HasPrefix(strings.TrimSpace(line), "<!-- decision:") {
					clean = append(clean, line)
				}
			}
			text = strings.Join(clean, "\n")
			if strings.TrimSpace(text) == "" {
				ui.Text(c, tr("暂无决策记录。", "No decisions recorded yet.")).FontSize(theme.InterfaceSize(13))
			} else {
				projectMarkdown(c, m, text)
			}
		}})
	}
}

func materialFooter(c *ui.Context, m Model) {
	d := m.Material
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		label := tr("保存", "Save")
		if d.Notice != "" && !d.Dirty() {
			label = tr("已保存", "Saved")
		}
		if ui.PrimaryButton(c, label).Tooltip(d.Error).Disabled(m.Busy || !m.MaterialMutable || !d.Dirty() || len(d.Draft) > materialLimit(d.Path)).Clicked() && m.OnSaveMaterial != nil {
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
