package settings

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

type SkillCreateDraft struct {
	Name, Description, WhenToUse, Groups, Content, Root string
	ModelInvocable, UserInvocable                       bool
}

type SkillCreateAccess struct {
	Open, Busy bool
	Draft      *SkillCreateDraft
	Error      string
	OnSave     func(SkillCreateDraft)
	OnCancel   func()
}

var skillCreateName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func skillCreateError(d SkillCreateDraft) string {
	if !skillCreateName.MatchString(d.Name) {
		return providerCopy("名称需使用小写字母、数字和连字符，例如 review-code。", "Use a lowercase kebab-case name, such as review-code.")
	}
	if strings.TrimSpace(d.Description) == "" {
		return providerCopy("请输入技能描述。", "Enter a skill description.")
	}
	if strings.TrimSpace(d.Content) == "" {
		return providerCopy("请输入技能正文。", "Enter the skill body.")
	}
	return ""
}

func skillCreateDialog(c *ui.Context, a *SkillsAccess, fg, muted ui.Color) {
	create := a.Create
	if create == nil || !create.Open || create.Draft == nil {
		return
	}
	draft := create.Draft
	open := true
	bodyDialog(c, &open, providerCopy("添加技能", "Add skill"), func() {
		ui.Text(c, providerCopy("名称", "Name")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.TextInput(c, &draft.Name).FillWidth().Disabled(create.Busy).Placeholder("review-code")
		ui.Text(c, providerCopy("描述", "Description")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.TextInput(c, &draft.Description).FillWidth().Disabled(create.Busy)
		ui.Text(c, providerCopy("使用时机（可选）", "When to use (optional)")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.TextArea(c, &draft.WhenToUse).FillWidth().Lines(2, 4).Disabled(create.Busy)
		ui.Text(c, providerCopy("分组（以逗号分隔）", "Groups (comma-separated)")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.TextInput(c, &draft.Groups).FillWidth().Disabled(create.Busy)
		for _, group := range skillSections(a.Skills) {
			if group.Name == "" {
				continue
			}
			selected := false
			for _, name := range strings.Split(draft.Groups, ",") {
				selected = selected || strings.TrimSpace(name) == group.Name
			}
			label := "+ " + group.Name
			if selected {
				label = "✓ " + group.Name
			}
			if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Disabled(create.Busy).Clicked() {
				var groups []string
				for _, name := range strings.Split(draft.Groups, ",") {
					name = strings.TrimSpace(name)
					if name != "" && name != group.Name {
						groups = append(groups, name)
					}
				}
				if !selected {
					groups = append(groups, group.Name)
				}
				draft.Groups = strings.Join(groups, ", ")
			}
		}
		ui.Text(c, providerCopy("技能正文", "Skill body")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.TextArea(c, &draft.Content).FillWidth().Lines(5, 14).Disabled(create.Busy)
		ui.Text(c, providerCopy("保存位置", "Save to")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			for _, root := range []struct{ value, label string }{{"user-dsh", providerCopy("用户技能", "User skills")}, {"project-dsh", providerCopy("项目技能", "Project skills")}} {
				label := root.label
				if draft.Root == root.value {
					label = "✓ " + label
				}
				if ui.Button(c, label).Disabled(create.Busy || root.value == "project-dsh" && a.ProjectCwd == "").Clicked() {
					draft.Root = root.value
				}
			}
		})
		if draft.Root == "project-dsh" {
			ui.Text(c, a.ProjectCwd).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Checkbox(c, &draft.ModelInvocable, providerCopy("模型可调用", "Model invocable")).Disabled(create.Busy)
		ui.Checkbox(c, &draft.UserInvocable, providerCopy("用户可调用", "User invocable")).Disabled(create.Busy)
		if create.Error != "" {
			ui.Text(c, create.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		validation := ui.Local(c, "skill-create-validation", func() string { return "" })
		if *validation != "" {
			ui.Text(c, *validation).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(create.Busy).Clicked() && create.OnCancel != nil {
				create.OnCancel()
			}
			if ui.Button(c, providerCopy("创建技能", "Create skill")).Disabled(create.Busy || create.OnSave == nil).Clicked() {
				*validation = skillCreateError(*draft)
				if *validation == "" {
					create.OnSave(*draft)
				}
			}
		})
	})
	if !open && !create.Busy && create.OnCancel != nil {
		create.OnCancel()
	}
}
