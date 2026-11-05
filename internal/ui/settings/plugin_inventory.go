package settings

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type PluginInventoryAccess struct {
	ClientModules                        pluginslots.ClientModulesSnapshot
	ClientPage                           string
	OnRetryClientModules                 func()
	Snapshot                             plugininventory.Snapshot
	OnRefresh, OnToggleGlobal            func()
	OnQuery, OnSelectPreset, OnToggleRow func(string)
}

func inventoryEnablement(value string) string {
	switch value {
	case "enabled":
		return pluginsText("已启用", "Enabled")
	case "disabled":
		return pluginsText("已停用", "Disabled")
	case "conditional":
		return pluginsText("按条件启用", "Conditional")
	default:
		return pluginsText("启用状态未知", "Enablement unknown")
	}
}
func inventoryPhase(value string) string {
	switch value {
	case "pending":
		return pluginsText("等待中", "Pending")
	case "loading":
		return pluginsText("加载中", "Loading")
	case "active":
		return pluginsText("运行中", "Active")
	case "failed":
		return pluginsText("加载失败", "Failed")
	case "unloading":
		return pluginsText("卸载中", "Unloading")
	case "":
		return pluginsText("尚无运行实例", "No observed runtime instance")
	default:
		return value
	}
}

func pluginInventoryPane(c *ui.Context, a *PluginInventoryAccess, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MaxWidth(760).MinWidth(0).Gap(14).Children(func() {
		pluginInventoryContent(c, a, fg, muted)
	})
}

func pluginInventoryContent(c *ui.Context, a *PluginInventoryAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	v := s.View(i18n.Locale())
	if !s.Connected {
		ui.Text(c, pluginsText("运行时未连接。", "Runtime disconnected.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Err != "" {
		ui.Text(c, s.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		if s.Ready {
			ui.Text(c, pluginsText("刷新失败，以下保留本次 Host 上次成功读取的实例。", "Refresh failed. Showing the last successful inventory from this Host.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	}
	if s.Err != "" && a.OnRefresh != nil {
		if ui.Button(c, pluginsText("重新读取插件", "Retry plugin inventory")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || !s.Connected).Clicked() {
			a.OnRefresh()
		}
	}
	// 0.3.13: one full-width search field, then the session plugins of the chosen
	// Agent preset under a collapsible header that carries the preset select.
	query := s.Query
	ui.Row(c.Key("plugin-search")).FillWidth().MinWidth(0).Height(36).Shrink(0).Padding(0, 12).Gap(8).Radius(12).AlignItems(ui.Center).
		Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).Children(func() {
		ui.Icon(c, bodyIcon("search")).Size(16, 16).Shrink(0).TextColor(muted)
		if ui.TextInput(c, &query).Grow(1).MinWidth(0).Height(34).Padding(0).Radius(0).BorderWidth(0).Background(ui.Transparent).FontSize(fonttheme.InterfaceSize(13)).Label("Plugin search").Placeholder(pluginsText("搜索插件", "Search plugins")).Changed() && a.OnQuery != nil {
			a.OnQuery(query)
		}
	})
	if !s.Ready {
		return
	}
	sessionHeader := ui.Column(c.Key("plugin-session-section")).FillWidth().MinWidth(0).Gap(10)
	sessionOpen := ui.Local(sessionHeader, "open", func() bool { return true })
	sessionHeader.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				if inventoryGroupToggle(c, pluginsText("会话插件", "Session plugins"), *sessionOpen || v.Searching, fg, muted) {
					*sessionOpen = !*sessionOpen
				}
				ui.Box(c).Grow(1).MinWidth(0)
				if len(v.Options) > 0 {
					options, ids := []string{}, map[string]string{}
					selected := ""
					for _, p := range v.Options {
						label := inventoryPresetName(p)
						if p.Default {
							label += pluginsText("（默认）", " (default)")
						}
						if p.Broken != "" {
							label += pluginsText(" · 配置异常", " · broken")
						}
						options, ids[label] = append(options, label), p.ID
						if p.ID == v.SelectedID {
							selected = label
						}
					}
					if settingsPreferenceSelectRadius(c, &selected, options, 12).Label(pluginsText("Agent 预设", "Agent preset")).Changed() && a.OnSelectPreset != nil {
						a.OnSelectPreset(ids[selected])
					}
				}
			})
			inventoryGroupSub(c, pluginsText("由 Agent 预设按会话组成", "Composed per session by the Agent preset"), muted)
		})
		if *sessionOpen || v.Searching {
			for _, p := range v.Presets {
				if len(v.Presets) > 1 {
					ui.Text(c, fmt.Sprintf("%s (%d)", p.Label, len(p.Rows))).FontSize(fonttheme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
				}
				if p.Broken != "" {
					ui.Text(c, p.Broken).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				}
				if len(p.Rows) == 0 && p.Broken == "" {
					ui.Text(c, pluginsText("此预设没有匹配条目。", "No matching entries in this preset.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
				pluginInventoryCards(c.Key("inventory-preset-"+p.ID), a, p.Rows, fg, muted)
			}
		}
	})
	open := s.GlobalOpen || v.Searching
	globalGroup := ui.Column(c.Key("plugin-global-section")).FillWidth().MinWidth(0).Gap(10)
	globalGroup.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				if inventoryGroupToggle(c, pluginsText("全局插件", "Global plugins"), open, fg, muted) && a.OnToggleGlobal != nil {
					a.OnToggleGlobal()
				}
			})
			inventoryGroupSub(c, pluginsText("系统与所有会话共用", "Shared by the system and every session"), muted)
		})
		if open {
			pluginInventoryCards(c.Key("inventory-global"), a, v.Global, fg, muted)
		}
	})
	if v.Empty {
		ui.Text(c, pluginsText("没有匹配的插件实例。", "No matching plugin entries.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
}

// Each inventory scope measures its own content width. Expanded cards keep
// their peers aligned to the top, as in the source's two-column inventory.
func pluginInventoryCards(c *ui.Context, a *PluginInventoryAccess, rows []plugininventory.DisplayRow, fg, muted ui.Color) {
	grid := ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(10).AlignItems(ui.Start)
	width := grid.Bounds().W
	grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	grid.Children(func() {
		for _, row := range rows {
			pluginInventoryCard(c, a, row, width, fg, muted)
		}
	})
}

func inventoryChevron(c *ui.Context, open bool, ink ui.Color) {
	ui.Box(c).Width(12).Height(12).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		if open {
			path.MoveTo(r.X+3, r.Y+4)
			path.LineTo(r.X+6, r.Y+7)
			path.LineTo(r.X+9, r.Y+4)
		} else {
			path.MoveTo(r.X+4, r.Y+3)
			path.LineTo(r.X+7, r.Y+6)
			path.LineTo(r.X+4, r.Y+9)
		}
		p.StrokePath(path, 1.2, ink)
	})
}

