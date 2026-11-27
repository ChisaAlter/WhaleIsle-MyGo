package settings

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

var skillsEmptySparkleIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z"/><path d="M18 16l.8 2.2L21 19l-2.2.8L18 22l-.8-2.2L15 19l2.2-.8z"/></svg>`))

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
	EditGroups                 *[]string
	Busy                       bool
	Notice                     string
	OnSetGroupInvocable        func(group string, enabled bool)
	OnSetVisibleGroupInvocable func(members []SkillRow, enabled bool)
	EditName                   string
	EditDesc                   *string
	EditContent                *string
	EditWritable               bool
	OnSaveUpdate               func()
	EditWhenToUse              *string
	EditModelInvocable         *bool
	EditUserInvocable          *bool
	EditCanSave                bool
	OnCancelEdit               func()
	OnReloadEdit               func()

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
	controls := ui.Column(c.Key("skills-surface")).Absolute().Size(0, 0).PassThrough()
	createOpen := ui.Local(controls, "create-open", func() bool { return false })
	hubOpen := ui.Local(controls, "hub-open", func() bool { return false })
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Wrap().Children(func() {
			ui.Column(c).Grow(1).MinWidth(180).Gap(0).Children(func() {
				kBody(c, providerCopy("查找并管理用户、当前项目与捆绑技能。", "Find and manage user, current-project, and bundled skills.")).TextColor(muted)
			})
			ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				browseDisabled := a.HubQuery == nil || a.OnSearchHub == nil
				browse := skillsBrowseButton(c, i18n.T(i18n.KeySurfacesSkillsBrowseHub), browseDisabled)
				if browseDisabled {
					browse.Tooltip(providerCopy("需要连接 Host", "Connect to a Host first"))
				} else if browse.Clicked() {
					if a.Hub != nil && a.Hub.OnOpen != nil {
						a.Hub.OnOpen()
					} else {
						*hubOpen = true
					}
				}
				if (a.OnCreate != nil || a.OnOpenCreate != nil) && mcpHeaderIconButton(c, providerCopy("添加技能", "Add skill"), "plus", a.Busy) {
					if a.OnOpenCreate != nil {
						a.OnOpenCreate()
					} else {
						*createOpen = true
					}
				}
				if mcpHeaderIconButton(c, i18n.T(i18n.KeySettingsRefresh), "refresh", a.Busy || a.OnRefresh == nil) && a.OnRefresh != nil {
					a.OnRefresh()
				}
			})
		})
		settingsNotice(c, "skills:127", a.Notice != "", func() string { return a.Notice })
		settingsAlert(c, "skills:130", a.Err != "", providerCopy("没能读取技能列表，请重试。", "Skills could not be read. Try again."), a.Err)
		settingsNotice(c, "skills:133", a.LastAction != "", func() string { return settingsActionText(a.LastAction) })
		if a.DetailPreview != "" && !a.EditWritable {
			kHint(c, a.DetailPreview)
		}
		if a.EditWritable && a.EditName != "" && a.EditDesc != nil && a.EditContent != nil && a.OnSaveUpdate != nil {
			open := true
			bodyDialogFooterWidth(c, &open, providerCopy("编辑技能", "Edit skill"), 720, func() {
				heading := kHeading(c, i18n.T(i18n.KeySettingsSkillsEditTitle)+": "+a.EditName).Key("edit-heading/" + a.EditName)
				recovery := ui.Local(heading, "recovery-visible", func() bool { return false })
				if !a.Busy {
					*recovery = !a.EditCanSave
				}
				if actionText := settingsActionText(a.LastAction); actionText != "" {
					ui.Text(c, actionText).FillWidth().Selectable().TextColor(fg)
				}
				if a.DetailPreview != "" {
					ui.Text(c, a.DetailPreview).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
				kInput(c, a.EditDesc, coss.InputOptions{
					Label:       i18n.T(i18n.KeySettingsSkillsEditDesc),
					Placeholder: i18n.T(i18n.KeySettingsSkillsEditDesc),
					Disabled:    a.Busy,
				})
				if a.EditWhenToUse != nil {
					kHint(c, providerCopy("使用条件（清空以移除）", "When to use (empty removes it)"))
					kTextarea(c, a.EditWhenToUse, coss.TextareaOptions{
						Label:    providerCopy("使用条件", "When to use"),
						MaxLines: 5,
						Disabled: a.Busy,
					}).Lines(2, 5)
				}
				kTextarea(c, a.EditContent, coss.TextareaOptions{
					Label:       i18n.T(i18n.KeySettingsSkillsEditContent),
					Placeholder: i18n.T(i18n.KeySettingsSkillsEditContent),
					MaxLines:    16,
					Disabled:    a.Busy,
				}).Lines(5, 16)
				if a.EditModelInvocable != nil {
					kCheckbox(c, a.EditModelInvocable, providerCopy("模型可调用", "Model invocable"), a.Busy)
				}
				if a.EditUserInvocable != nil {
					kCheckbox(c, a.EditUserInvocable, providerCopy("用户可调用", "User invocable"), a.Busy)
				}
				kHint(c, providerCopy("仅保存会写入 Skill 文件；文本不会在此执行。", "Only Save writes the Skill file; text is not executed here."))
				if a.EditGroups != nil {
					skillGroupPicker(c, "edit-skill-groups/"+a.EditName, a.EditGroups, a.Skills, a.Busy, fg, muted)
				}
				if *recovery {
					kHint(c, providerCopy("草稿已保留。重新核对原文件后可以继续保存。", "Draft retained. Recheck the original file to save again."))
					label := providerCopy("重新核对并保留草稿", "Recheck and keep draft")
					if a.Busy {
						label = providerCopy("正在核对…", "Rechecking…")
					}
					if kButtonDisabled(c, label, a.Busy || a.OnReloadEdit == nil).AlignSelf(ui.Start).Clicked() && a.OnReloadEdit != nil {
						a.OnReloadEdit()
					}
				}
			}, func() {
				if kButtonDisabled(c, providerCopy("取消", "Cancel"), a.Busy || a.OnCancelEdit == nil).Clicked() && a.OnCancelEdit != nil {
					a.OnCancelEdit()
				}
				if kPrimary(c, i18n.T(i18n.KeySettingsSkillsSaveUpdate), a.Busy || !a.EditCanSave).Clicked() {
					a.OnSaveUpdate()
				}
			}, ui.DialogOptions{DismissDisabled: a.Busy})
			if !open && !a.Busy && a.OnCancelEdit != nil {
				a.OnCancelEdit()
			}
		}
		if a.RemovalID != 0 {
			open := true
			bodyDialogFooterWidth(c.Key("skill-delete-review"), &open, providerCopy("永久删除技能", "Permanently delete skill"), 380, func() {
				ui.Text(c, a.RemovalLabel).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				kBody(c, a.RemovalMessage)
			}, func() {
				if a.OnCancelRemoval != nil && kButtonDisabled(c, providerCopy("取消", "Cancel"), a.RemovalBusy).Clicked() {
					a.OnCancelRemoval()
				}
				if a.OnConfirmRemoval != nil && kDanger(c, providerCopy("确认永久删除", "Confirm permanent deletion"), !a.RemovalReady || a.RemovalBusy).Clicked() {
					a.OnConfirmRemoval()
				}
			}, ui.DialogOptions{DismissDisabled: a.RemovalBusy})
			if !open && !a.RemovalBusy && a.OnCancelRemoval != nil {
				a.OnCancelRemoval()
			}
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
			if a.Query != nil {
				ui.Box(c).Grow(1).MinWidth(180).Children(func() {
					coss.InputGroup(c, a.Query, coss.InputGroupOptions{
						InputOptions: coss.InputOptions{
							Placeholder: providerCopy("搜索名称、描述、分组或何时使用", "Search names, descriptions, groups or when to use"),
							Label:       providerCopy("搜索技能", "Search skills"),
						},
						Start: func() { coss.AddonIcon(c, coss.IconSearch) },
					})
				})
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
				if settingsPreferenceSelect(c, &current, options).Width(design.Sp(36)).Label(providerCopy("技能来源筛选", "Skill source filter")).Changed() {
					for _, choice := range choices {
						if current == choice.label {
							*a.SourceFilter = choice.value
							break
						}
					}
				}
			}
			hasFilters := a.Query != nil && strings.TrimSpace(*a.Query) != "" || a.SourceFilter != nil && *a.SourceFilter != "" && *a.SourceFilter != "all"
			if hasFilters && kGhost(c, providerCopy("清除筛选", "Clear filters"), false).Clicked() {
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
				kHint(c, i18n.T(i18n.KeySettingsSkillsEmpty))
			} else {
				ui.Column(c).FillWidth().Padding(design.Sp(16), 0).AlignItems(ui.Center).Gap(design.Sp(2)).Children(func() {
					ui.Icon(c, skillsEmptySparkleIcon).Size(28, 28).TextColor(muted)
					kHeading(c, providerCopy("还没有可用技能", "No skills available yet")).AlignSelf(ui.Center)
					kHint(c, i18n.T(i18n.KeySettingsSkillsEmpty)).AlignSelf(ui.Center)
					if a.HubQuery != nil && a.OnSearchHub != nil {
						if kButton(c, i18n.T(i18n.KeySurfacesSkillsBrowseHub)).Clicked() {
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
						FillWidth().Padding(design.Sp(1.25), 0).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
					ui.Column(c.Key("skill-rows")).FillWidth().MinWidth(0).Gap(0).Children(func() {
						ui.Box(c).FillWidth().Height(.667).Background(fonttheme.Color(modelColors(c).BorderL1))
						for _, row := range group.Skills {
							skillCompactRow(c, a, row, fg, muted)
						}
					})
					continue
				}
				label := group.Name
				if label == "" {
					label = i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup)
				}
				section := ui.Column(c.Key("skill-group/" + group.Name)).FillWidth().MinWidth(0).Gap(design.Sp(1))
				open := ui.Local(section, "expanded", func() bool { return true })
				section.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
						disclosure := ui.ButtonBase(c).Label("Skill group · "+label).Grow(1).MinWidth(0).BorderWidth(0).Background(ui.Transparent).Padding(design.Sp(1), 0).Justify(ui.Start).AlignItems(ui.Center).Gap(design.Sp(1.5)).Cursor(ui.CursorPointer)
						disclosure.Children(func() {
							icon := "right"
							if *open {
								icon = "down"
							}
							ui.Icon(c, bodyIcon(icon)).Size(12, 12).TextColor(muted).PassThrough()
							ui.Text(c, skillsGroupCountLabel(label, len(group.Skills))).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted).PassThrough()
						})
						if disclosure.Clicked() {
							*open = !*open
						}
						enabled := group.AllEnabled
						groupDisabled := a.Busy || group.Writable == 0 || a.OnSetVisibleGroupInvocable == nil && a.OnSetGroupInvocable == nil
						if settingsToggle(c, "Enable skill group · "+label, &enabled, groupDisabled).Clicked() && !groupDisabled {
							if a.OnSetVisibleGroupInvocable != nil {
								a.OnSetVisibleGroupInvocable(append([]SkillRow(nil), group.Skills...), !enabled)
							} else {
								a.OnSetGroupInvocable(group.Name, !enabled)
							}
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
			bodyDialogFooterWidth(c, createOpen, providerCopy("添加技能", "Add skill"), 720, func() {
				if a.OnSelectRoot != nil {
					kHeading(c, providerCopy("创建或安装到", "Create or install into"))
					ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
						userLabel, projectLabel := "User skills", "Project skills"
						isProject := a.TargetRoot == "project-dsh"
						if isProject {
							projectLabel = "✓ " + projectLabel
						} else {
							userLabel = "✓ " + userLabel
						}
						if kChoiceButton(c, userLabel, !isProject, false).Clicked() {
							a.OnSelectRoot("user-dsh")
						}
						if a.ProjectCwd != "" && kChoiceButton(c, projectLabel, isProject, false).Clicked() {
							a.OnSelectRoot("project-dsh")
						}
					})
					if a.TargetRoot == "project-dsh" && a.ProjectCwd != "" {
						kHint(c, providerCopy("项目目录：", "Project directory: ")+a.ProjectCwd)
					} else if a.ProjectCwd == "" {
						kHint(c, providerCopy("先选择一个工作区目录，才能创建或安装项目技能。", "Select a workspace directory before creating or installing project skills."))
					}
				}
				if a.CreateName != nil && a.OnCreate != nil {
					ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
						createIn := kInputRow(c, a.CreateName, coss.InputOptions{
							Placeholder: i18n.T(i18n.KeyComposerSkillsCreateName),
							Label:       i18n.T(i18n.KeyComposerSkillsCreateName),
						})
						submitCreate := func() {
							name := strings.TrimSpace(*a.CreateName)
							if name != "" {
								a.OnCreate(name)
								*a.CreateName = ""
							}
						}
						if createIn.Submitted() && !createIn.Composing() {
							submitCreate()
						}
						if kPrimary(c, i18n.T(i18n.KeyComposerSkillsCreate), false).Clicked() {
							submitCreate()
						}
					})
				}
			}, nil)
		}
		if *hubOpen && a.Hub == nil && a.HubQuery != nil && a.OnSearchHub != nil {
			bodyDialogFooterWidth(c, hubOpen, providerCopy("搜索技能", "Find skills"), 760, func() {
				kHeading(c, i18n.T(i18n.KeySettingsSkillsHubTitle)).Padding(design.Sp(3), 0, 0, 0)
				kHint(c, i18n.T(i18n.KeySettingsSkillsHubHint))
				ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					hubIn := kInputRow(c, a.HubQuery, coss.InputOptions{
						Placeholder: i18n.T(i18n.KeySettingsSkillsHubPlaceholder),
						Label:       i18n.T(i18n.KeySettingsSkillsHubSearch),
					})
					if hubIn.Submitted() && !hubIn.Composing() && strings.TrimSpace(*a.HubQuery) != "" {
						a.OnSearchHub(*a.HubQuery)
					}
					if kPrimary(c, i18n.T(i18n.KeySettingsSkillsHubSearch), false).Clicked() {
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
							kHint(c, detail)
						}
						if a.OnInstallHub != nil && h.Repo != "" && h.Path != "" && h.SkillName != "" && h.Status != "installed" && h.Status != "conflict" {
							if kButton(c, i18n.T(i18n.KeySettingsSkillsHubInstall)).AlignSelf(ui.Start).Clicked() {
								a.OnInstallHub(h.Repo, h.Path, h.SkillName)
							}
						}
					}, fg, muted)
				}
			}, nil)

		}
	})
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
	colors := modelColors(c)
	ui.Column(c.Key("skill-row/" + row.Source + "/" + row.Name)).FillWidth().MinWidth(0).Gap(0).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
			main := ui.ButtonBase(c).Label("Inspect skill · " + row.Name).Height(40).Grow(1).MinWidth(180).Padding(0).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Disabled(!row.Writable || a.Busy || a.OnInspect == nil)
			main.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Size(28, 28).Shrink(0).Radius(design.RadiusLG).Background(modelWell(c).Alpha(.8)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
						ui.Icon(c, settingsIcons["SkillOutline"]).Size(16, 16).TextColor(fonttheme.Color(colors.Secondary)).PassThrough()
					})
					ui.Column(c).Grow(1).MinWidth(0).Gap(0).Children(func() {
						ui.Text(c, row.Name).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(design.WeightMedium).TextColor(fg)
						if row.Description != "" {
							ui.Text(c, row.Description).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted).SingleLine().Tooltip(row.Description)
						}
					})
				})
			})
			if main.Clicked() && a.OnInspect != nil {
				a.OnInspect(row.Name)
			}
			ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
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
			ui.Box(c).FillWidth().Padding(0, design.Sp(2), design.Sp(2), design.Sp(2)).Children(func() {
				settingsFailureLine(c, "skill:row/"+row.Name, providerCopy("这个技能没能加载，请重试。", "This skill could not be loaded. Try again."), row.Error)
			})
		}
		ui.Box(c).FillWidth().Height(.667).Background(fonttheme.Color(colors.BorderL1))
	})
}

// skillsBrowseButton is the outline sm action with the magnifier glyph that the
// installed build puts at the right of the skills header.
func skillsBrowseButton(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{
		Variant:  coss.ButtonOutline,
		Size:     coss.ButtonSM,
		Icon:     coss.IconSearch,
		Disabled: disabled,
	}).Label(label).Shrink(0)
}

// skillSourceBadge is the round pill after a skill row: the source name in 12/18
// secondary ink on the raised layer, the way the installed build tags bundled skills.
func skillSourceBadge(c *ui.Context, label, directory string, muted ui.Color) {
	if label == "" {
		return
	}
	badge := coss.Badge(c, label, coss.BadgeOptions{Variant: coss.BadgeSecondary, Size: coss.BadgeMD}).Shrink(0)
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
