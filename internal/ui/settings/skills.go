package settings

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// SkillRow is one skills/list entry for the dedicated settings page.
type SkillRow struct {
	Name                    string
	Source, Path, Directory string
	Description             string
	ModelInvocable          bool
	Groups                  []string
	Writable                bool
	UserInvocable           bool
	Error                   string
}

// SkillHubHit is one skillInventory/searchHub result row.
type SkillHubHit struct {
	SkillName   string
	Repo        string
	Path        string
	Description string
	Stars       float64
	Status      string
}

// SkillsAccess is the dedicated Skills settings page (session skills/list snapshot).
type SkillsAccess struct {
	Skills     []SkillRow
	SessionID  string
	Err        string
	LastAction string
	OnRefresh  func()
	// OnInsert inserts `/name ` into the composer (Electron ui-skill slash invoke).
	OnInsert func(name string)
	// OnSetModelInvocable toggles skillInventory/setInvocation modelInvocable.
	OnSetModelInvocable               func(name string, modelInvocable bool)
	OnDelete                          func(name string)
	Query, SourceFilter               *string
	OnOpenDirectory                   func(name string)
	RemovalID                         uint64
	RemovalLabel, RemovalMessage      string
	RemovalReady, RemovalBusy         bool
	OnConfirmRemoval, OnCancelRemoval func()
	OnCreate                          func(name string)
	OnInspect                         func(name string)
	CreateName                        *string
	DetailPreview                     string
	// Edit* hold skillInventory/get → skillInventory/update draft (writable skills only).
	EditGroups          *string
	Busy                bool
	Notice              string
	OnSetGroupInvocable func(group string, enabled bool)
	EditName            string
	EditDesc            *string
	EditContent         *string
	EditWritable        bool
	OnSaveUpdate        func()
	EditWhenToUse       *string
	EditModelInvocable  *bool
	EditUserInvocable   *bool
	EditCanSave         bool
	OnCancelEdit        func()

	TargetRoot   string
	ProjectCwd   string
	OnSelectRoot func(root string)
	HubQuery     *string
	HubHits      []SkillHubHit
	OnSearchHub  func(query string)
	OnInstallHub func(repo, path, skillName string)
}

func skillsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Skills == nil {
		return
	}
	a := m.Skills
	controls := ui.Column(c.Key("skills-surface")).FillWidth().MinWidth(0)
	createOpen := ui.Local(controls, "create-open", func() bool { return false })
	hubOpen := ui.Local(controls, "hub-open", func() bool { return false })
	ui.Row(c).FillWidth().MinWidth(0).Gap(12).Wrap().Children(func() {
		ui.Column(c).Grow(1).MinWidth(180).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsTitle)).FontSize(fonttheme.InterfaceSize(16)).Bold().TextColor(fg)
			ui.Text(c, providerCopy("查找并管理用户、当前项目与捆绑技能。", "Find and manage user, current-project, and bundled skills.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if a.HubQuery != nil && a.OnSearchHub != nil && ui.Button(c, providerCopy("搜索技能", "Find skills")).Height(30).Clicked() {
				*hubOpen = true
			}
			if a.OnCreate != nil && bodyIconAction(c, providerCopy("添加技能", "Add skill"), "plus", a.Busy) {
				*createOpen = true
			}
			if bodyIconAction(c, i18n.T(i18n.KeySettingsRefresh), "refresh", a.Busy || a.OnRefresh == nil) && a.OnRefresh != nil {
				a.OnRefresh()
			}
		})
	})
	if a.Notice != "" {
		ui.Text(c, a.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.Busy {
		ui.Text(c, "Saving skills…").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.DetailPreview != "" {
		ui.Text(c, a.DetailPreview).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.EditName != "" && a.OnCancelEdit != nil && ui.Button(c, "取消编辑 / Cancel edit").Disabled(a.Busy).Clicked() {
		a.OnCancelEdit()
	}
	if a.EditWritable && a.EditName != "" && a.EditDesc != nil && a.EditContent != nil && a.OnSaveUpdate != nil {
		open := true
		bodyDialog(c, &open, providerCopy("编辑技能", "Edit skill"), func() {
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsEditTitle)+": "+a.EditName).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
			ui.TextInput(c, a.EditDesc).FillWidth().Disabled(a.Busy).Placeholder(i18n.T(i18n.KeySettingsSkillsEditDesc))
			if a.EditWhenToUse != nil {
				ui.Text(c, "使用条件（清空以移除） / When to use (empty removes it)").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.TextArea(c, a.EditWhenToUse).FillWidth().Lines(2, 5).Disabled(a.Busy)
			}
			ui.TextArea(c, a.EditContent).FillWidth().Lines(5, 16).Disabled(a.Busy).Placeholder(i18n.T(i18n.KeySettingsSkillsEditContent))
			if a.EditModelInvocable != nil {
				ui.Checkbox(c, a.EditModelInvocable, "模型可调用 / Model invocable").Disabled(a.Busy)
			}
			if a.EditUserInvocable != nil {
				ui.Checkbox(c, a.EditUserInvocable, "用户可调用 / User invocable").Disabled(a.Busy)
			}
			ui.Text(c, "仅保存会写入 Skill 文件；文本不会在此执行。 / Only Save writes the Skill file; text is not executed here.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if a.EditGroups != nil {
				ui.Text(c, "Groups (comma-separated; leave empty to clear)").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.TextInput(c, a.EditGroups).FillWidth().Disabled(a.Busy)
				for _, group := range skillSections(a.Skills) {
					if group.Name == "" {
						continue
					}
					selected := false
					for _, tag := range strings.Split(*a.EditGroups, ",") {
						if strings.TrimSpace(tag) == group.Name {
							selected = true
						}
					}
					label := "+ " + group.Name
					if selected {
						label = "− " + group.Name
					}
					if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Disabled(a.Busy).Clicked() {
						tags := []string{}
						for _, tag := range strings.Split(*a.EditGroups, ",") {
							tag = strings.TrimSpace(tag)
							if tag != "" && tag != group.Name {
								tags = append(tags, tag)
							}
						}
						if !selected {
							tags = append(tags, group.Name)
						}
						*a.EditGroups = strings.Join(tags, ", ")
					}
				}

			}
			if !a.EditCanSave && !a.Busy {
				ui.Text(c, "草稿已保留；目标或版本已变化，请重新查看后再保存。 / Draft retained; inspect again before saving after a target, revision, or result change.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			if ui.Button(c, i18n.T(i18n.KeySettingsSkillsSaveUpdate)).FontSize(fonttheme.InterfaceSize(11)).Disabled(a.Busy || !a.EditCanSave).Clicked() {
				a.OnSaveUpdate()
			}
		})
		if !open && !a.Busy && a.OnCancelEdit != nil {
			a.OnCancelEdit()
		}
	}
	if a.RemovalID != 0 {
		ui.Text(c, "删除审阅 / Review deletion").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, a.RemovalLabel).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, a.RemovalMessage).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		if a.OnCancelRemoval != nil && ui.Button(c, "取消 / Cancel").Disabled(a.RemovalBusy).Clicked() {
			a.OnCancelRemoval()
		}
		if a.OnConfirmRemoval != nil && ui.Button(c, "确认永久删除 / Confirm permanent deletion").Disabled(!a.RemovalReady || a.RemovalBusy).Clicked() {
			a.OnConfirmRemoval()
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Children(func() {
		if a.Query != nil {
			ui.TextInput(c, a.Query).Grow(1).MinWidth(180).Height(36).Placeholder(providerCopy("搜索技能", "Search skills"))
		}
		if a.SourceFilter != nil {
			all := providerCopy("全部来源", "All sources")
			options := []string{all}
			seen := map[string]bool{}
			current := *a.SourceFilter
			if current == "" {
				current = all
			}
			for _, row := range a.Skills {
				if row.Source != "" && !seen[row.Source] {
					seen[row.Source] = true
					options = append(options, row.Source)
				}
			}
			if ui.Select(c, &current, options).Label("Skill source filter").Width(130).Changed() {
				if current == all {
					*a.SourceFilter = ""
				} else {
					*a.SourceFilter = current
				}
			}
		}
	})
	visible := filterSkillRows(a.Skills, a.Query, a.SourceFilter)
	ui.Text(c, fmt.Sprintf(providerCopy("%d 个技能", "%d skills"), len(visible))).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if len(visible) == 0 {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	} else {
		groups := skillSections(visible)
		for _, group := range groups {
			if len(groups) == 1 && group.Name == "" {
				for _, row := range group.Skills {
					skillCompactRow(c, a, row, fg, muted)
				}
				continue
			}
			label := group.Name
			if label == "" {
				label = providerCopy("未分组", "Ungrouped")
			}
			section := ui.Column(c.Key("skill-group/" + group.Name)).FillWidth().MinWidth(0).Gap(4)
			open := ui.Local(section, "expanded", func() bool { return true })
			section.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					disclosure := ui.Button(c, "").Label("Skill group · "+label).Grow(1).MinWidth(0).BorderWidth(0).Background(ui.Transparent).Padding(4, 0).Justify(ui.Start).AlignItems(ui.Center).Gap(6)
					disclosure.Children(func() {
						icon := "right"
						if *open {
							icon = "down"
						}
						ui.Icon(c, bodyIcon(icon)).Size(14, 14).TextColor(muted)
						ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
						ui.Text(c, fmt.Sprint(len(group.Skills))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					})
					if disclosure.Clicked() {
						*open = !*open
					}
					enabled := group.AllEnabled
					if ui.Switch(c, &enabled).Label("Enable skill group · "+label).Disabled(a.Busy || group.Writable == 0 || a.OnSetGroupInvocable == nil).Changed() && a.OnSetGroupInvocable != nil {
						a.OnSetGroupInvocable(group.Name, enabled)
					}
				})
				if *open {
					for _, row := range group.Skills {
						skillCompactRow(c, a, row, fg, muted)
					}
				}
			})
		}
	}
	if *createOpen {
		bodyDialog(c, createOpen, providerCopy("添加技能", "Add skill"), func() {
			if a.OnSelectRoot != nil {
				ui.Text(c, "Create / install target").FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
				ui.Row(c).Gap(8).Wrap().Children(func() {
					userLabel, projectLabel := "User skills", "Project skills"
					if a.TargetRoot == "project-dsh" {
						projectLabel = "✓ " + projectLabel
					} else {
						userLabel = "✓ " + userLabel
					}
					if ui.Button(c, userLabel).Clicked() {
						a.OnSelectRoot("user-dsh")
					}
					if a.ProjectCwd != "" && ui.Button(c, projectLabel).Clicked() {
						a.OnSelectRoot("project-dsh")
					}
				})
				if a.TargetRoot == "project-dsh" && a.ProjectCwd != "" {
					ui.Text(c, "Project directory: "+a.ProjectCwd).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				} else if a.ProjectCwd == "" {
					ui.Text(c, "Select a workspace directory before creating or installing project skills.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			}
			if a.CreateName != nil && a.OnCreate != nil {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.TextInput(c, a.CreateName).Grow(1).Placeholder(i18n.T(i18n.KeyComposerSkillsCreateName))
					if ui.Button(c, i18n.T(i18n.KeyComposerSkillsCreate)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
						name := strings.TrimSpace(*a.CreateName)
						if name != "" {
							a.OnCreate(name)
							*a.CreateName = ""
						}
					}
				})
			}
		})
	}
	if *hubOpen && a.HubQuery != nil && a.OnSearchHub != nil {
		bodyDialog(c, hubOpen, providerCopy("搜索技能", "Find skills"), func() {
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsHubTitle)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg).Padding(12, 0, 0, 0)
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsHubHint)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, a.HubQuery).Grow(1).Placeholder(i18n.T(i18n.KeySettingsSkillsHubPlaceholder))
				if ui.Button(c, i18n.T(i18n.KeySettingsSkillsHubSearch)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
					a.OnSearchHub(*a.HubQuery)
				}
			})
			for _, hit := range a.HubHits {
				h := hit
				label := h.SkillName
				if label == "" {
					label = h.Path
				}
				bodyRosterRow(c, emptyDash(label), func() {
					detail := h.Repo
					if h.Description != "" {
						if detail != "" {
							detail += " · "
						}
						detail += h.Description
					}
					if h.Status != "" {
						if detail != "" {
							detail += " · "
						}
						detail += h.Status
					}
					if detail != "" {
						ui.Text(c, detail).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					}
					if a.OnInstallHub != nil && h.Repo != "" && h.Path != "" && h.SkillName != "" && h.Status != "installed" && h.Status != "conflict" {
						if ui.Button(c, i18n.T(i18n.KeySettingsSkillsHubInstall)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
							a.OnInstallHub(h.Repo, h.Path, h.SkillName)
						}
					}
				}, fg, muted)
			}
		})

	}
}

