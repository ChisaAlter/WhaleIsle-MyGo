package settings

import (
	"regexp"
	"strings"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

type SkillCreateDraft struct {
	Name, Description, WhenToUse, Content, Root string
	Groups                                      []string
	ModelInvocable, UserInvocable               bool
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
	var validation *string
	bodyDialogFooterWidth(c, &open, providerCopy("添加技能", "Add skill"), 720, func() {
		nameLabel := kLabel(c, providerCopy("名称", "Name"), fg)
		validation = ui.Local(nameLabel, "skill-create-validation", func() string { return "" })
		nameInvalid := *validation != "" && !skillCreateName.MatchString(draft.Name)
		descInvalid := *validation != "" && strings.TrimSpace(draft.Description) == ""
		bodyInvalid := *validation != "" && strings.TrimSpace(draft.Content) == ""
		kInput(c, &draft.Name, coss.InputOptions{
			Label:       providerCopy("名称", "Name"),
			Placeholder: "review-code",
			Disabled:    create.Busy,
			Invalid:     nameInvalid,
		})
		kLabel(c, providerCopy("描述", "Description"), fg)
		kInput(c, &draft.Description, coss.InputOptions{
			Label:    providerCopy("描述", "Description"),
			Disabled: create.Busy,
			Invalid:  descInvalid,
		})
		kLabel(c, providerCopy("使用时机（可选）", "When to use (optional)"), fg)
		kTextarea(c, &draft.WhenToUse, coss.TextareaOptions{
			Label:    providerCopy("使用时机（可选）", "When to use (optional)"),
			MaxLines: 4,
			Disabled: create.Busy,
		}).Lines(2, 4)
		skillGroupPicker(c, "create-skill-groups", &draft.Groups, a.Skills, create.Busy, fg, muted)
		kLabel(c, providerCopy("技能正文", "Skill body"), fg)
		kTextarea(c, &draft.Content, coss.TextareaOptions{
			Label:    providerCopy("技能正文", "Skill body"),
			MaxLines: 14,
			Disabled: create.Busy,
			Invalid:  bodyInvalid,
		}).Lines(5, 14)
		kLabel(c, providerCopy("保存位置", "Save to"), fg)
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			for _, root := range []struct{ value, label string }{{"user-dsh", providerCopy("用户技能", "User skills")}, {"project-dsh", providerCopy("项目技能", "Project skills")}} {
				selected := draft.Root == root.value
				label := root.label
				if selected {
					label = "✓ " + label
				}
				if kChoiceButton(c, label, selected, create.Busy || root.value == "project-dsh" && a.ProjectCwd == "").Clicked() {
					draft.Root = root.value
				}
			}
		})
		if draft.Root == "project-dsh" {
			kHint(c, a.ProjectCwd)
		}
		kCheckbox(c, &draft.ModelInvocable, providerCopy("模型可调用", "Model invocable"), create.Busy)
		kCheckbox(c, &draft.UserInvocable, providerCopy("用户可调用", "User invocable"), create.Busy)
		if create.Error != "" {
			settingsFailureLine(c, "skill:create", providerCopy("没能创建这个技能，请重试。", "The skill could not be created. Try again."), create.Error)
		}
		if *validation != "" {
			kError(c, *validation)
		}
	}, func() {
		if kButtonDisabled(c, providerCopy("取消", "Cancel"), create.Busy).Clicked() && create.OnCancel != nil {
			create.OnCancel()
		}
		if kPrimary(c, providerCopy("创建技能", "Create skill"), create.Busy || create.OnSave == nil).Clicked() {
			*validation = skillCreateError(*draft)
			if *validation == "" {
				create.OnSave(*draft)
			}
		}
	}, ui.DialogOptions{DismissDisabled: create.Busy})
	if !open && !create.Busy && create.OnCancel != nil {
		create.OnCancel()
	}
}
