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
	WhenToUse               string
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
	Create       *SkillCreateAccess
	Hub          *SkillHubAccess
	OnOpenCreate func()
	Skills       []SkillRow
	SessionID    string
	Err          string
	LastAction   string
	OnRefresh    func()
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
			browse := skillsBrowseButton(c, i18n.T(i18n.KeySurfacesSkillsBrowseHub), muted)
			if a.HubQuery == nil || a.OnSearchHub == nil {
				browse = browse.Disabled(true).Opacity(0.5)
				browse.Tooltip(providerCopy("需要连接 Host", "Connect to a Host first"))
			} else if browse.Clicked() {
				if a.Hub != nil && a.Hub.OnOpen != nil {
					a.Hub.OnOpen()
				} else {
					*hubOpen = true
				}
			}
			if (a.OnCreate != nil || a.OnOpenCreate != nil) && bodyIconAction(c, providerCopy("添加技能", "Add skill"), "plus", a.Busy) {
				if a.OnOpenCreate != nil {
					a.OnOpenCreate()
				} else {
					*createOpen = true
				}
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
		ui.Text(c, settingsActionText(a.LastAction)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.DetailPreview != "" {
		ui.Text(c, a.DetailPreview).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.EditName != "" && a.OnCancelEdit != nil && ui.Button(c, providerCopy("取消编辑", "Cancel edit")).Disabled(a.Busy).Clicked() {
		a.OnCancelEdit()
	}
	if a.EditWritable && a.EditName != "" && a.EditDesc != nil && a.EditContent != nil && a.OnSaveUpdate != nil {
		open := true
		bodyDialog(c, &open, providerCopy("编辑技能", "Edit skill"), func() {
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsEditTitle)+": "+a.EditName).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
			ui.TextInput(c, a.EditDesc).FillWidth().Disabled(a.Busy).Placeholder(i18n.T(i18n.KeySettingsSkillsEditDesc))
			if a.EditWhenToUse != nil {
				ui.Text(c, providerCopy("使用条件（清空以移除）", "When to use (empty removes it)")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.TextArea(c, a.EditWhenToUse).FillWidth().Lines(2, 5).Disabled(a.Busy)
			}
			ui.TextArea(c, a.EditContent).FillWidth().Lines(5, 16).Disabled(a.Busy).Placeholder(i18n.T(i18n.KeySettingsSkillsEditContent))
			if a.EditModelInvocable != nil {
				ui.Checkbox(c, a.EditModelInvocable, providerCopy("模型可调用", "Model invocable")).Disabled(a.Busy)
			}
			if a.EditUserInvocable != nil {
				ui.Checkbox(c, a.EditUserInvocable, providerCopy("用户可调用", "User invocable")).Disabled(a.Busy)
			}
			ui.Text(c, providerCopy("仅保存会写入 Skill 文件；文本不会在此执行。", "Only Save writes the Skill file; text is not executed here.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
				ui.Text(c, providerCopy("草稿已保留；目标或版本已变化，请重新查看后再保存。", "Draft retained; inspect again before saving after a target, revision, or result change.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
		ui.Text(c, providerCopy("删除审阅", "Review deletion")).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, a.RemovalLabel).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, a.RemovalMessage).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		if a.OnCancelRemoval != nil && ui.Button(c, providerCopy("取消", "Cancel")).Disabled(a.RemovalBusy).Clicked() {
			a.OnCancelRemoval()
		}
		if a.OnConfirmRemoval != nil && ui.Button(c, providerCopy("确认永久删除", "Confirm permanent deletion")).Disabled(!a.RemovalReady || a.RemovalBusy).Clicked() {
			a.OnConfirmRemoval()
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Children(func() {
		if a.Query != nil {
			ui.TextInput(c, a.Query).Grow(1).MinWidth(180).Height(32).Placeholder(providerCopy("搜索名称、描述、分组或何时使用", "Search names, descriptions, groups or when to use"))
		}
		if a.SourceFilter != nil {
			choices := []struct{ value, label string }{
				{"", providerCopy("全部", "All")},
				{"user", providerCopy("用户", "User")},
				{"project", providerCopy("项目", "Project")},
				{"bundled", providerCopy("内置", "Bundled")},
				{"other", providerCopy("其他", "Other")},
			}
			options := make([]string, 0, len(choices))
			current := choices[0].label
			for _, choice := range choices {
				options = append(options, choice.label)
				if *a.SourceFilter == choice.value {
					current = choice.label
				}
			}
			if ui.Select(c, &current, options).Label("Skill source filter").Width(88).Height(32).Changed() {
				for _, choice := range choices {
					if current == choice.label {
						*a.SourceFilter = choice.value
						break
					}
				}
			}
		}
		hasFilters := a.Query != nil && strings.TrimSpace(*a.Query) != "" || a.SourceFilter != nil && *a.SourceFilter != "" && *a.SourceFilter != "all"
		if hasFilters && ui.Button(c, providerCopy("清除筛选", "Clear filters")).Height(36).Clicked() {
			if a.Query != nil {
				*a.Query = ""
			}
			if a.SourceFilter != nil {
				*a.SourceFilter = ""
			}
		}
	})
	filtering := a.Query != nil && strings.TrimSpace(*a.Query) != "" || a.SourceFilter != nil && *a.SourceFilter != "" && *a.SourceFilter != "all"
	visible := filterSkillRows(a.Skills, a.Query, a.SourceFilter)
	if len(visible) == 0 {
		if filtering {
			ui.Text(c, i18n.T(i18n.KeySettingsSkillsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		} else {
			ui.Column(c).FillWidth().Padding(64, 0).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z"/><path d="M18 16l.8 2.2L21 19l-2.2.8L18 22l-.8-2.2L15 19l2.2-.8z"/></svg>`))).Size(28, 28).TextColor(muted)
				ui.Text(c, providerCopy("还没有可用技能", "No skills available yet")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
				ui.Text(c, i18n.T(i18n.KeySettingsSkillsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				if a.HubQuery != nil && a.OnSearchHub != nil {
					browse := ui.Button(c, i18n.T(i18n.KeySurfacesSkillsBrowseHub)).Height(32).Padding(0, 14)
					if browse.Clicked() {
						if a.Hub != nil && a.Hub.OnOpen != nil {
							a.Hub.OnOpen()
						} else {
							*hubOpen = true
						}
					}
				}
			})
		}
	} else {
		groups := skillSections(visible)
		for _, group := range groups {
			if len(groups) == 1 && group.Name == "" {
				ui.Text(c, skillsGroupCountLabel(i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup), len(group.Skills))).
					FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
				for _, row := range group.Skills {
					skillCompactRow(c, a, row, fg, muted)
				}
				continue
			}
			label := group.Name
			if label == "" {
				label = i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup)
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
						ui.Icon(c, bodyIcon(icon)).Size(12, 12).TextColor(muted)
						ui.Text(c, skillsGroupCountLabel(label, len(group.Skills))).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
					})
					if disclosure.Clicked() {
						*open = !*open
					}
					enabled := group.AllEnabled
					groupDisabled := a.Busy || group.Writable == 0 || a.OnSetGroupInvocable == nil
					if settingsToggle(c, "Enable skill group · "+label, &enabled, groupDisabled).Clicked() && !groupDisabled && a.OnSetGroupInvocable != nil {
						a.OnSetGroupInvocable(group.Name, !enabled)
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
	skillCreateDialog(c, a, fg, muted)
	skillHubDialog(c, a, fg, muted)
	if *createOpen && a.Create == nil {
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
	if *hubOpen && a.Hub == nil && a.HubQuery != nil && a.OnSearchHub != nil {
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
						detail += skillStatusLabel(h.Status)
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

// skillStatusLabel localizes the hub status token; unknown values stay visible
// so a new Host status is not hidden.
func skillStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "installed":
		return pluginsText("已安装", "Installed")
	case "conflict":
		return pluginsText("冲突", "Conflict")
	case "not-installed":
		return pluginsText("未安装", "Not installed")
	case "available":
		return pluginsText("可安装", "Available")
	case "update-available":
		return pluginsText("有更新", "Update available")
	default:
		return status
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
	ui.Column(c.Key("skill-row/" + row.Source + "/" + row.Name)).FillWidth().MinWidth(0).Gap(4).MinHeight(64).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(10).Wrap().AlignItems(ui.Center).Children(func() {
			main := ui.Button(c, "").Label("Inspect skill · " + row.Name).MinHeight(48).Grow(1).MinWidth(180).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(!row.Writable || a.Busy || a.OnInspect == nil)
			main.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, settingsIcons["SkillOutline"]).Size(24, 24).Shrink(0).TextColor(muted)
					ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
						ui.Text(c, row.Name).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
						if row.Description != "" {
							ui.Text(c, row.Description).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Tooltip(row.Description)
						}
					})
				})
			})
			if main.Clicked() && a.OnInspect != nil {
				a.OnInspect(row.Name)
			}
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				skillSourceBadge(c, skillSourceLabel(row.Source), row.Directory, muted)
				if row.Directory != "" && a.OnOpenDirectory != nil && bodyIconAction(c, "Open directory · "+row.Name, "folder", a.Busy) {
					a.OnOpenDirectory(row.Name)
				}
				enabled := row.ModelInvocable
				switchDisabled := !row.Writable || a.Busy || a.OnSetModelInvocable == nil
				if settingsToggle(c, "Model invocation · "+row.Name, &enabled, switchDisabled).Clicked() && !switchDisabled && a.OnSetModelInvocable != nil {
					a.OnSetModelInvocable(row.Name, !enabled)
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

// skillsBrowseButton is the outline sm action with the magnifier glyph that the
// installed build puts at the right of the skills header.
func skillsBrowseButton(c *ui.Context, label string, muted ui.Color) ui.Element {
	button := ui.Button(c, "").Label(label).Height(28).Shrink(0).Padding(0, 10).Radius(8).Gap(6).
		Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .35)).Background(ui.Transparent).AlignItems(ui.Center)
	button.Children(func() {
		ui.Icon(c, bodyIcon("search")).Size(14, 14).TextColor(c.Theme().Text)
		ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Text)
	})
	return button
}

// skillSourceBadge is the small white chip after a skill row: the source name in
// 11px over a 0.5 hairline, the way the installed build tags bundled skills.
func skillSourceBadge(c *ui.Context, label, directory string, muted ui.Color) {
	if label == "" {
		return
	}
	badge := ui.Text(c, label).FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().TextMuted).
		Padding(1, 6).Radius(4).Shrink(0).Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .3)).
		Background(c.Theme().Surface)
	if directory != "" {
		badge.Tooltip(directory)
	}
}

// skillsGroupCountLabel orders the group label and its count the way the
// installed page does: "工作区与个人技能 0 项" but "0 workspace and personal skills".
func skillsGroupCountLabel(label string, count int) string {
	if i18n.Locale() == i18n.LocaleEN && label != "" && label == i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup) {
		return fmt.Sprint(count) + " " + strings.ToLower(label[:1]) + label[1:]
	}
	return label + " " + fmt.Sprint(count) + " " + providerCopy("项", "skills")
}
