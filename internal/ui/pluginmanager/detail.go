package pluginmanager

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/errcopy"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

// errcopyText is the sentence for a failure the controller kept as text.
func errcopyText(raw string) string { return errcopy.SentenceText(raw) }

// detailView is one item, package or package row: its header with the
// package actions, its configuration, components and source facts.
func detailView(c *ui.Context, p *page) {
	s, a, m := p.s, p.a, p.m
	subject := s.View
	var pkg *manager.Package
	for i := range s.Packages {
		if s.Packages[i].Name == subject.Name {
			pkg = &s.Packages[i]
			break
		}
	}
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		if pagekit.Back(c, copyText("返回插件", "Back to plugins")) {
			a.Back()
		}
		ui.Spacer(c)
		if pkg != nil && subject.Kind == "package" {
			busy := s.ActionBusy(pkg.Name)
			if manager.UpdateAvailable(*pkg) && coss.Button(c, copyText("更新", "Update"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconDownload, Disabled: busy}).Clicked() {
				a.Update(pkg.Name)
			}
			if (pkg.Installed || pkg.Removable) && coss.Button(c, copyText("卸载", "Uninstall"), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonSM, Icon: coss.IconTrash, Disabled: busy || !pkg.Removable || pkg.ReadOnlyReason != ""}).Clicked() {
				a.Remove(pkg.Name)
			}
			value := pkg.Enabled
			if coss.Switch(c, &value, copyText("启用", "Enabled"), coss.CheckOptions{Disabled: busy || pkg.SwitchLocked()}).Label(copyText("启用 ", "Enable ") + pkg.Name).Changed() {
				a.Enable(pkg.Name, value)
			}
		}
	})
	if subject.Kind == "item" {
		item := configFor(m, subject)
		if item == nil {
			coss.Empty(c, coss.EmptyOptions{Icon: coss.IconPuzzle, Title: copyText("配置页面已不可用。", "Configuration page is no longer available.")}, nil)
			return
		}
		ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
			pagekit.Title(c, item.Title)
			pagekit.Muted(c, item.Summary).FillWidth()
		})
		renderConfig(c, m, subject)
		return
	}
	if pkg == nil {
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconPuzzle, Title: copyText("插件已不在当前清单中。", "Plugin is no longer in the current inventory.")}, nil)
		return
	}
	if pkg.Meta != nil && pkg.Meta.Error != "" {
		pagekit.Note(c, coss.AlertWarning, copyText("插件描述无法完整读取", "The plugin's metadata could not be read in full"), errcopyText(pkg.Meta.Error), nil)
	}
	title, description := pkg.Title(i18n.Locale()), pkg.Summary(i18n.Locale())
	var row *manager.Row
	if subject.Kind == "row" {
		for i := range pkg.Parts {
			if pkg.Parts[i].RowID == subject.RowID {
				row = &pkg.Parts[i]
				break
			}
		}
		if row != nil {
			title = row.Title(i18n.Locale())
			if row.Meta != nil {
				description = row.Meta.Description.Resolve(i18n.Locale())
			}
		}
	}
	version := p.version()
	ui.Row(c).Key("plugin-detail-head").FillWidth().MinWidth(0).Gap(design.Sp(4)).AlignItems(ui.Start).Memo(version).Children(func() {
		artworkSized(c, p, design.Sp(12), design.Sp(9), metaIcon(pkg.Meta))
		ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				pagekit.Title(c, title)
				if pkg.Version != "" {
					coss.Badge(c, "v"+pkg.Version, coss.BadgeOptions{Variant: coss.BadgeOutline})
				}
				packageTags(c, *pkg)
			})
			pagekit.Code(c, pkg.Name).FillWidth().Selectable()
			if description != "" {
				pagekit.Muted(c, description).FillWidth()
			}
		})
	})
	if subject.Kind == "row" {
		if row != nil {
			pagekit.Code(c, row.ModuleName+"\nLoader: "+row.EntryID).FillWidth().Selectable()
		}
		renderConfig(c, m, subject)
		return
	}
	packagePreview(c, pkg.Name, m.Media)
	if pkg.Error != nil {
		pagekit.FaultSentence(c, copyText("插件加载异常", "The plugin has a problem"), errcopySentenceFor(pkg.Error.Code, pkg.Error.Diagnostic), "", nil)
	}
	if pkg.ReadOnlyReason != "" {
		pagekit.Note(c, coss.AlertInfo, copyText("此插件为只读", "This plugin is read-only"), pkg.ReadOnlyReason, nil)
	}
	if configFor(m, subject) != nil || m.RenderConfiguration != nil {
		renderConfig(c, m, subject)
	}
	if pkg.Availability != "missing" {
		partsSection(c, p, pkg, version)
	}
	sourceSection(c, pkg)
}

