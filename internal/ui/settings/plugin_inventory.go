package settings

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/desktop/pluginslots"
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
		return pluginsText("已禁用", "Disabled")
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
	clientModulesPane(c, a, fg, muted)
	s := a.Snapshot
	v := s.View(i18n.Locale())
	ui.Text(c, pluginsText("插件运行实例", "Plugin runtime inventory")).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, pluginsText("按 Agent 预设或全局配置查看实例；点击条目查看 Loader 标识、条件和运行状态。", "Inspect agent-preset and global entries; expand an entry for its Loader identity, condition and runtime state.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if !s.Connected {
		ui.Text(c, pluginsText("运行时未连接。", "Runtime disconnected.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Err != "" {
		ui.Text(c, s.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		if s.Ready {
			ui.Text(c, pluginsText("刷新失败，以下保留本次 Host 上次成功读取的实例。", "Refresh failed. Showing the last successful inventory from this Host.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	}
	if s.Busy {
		ui.Text(c, pluginsText("正在读取实例…", "Loading inventory…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		query := s.Query
		if ui.TextInput(c, &query).Grow(1).MinWidth(0).Height(36).Placeholder(pluginsText("搜索名称、模块或实例标识", "Search name, module or entry ID")).Changed() && a.OnQuery != nil {
			a.OnQuery(query)
		}
		if a.OnRefresh != nil && ui.Button(c, pluginsText("刷新实例", "Refresh inventory")).Disabled(!s.Connected || s.Busy).Clicked() {
			a.OnRefresh()
		}
	})
	if !s.Ready {
		return
	}
	if len(v.Options) > 0 {
		options, ids := []string{}, map[string]string{}
		selected := ""
		for _, p := range v.Options {
			label := p.Label
			if p.Default {
				label += pluginsText(" · 默认", " · default")
			}
			if p.Broken != "" {
				label += pluginsText(" · 配置异常", " · broken")
			}
			options, ids[label] = append(options, label), p.ID
			if p.ID == v.SelectedID {
				selected = label
			}
		}
		desktopRow(c, pluginsText("Agent 预设", "Agent preset"), func() {
			if ui.Select(c, &selected, options).FillWidth().Changed() && a.OnSelectPreset != nil {
				a.OnSelectPreset(ids[selected])
			}
		}, fg, muted)
	}
	for _, p := range v.Presets {
		ui.Text(c, fmt.Sprintf("%s (%d)", p.Label, len(p.Rows))).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		if p.Broken != "" {
			ui.Text(c, p.Broken).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
		if len(p.Rows) == 0 && p.Broken == "" {
			ui.Text(c, pluginsText("此预设没有匹配条目。", "No matching entries in this preset.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		pluginInventoryCards(c.Key("inventory-preset-"+p.ID), a, p.Rows, fg, muted)
	}
	open := s.GlobalOpen || v.Searching
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		button := ui.Button(c, fmt.Sprintf("%s (%d)", pluginsText("全局条目", "Global entries"), len(v.Global))).Role(ui.RoleDisclosure).Expanded(open).BorderWidth(0).Background(ui.Transparent).TextColor(fg)
		button.Children(func() { inventoryChevron(c, open, muted) })
		if a.OnToggleGlobal != nil && button.Clicked() {
			a.OnToggleGlobal()
		}
	})
	if open {
		pluginInventoryCards(c.Key("inventory-global"), a, v.Global, fg, muted)
	}
	if v.Empty {
		ui.Text(c, pluginsText("没有匹配的插件实例。", "No matching plugin entries.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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

func pluginInventoryCard(c *ui.Context, a *PluginInventoryAccess, r plugininventory.DisplayRow, width float32, fg, muted ui.Color) {
	open := a.Snapshot.Expanded == r.Key
	stroke := c.Theme().Border
	if r.Phase == "failed" || r.MetadataError != "" {
		stroke = c.Theme().Danger
	}
	card := ui.Column(c.Key("inventory-card-"+r.Key)).MinWidth(0).Border(.5, stroke).Radius(12).Background(c.Theme().Background).Clip()
	if width > 520 {
		card.Width((width - 10) / 2)
	} else {
		card.FillWidth()
	}
	card.Children(func() {
		button := ui.Button(c.Key("inventory-toggle-"+r.Key), "").Label(r.Title).Description(inventoryEnablement(r.Enabled)+"; "+inventoryPhase(r.Phase)).Role(ui.RoleDisclosure).Expanded(open).FillWidth().MinWidth(0).MinHeight(52).Padding(12, 14).BorderWidth(0).Radius(0).Background(ui.Transparent).TextColor(fg)
		if button.Hovered() || open {
			button.Background(c.Theme().SurfaceHover)
		}
		button.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).PassThrough().Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, r.Title).Grow(1).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
					ink := inventoryPhaseColor(c, r.Phase, muted)
					ui.Box(c).Width(8).Height(8).Radius(4).Background(ink).Label(inventoryPhase(r.Phase))
					inventoryChevron(c, open, muted)
				})
				if r.Description != "" {
					text := ui.Text(c, r.Description).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					if !open {
						text.MaxLines(2)
					}
				}
				if r.EntryID != "" {
					ui.Text(c, r.EntryID).SingleLine().FillWidth().MinWidth(0).Font(fonttheme.CurrentPreferences().Code.Family).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Padding(1, 6).Background(c.Theme().Surface).Radius(3).Tooltip(r.EntryID)
				}
			})
		})
		if button.Clicked() && a.OnToggleRow != nil {
			a.OnToggleRow(r.Key)
		}
		if r.MetadataError != "" {
			ui.Text(c, pluginsText("元数据错误：", "Metadata error: ")+r.MetadataError).FillWidth().Padding(0, 14, 10, 14).FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		if open {
			ui.Column(c).FillWidth().MinWidth(0).Padding(10, 14, 12, 14).Gap(6).Background(c.Theme().Surface).Children(func() {
				ui.Text(c, pluginsText("模块：", "Module: ")+r.Module).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				entryID := r.EntryID
				if entryID == "" {
					entryID = pluginsText("未声明条目标识", "No entry ID declared")
				}
				ui.Text(c, "Loader: "+entryID).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				ui.Text(c, inventoryEnablement(r.Enabled)+" · "+inventoryPhase(r.Phase)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
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
	ui.Text(c, pluginsText("客户端模块", "Client modules")).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	if !s.Connected {
		ui.Text(c, pluginsText("客户端页面未连接。打开此 Host 的插件页面后可查看模块同步状态。", "Client page disconnected. Open a plugin page for this Host to inspect module synchronization.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		return
	}
	ui.Text(c, pluginsText("所属页面：", "Page: ")+a.ClientPage).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if s.Syncing || s.Retrying {
		ui.Text(c, pluginsText("正在同步客户端模块…", "Synchronizing client modules…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	} else if len(s.Failures) == 0 {
		ui.Text(c, pluginsText("客户端模块已同步。", "Client modules synchronized.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	for _, failure := range s.Failures {
		ui.Text(c, failure.ID+": "+failure.Message).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	if s.Error != "" {
		ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	if a.OnRetryClientModules != nil && ui.Button(c, pluginsText("重试客户端模块", "Retry client modules")).Disabled(s.Syncing || s.Retrying || (len(s.Failures) == 0 && s.Error == "")).Clicked() {
		a.OnRetryClientModules()
	}
}