func inventoryPhaseColor(c *ui.Context, phase string, muted ui.Color) ui.Color {
	switch phase {
	case "active":
		return c.Theme().Success
	case "failed":
		return c.Theme().Danger
	case "pending", "loading", "unloading":
		return c.Theme().Warning
	default:
		return muted
	}
}

func inventoryRowState(r plugininventory.DisplayRow) string {
	if r.Phase == "failed" {
		return inventoryPhase(r.Phase)
	}
	if len(r.ProvidedBy) > 0 {
		return pluginsText("由预设提供", "Provided by presets")
	}
	return inventoryEnablement(r.Enabled)
}

func pluginInventoryCard(c *ui.Context, a *PluginInventoryAccess, r plugininventory.DisplayRow, width float32, fg, muted ui.Color) {
	open := a.Snapshot.Expanded == r.Key
	stroke := modelInputBorder(c)
	if r.Phase == "failed" || r.MetadataError != "" {
		stroke = c.Theme().Danger
	}
	card := ui.Column(c.Key("inventory-card-"+r.Key)).MinWidth(0).Border(.667, stroke).Radius(20).Background(modelWell(c).Alpha(.8)).Clip()
	if width > 520 {
		card.Width((width - 10) / 2)
	} else {
		card.FillWidth()
	}
	card.Children(func() {
		button := ui.Button(c.Key("inventory-toggle-"+r.Key), "").Label(r.Title).Description(inventoryRowState(r)+"; "+inventoryPhase(r.Phase)).Role(ui.RoleDisclosure).Expanded(open).FillWidth().MinWidth(0).Padding(12, 14).BorderWidth(0).Radius(0).Background(ui.Transparent).TextColor(fg)
		if button.Hovered() || open {
			button.Background(c.Theme().SurfaceHover)
		}
		button.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(2).PassThrough().Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, r.Title).Grow(1).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
					// 0.3.13 marks only the exceptions: a pill for a disabled, failed or
					// preset-provided entry, nothing for an ordinary enabled one.
					if r.Enabled != "enabled" || r.Phase == "failed" || len(r.ProvidedBy) > 0 {
						ui.Text(c, inventoryRowState(r)).Shrink(0).Padding(1, 8).Radius(999).Background(fonttheme.Color(modelColors(c).SelectorFill)).Label(inventoryPhase(r.Phase)).
							FontSize(fonttheme.InterfaceSize(11)).FixedLineHeight(fonttheme.InterfaceSize(17)).FontWeight(500).TextColor(inventoryPhaseColor(c, r.Phase, fonttheme.Color(modelColors(c).Secondary)))
					}
					inventoryCardChevron(c, open, muted)
				})
				if r.Description != "" {
					text := ui.Text(c, r.Description).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
					if !open {
						text.MaxLines(2)
					}
				}
				subtitle := strings.TrimPrefix(r.EntryID, "include:")
				if subtitle != "" && subtitle != r.Title {
					ui.Row(c).FillWidth().MinWidth(0).Padding(6, 0, 0, 0).Children(func() {
						ui.Text(c, subtitle).SingleLine().MinWidth(0).Font(fonttheme.RenderPreferences().Code.Family).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).
							TextColor(fonttheme.Color(modelColors(c).Secondary)).Padding(1, 6).Background(fonttheme.Color(modelColors(c).SelectorFill)).Radius(4).Tooltip(r.EntryID)
					})
				}
			})
		})
		if button.Clicked() && a.OnToggleRow != nil {
			a.OnToggleRow(r.Key)
		}
		if r.MetadataError != "" {
			ui.Text(c, pluginsText("元数据错误：", "Metadata error: ")+r.MetadataError).FillWidth().Padding(0, 16, 10, 16).FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		if open {
			ui.Column(c).FillWidth().MinWidth(0).Padding(10, 16, 12, 16).Gap(6).Background(c.Theme().Surface).Children(func() {
				ui.Text(c, pluginsText("模块：", "Module: ")+r.Module).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				entryID := r.EntryID
				if entryID == "" {
					entryID = pluginsText("未声明条目标识", "No entry ID declared")
				}
				ui.Text(c, "Loader: "+entryID).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				if len(r.ProvidedBy) > 0 {
					ui.Text(c, pluginsText("全局已禁用；Agent 预设在各会话中提供此模块。", "Disabled globally; agent presets provide it per session.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					ui.Text(c, pluginsText("启用预设：", "Enabled in: ")+strings.Join(r.ProvidedBy, " · ")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				} else {
					ui.Text(c, inventoryEnablement(r.Enabled)+" · "+inventoryPhase(r.Phase)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
				if r.Condition != "" {
					ui.Text(c, pluginsText("启用条件：", "Enablement condition: ")+r.Condition).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
			})
		}
	})
}

// The client page owns this status and retry. Host inventory refresh remains a
// separate read-only action and must never stand in for entries.retry().
func clientModulesPane(c *ui.Context, a *PluginInventoryAccess, fg, muted ui.Color) {
	s := a.ClientModules
	ui.Text(c, pluginsText("客户端模块", "Client modules")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	if !s.Connected {
		ui.Text(c, pluginsText("客户端页面未连接。打开此 Host 的插件页面后可查看模块同步状态。", "Client page disconnected. Open a plugin page for this Host to inspect module synchronization.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	}
	ui.Text(c, pluginsText("所属页面：", "Page: ")+a.ClientPage).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	for _, failure := range s.Failures {
		ui.Text(c, failure.ID+": "+failure.Message).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	settingsNotice(c, "plugin_inventory:304", "error", s.Error != "", func() string { return s.Error })
	if a.OnRetryClientModules != nil && ui.Button(c, pluginsText("重试客户端模块", "Retry client modules")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Syncing || s.Retrying || (len(s.Failures) == 0 && s.Error == "")).Clicked() {
		a.OnRetryClientModules()
	}
}

// inventoryCardChevron is the 12 DIP chevron at a card's trailing edge: down while
// the card is closed, up once it is expanded.
func inventoryCardChevron(c *ui.Context, open bool, ink ui.Color) {
	ui.Box(c).Width(12).Height(12).Shrink(0).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		if open {
			path.MoveTo(r.X+3, r.Y+7.2)
			path.LineTo(r.X+6, r.Y+4.5)
			path.LineTo(r.X+9, r.Y+7.2)
		} else {
			path.MoveTo(r.X+3, r.Y+4.5)
			path.LineTo(r.X+6, r.Y+7.2)
			path.LineTo(r.X+9, r.Y+4.5)
		}
		p.StrokePath(path, 1.2, ink)
	})
}

// inventoryGroupToggle is a group's disclosure: a 22 DIP row with the 12 DIP
// chevron and the 14/22 title, as wide as its content.
func inventoryGroupToggle(c *ui.Context, title string, open bool, fg, muted ui.Color) bool {
	button := ui.Button(c, "").Label(title).Role(ui.RoleDisclosure).Expanded(open).MinWidth(0).Height(22).Padding(0).Gap(8).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).FocusRing(false)
	button.Children(func() {
		inventoryChevron(c, open, muted)
		ui.Text(c, title).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(fg).SingleLine()
	})
	return button.Clicked()
}

// inventoryGroupSub is the 12/18 note under a group title, indented past the chevron.
func inventoryGroupSub(c *ui.Context, text string, muted ui.Color) {
	ui.Text(c, text).FillWidth().Padding(0, 0, 0, 20).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
}

// inventoryPresetName is the name the switcher shows: the shipped presets read
// as their localized mode names, everything else keeps the inventory label.
func inventoryPresetName(p plugininventory.DisplayPreset) string {
	if name, _ := agentPresetDisplay(session.ManagedAgentPreset{ID: p.ID, BuiltIn: true}); name != "" && name != p.ID {
		return name
	}
	return p.Label
}