// partsSection lists a package's components with their switches.
func partsSection(c *ui.Context, p *page, pkg *manager.Package, version uint64) {
	s, a, m := p.s, p.a, p.m
	pagekit.Section(c, copyText("包含的组件", "Plugins in this package"), func() {
		pagekit.Caption(c, partsSummary(pkg.Parts))
	}, func() {
		filter := s.RowFilter
		if len(pkg.Parts) > 10 {
			if coss.InputGroup(c, &filter, coss.InputGroupOptions{InputOptions: coss.InputOptions{Label: copyText("筛选组件", "Filter components"), Placeholder: copyText("筛选组件", "Filter components")},
				Start: func() { coss.AddonIcon(c, coss.IconSearch) }}).Changed() {
				a.FilterRows(filter)
			}
		}
		if len(pkg.Parts) == 0 {
			pagekit.Muted(c, copyText("这个插件包不包含任何组件。", "This package declares no plugin entries.")).FillWidth()
			return
		}
		query := strings.ToLower(strings.TrimSpace(filter))
		shown := 0
		ui.Column(c).Key("plugin-parts").FillWidth().Memo(pagekit.Version(version).String(query).Key()).Children(func() {
			rows(c, func() {
				for _, part := range pkg.Parts {
					r := part
					description := ""
					if r.Meta != nil {
						description = r.Meta.Description.Resolve(i18n.Locale())
					}
					if query != "" && !strings.Contains(strings.ToLower(strings.Join([]string{r.Title(i18n.Locale()), description, r.RowID, r.ModuleName}, " ")), query) {
						continue
					}
					if shown > 0 {
						rowSeparator(c)
					}
					shown++
					rowFrame(ui.Row(c.Key("plugin-row-" + r.RowID))).AlignItems(ui.Start).Children(func() {
						icon := metaIcon(r.Meta)
						if icon == "" {
							icon = builtInArtwork("subagent")
						}
						artworkSized(c, p, design.Sp(8), design.Sp(6), icon)
						ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
							ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								if configFor(m, manager.Subject{Kind: "row", Name: pkg.Name, RowID: r.RowID}) != nil {
									if coss.Button(c, r.Title(i18n.Locale()), coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonSM}).Margin(0, 0, 0, -design.Sp(2)).Clicked() {
										a.OpenRow(pkg.Name, r.RowID)
									}
								} else {
									pagekit.Strong(c, r.Title(i18n.Locale()))
								}
								phaseBadge(c, r)
							})
							pagekit.Code(c, r.RowID+" · "+r.ModuleName).FillWidth()
							if description != "" {
								pagekit.Muted(c, description).FillWidth()
							}
							if r.Meta != nil && r.Meta.Error != "" {
								pagekit.Danger(c, errcopyText(r.Meta.Error)).FillWidth().Role(ui.RoleStatus)
							}
							if r.ReadOnlyReason != "" {
								pagekit.Caption(c, r.ReadOnlyReason).FillWidth()
							}
						})
						value := r.Enabled
						if pkg.Enabled && coss.Switch(c, &value, "", coss.CheckOptions{Label: copyText("启用条目 ", "Enable entry ") + r.RowID,
							Disabled: s.ActionBusy(pkg.Name) || s.ActionBusy(r.EntryID) || r.EntryID == "" || r.ReadOnlyReason != ""}).Changed() {
							a.EnableRow(r.EntryID, value)
						}
					})
				}
				if shown == 0 {
					ui.Row(c).FillWidth().Padding(design.Sp(4)).Children(func() {
						pagekit.Muted(c, copyText("没有匹配的组件。", "No matching components."))
					})
				}
			})
		})
	})
}

// phaseBadge shows a component's run state.
func phaseBadge(c *ui.Context, r manager.Row) {
	switch {
	case !r.Enabled:
		coss.Badge(c, copyText("已关闭", "Off"), coss.BadgeOptions{Variant: coss.BadgeOutline})
	case r.Phase == "active":
		coss.Badge(c, copyText("运行中", "Running"), coss.BadgeOptions{Variant: coss.BadgeSuccess})
	case r.Phase == "failed":
		coss.Badge(c, copyText("异常", "Failed"), coss.BadgeOptions{Variant: coss.BadgeError})
	case r.Phase == "":
		coss.Badge(c, copyText("未运行", "Idle"), coss.BadgeOptions{Variant: coss.BadgeSecondary})
	default:
		coss.Badge(c, r.Phase, coss.BadgeOptions{Variant: coss.BadgeSecondary})
	}
}

// sourceSection names where the package came from.
func sourceSection(c *ui.Context, pkg *manager.Package) {
	pagekit.Section(c, copyText("来源信息", "Source information"), nil, func() {
		source := pkg.Source
		if source == "" {
			switch {
			case pkg.Availability == "missing" && pkg.Installed:
				source = copyText("已声明，但安装包不可用", "Declared dependency; installed files missing")
			case pkg.Availability == "missing":
				source = copyText("未安装", "Not installed")
			default:
				source = copyText("内置", "Built in")
			}
		}
		t := design.For(c)
		ui.Column(c).FillWidth().Gap(design.Sp(1)).Padding(design.Sp(3), design.Sp(4)).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Children(func() {
			pagekit.Body(c, source).FillWidth().Selectable()
			pagekit.Caption(c, pkg.Availability)
			if pkg.InstallTarget != nil {
				pagekit.Code(c, pkg.InstallTarget.Spec+" · "+pkg.InstallTarget.Version).FillWidth().Selectable()
			}
		})
	})
}

// errcopySentenceFor is the sentence for a Host failure with its code.
func errcopySentenceFor(code, raw string) string { return errcopy.SentenceFor(code, raw) }