// A skill belongs to each of its tags. The ungrouped section is always last.
type skillSection struct {
	Name       string
	Skills     []SkillRow
	Writable   int
	AllEnabled bool
}

func skillSections(rows []SkillRow) []skillSection {
	sections := []skillSection{}
	indexes := map[string]int{}
	ungrouped := []SkillRow{}
	for _, row := range rows {
		seen := map[string]bool{}
		added := false
		for _, name := range row.Groups {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			added = true
			idx, ok := indexes[name]
			if !ok {
				idx = len(sections)
				indexes[name] = idx
				sections = append(sections, skillSection{Name: name, AllEnabled: true})
			}
			sections[idx].Skills = append(sections[idx].Skills, row)
		}
		if !added {
			ungrouped = append(ungrouped, row)
		}
	}
	if len(ungrouped) > 0 {
		sections = append(sections, skillSection{Skills: ungrouped, AllEnabled: true})
	}
	for i := range sections {
		for _, row := range sections[i].Skills {
			if row.Writable {
				sections[i].Writable++
				if !row.ModelInvocable {
					sections[i].AllEnabled = false
				}
			}
		}
	}
	return sections
}

func skillCompactRow(c *ui.Context, a *SkillsAccess, row SkillRow, fg, muted ui.Color) {
	ui.Column(c.Key("skill-row/"+row.Source+"/"+row.Name)).FillWidth().MinWidth(0).Gap(4).Padding(8, 0).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(10).Wrap().AlignItems(ui.Center).Children(func() {
			main := ui.Button(c, "").Label("Inspect skill · " + row.Name).MinHeight(48).Grow(1).MinWidth(180).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(!row.Writable || a.Busy || a.OnInspect == nil)
			main.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, settingsIcons["SkillOutline"]).Size(16, 16).Shrink(0).TextColor(muted)
					ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
						ui.Text(c, row.Name).FontSize(fonttheme.InterfaceSize(14)).TextColor(fg)
						ui.Text(c, row.Description).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Tooltip(row.Description)
					})
				})
			})
			if main.Clicked() && a.OnInspect != nil {
				a.OnInspect(row.Name)
			}
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, row.Source).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Tooltip(row.Directory)
				if row.Directory != "" && a.OnOpenDirectory != nil && bodyIconAction(c, "Open directory · "+row.Name, "folder", a.Busy) {
					a.OnOpenDirectory(row.Name)
				}
				enabled := row.ModelInvocable
				if ui.Switch(c, &enabled).Label("Model invocation · "+row.Name).Disabled(!row.Writable || a.Busy || a.OnSetModelInvocable == nil).Changed() && a.OnSetModelInvocable != nil {
					a.OnSetModelInvocable(row.Name, enabled)
				}
				if row.Writable && a.OnDelete != nil && bodyIconAction(c, "Delete skill · "+row.Name, "trash", a.Busy || a.RemovalBusy) {
					a.OnDelete(row.Name)
				}
			})
		})
		if row.Error != "" {
			ui.Text(c, row.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
	})
}
