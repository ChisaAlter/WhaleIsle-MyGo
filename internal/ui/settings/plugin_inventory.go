package settings

import (
	"fmt"
	"math"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

type PluginInventoryAccess struct {
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

func pluginInventoryMemoKey(c *ui.Context, a *PluginInventoryAccess, width float32) uint64 {
	t := tk(c)
	k := newMemoKey(design.Revision()).u(fonttheme.Revision()).s(i18n.Locale()).
		b(t.Glass, t.IsDark(), chrome.BackdropActive(), c.Theme().Dark).
		u(uint64(pluginInventoryColumns(width)))
	if a == nil {
		return k.b(false).sum()
	}
	s := a.Snapshot
	k.b(true, s.Connected, s.Ready, s.Busy, s.GlobalOpen,
		a.OnRefresh != nil, a.OnToggleGlobal != nil, a.OnQuery != nil, a.OnSelectPreset != nil, a.OnToggleRow != nil).
		u(s.Host).u(s.Revision).u(uint64(len(s.Global))).u(uint64(len(s.Presets))).
		s(s.Query, s.SelectedID, s.Expanded)
	if s.Err != nil {
		k.s(s.Err.SentenceText())
	}
	return k.sum()
}

func pluginInventoryPane(c *ui.Context, a *PluginInventoryAccess, fg, muted ui.Color) {
	// The source's .section column keeps a 12 DIP rhythm; the search field adds
	// its own 2 DIP seat under the page intro.
	pane := ui.Column(c.Key("plugin-inventory-pane")).FillWidth().MaxWidth(760).MinWidth(0).Gap(design.Sp(3))
	width := settingsContentWidth(c, pane, 760)
	pane.Memo(pluginInventoryMemoKey(c, a, width)).Children(func() {
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
	if s.Err != nil {
		settingsFailureLine(c, "plugins:inventory", providerCopy("没能读取插件实例，请重试。", "The plugin inventory could not be read. Try again."), s.Err.SentenceText())
		if s.Ready {
			ui.Text(c, pluginsText("刷新失败，以下保留本次 Host 上次成功读取的实例。", "Refresh failed. Showing the last successful inventory from this Host.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	}
	if s.Err != nil && a.OnRefresh != nil {
		if kButtonDisabled(c, pluginsText("重新读取插件", "Retry plugin inventory"), s.Busy || !s.Connected).Clicked() {
			a.OnRefresh()
		}
	}
	// 0.3.13: one full-width search field, then the session plugins of the chosen
	// Agent preset under a collapsible header that carries the preset select.
	query := s.Query
	t := tk(c)
	searchBorderW, searchBorderC, searchBG := float32(.667), modelInputBorder(c), modelWell(c).Alpha(.8)
	if settingsGlassActive(t) {
		searchBorderW = design.BorderWidth / 2
		searchBG, searchBorderC = settingsGlassInnerCard(t.IsDark())
	}
	ui.Row(c.Key("plugin-search")).FillWidth().MinWidth(0).Height(36).Shrink(0).Padding(0, design.Sp(3)).Gap(design.Sp(2)).Radius(design.RadiusXL).AlignItems(ui.Center).Margin(design.Sp(0.5), 0, 0, 0).
		Border(searchBorderW, searchBorderC).Background(searchBG).Label(pluginsText("搜索插件", "Search plugins")).Children(func() {
		ui.Icon(c, bodyIcon("search")).Size(16, 16).Shrink(0).TextColor(muted)
		if kInputRow(c, &query, coss.InputOptions{
			Size:        coss.InputLG,
			Unstyled:    true,
			Label:       pluginsText("搜索插件", "Search plugins"),
			Placeholder: pluginsText("搜索插件", "Search plugins"),
		}).Changed() && a.OnQuery != nil {
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
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Label(pluginsText("会话插件", "Session plugins")).Children(func() {
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

func pluginInventoryColumns(width float32) int {
	if width > 520 {
		return 2
	}
	return 1
}

// pluginInventoryUniformCardHeight computes the exact uniform outer height of
// collapsed two-column cards from typography metrics so paired Flex rows achieve
// identical card heights in one O(N) layout pass without nested ui.Grid(Fr(1)).
func pluginInventoryUniformCardHeight(rows []plugininventory.DisplayRow) float32 {
	hasDesc, hasSub := false, false
	for _, r := range rows {
		title, desc := pluginInventoryCopy(r)
		if desc != "" {
			hasDesc = true
		}
		if sub := strings.TrimPrefix(r.EntryID, "include:"); sub != "" && sub != title {
			hasSub = true
		}
		if hasDesc && hasSub {
			break
		}
	}
	titleH := max(fonttheme.InterfaceSize(20), fonttheme.InterfaceSize(17)+2)
	h := 2*design.Sp(3) + 2 + titleH
	if hasDesc {
		h += 2 + 2*fonttheme.InterfaceSize(18)
	}
	if hasSub {
		h += 2 + 6 + 2 + fonttheme.InterfaceSize(18)
	}
	return float32(math.Ceil(float64(h)))
}

// Each inventory scope sizes from the known content width on frame 1 (avoiding
// width=0 double reflow) and lays out pairs in flat Flex rows rather than
// nested Fr(1) grids. Collapsed two-column cards share uniform outer height;
// expanded cards align peers to the top so opening one detail panel never
// stretches its neighbor.
func pluginInventoryCards(c *ui.Context, a *PluginInventoryAccess, rows []plugininventory.DisplayRow, fg, muted ui.Color) {
	if len(rows) == 0 {
		return
	}
	hasExpanded := false
	for _, row := range rows {
		if a.Snapshot.Expanded == row.Key {
			hasExpanded = true
			break
		}
	}
	list := ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2.5))
	width := settingsContentWidth(c, list, 760)
	columns := pluginInventoryColumns(width)
	list.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if pluginInventoryColumns(bounds.W) != columns {
			c.Invalidate()
		}
	})
	uniformHeight := columns > 1 && !hasExpanded
	var minCardH float32
	if uniformHeight {
		minCardH = pluginInventoryUniformCardHeight(rows)
	}
	list.Children(func() {
		if columns == 1 {
			for _, row := range rows {
				pluginInventoryCard(c, a, row, false, false, hasExpanded, 0, fg, muted)
			}
			return
		}
		rowAlign := ui.Start
		if uniformHeight {
			rowAlign = ui.Stretch
		}
		for i := 0; i < len(rows); i += 2 {
			end := min(i+2, len(rows))
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).AlignItems(rowAlign).Children(func() {
				for j := i; j < end; j++ {
					pluginInventoryCard(c, a, rows[j], true, uniformHeight, hasExpanded, minCardH, fg, muted)
				}
				if end-i == 1 {
					ui.Box(c).Basis(0).Grow(1).MinWidth(0)
				}
			})
		}
	})
}

func pluginInventoryCopy(r plugininventory.DisplayRow) (title, desc string) {
	title, desc = r.Title, r.Description
	key := strings.TrimPrefix(r.EntryID, "include:")
	if key == "" {
		key = r.Module
	}
	if key == "tool-schedule" || title == "tool-schedule" {
		if i18n.Speaks("zh") {
			if title == "" || title == "tool-schedule" {
				title = "定时提醒工具"
			}
			if desc == "" || desc == "Model-facing reminder management tools" {
				desc = "让 AI 创建、查看和管理定时提醒与周期任务。"
			}
		} else if title == "tool-schedule" {
			title = "Scheduled reminders"
		}
	}
	return title, desc
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

func pluginInventoryCard(c *ui.Context, a *PluginInventoryAccess, r plugininventory.DisplayRow, twoCol, uniformHeight, hasExpanded bool, minCardH float32, fg, muted ui.Color) {
	open := a.Snapshot.Expanded == r.Key
	title, desc := pluginInventoryCopy(r)
	t := tk(c)
	glassActive := settingsGlassActive(t)
	borderW, stroke, cardBG := float32(.667), modelInputBorder(c), modelWell(c).Alpha(.8)
	if glassActive {
		borderW = design.BorderWidth / 2
		cardBG, stroke = settingsGlassInnerCard(t.IsDark())
	}
	if r.Phase == "failed" || r.MetadataError != "" {
		stroke = c.Theme().Danger
	}
	card := ui.Column(c.Key("inventory-card-"+r.Key)).MinWidth(0).Border(borderW, stroke).Radius(20).Background(cardBG).Clip().Label(title)
	if twoCol {
		card.Basis(0).Grow(1)
	} else {
		card.FillWidth()
	}
	if uniformHeight && minCardH > 0 {
		card.MinHeight(minCardH)
	}
	if !open && hasExpanded {
		card.AlignSelf(ui.Start)
	}
	card.Children(func() {
		button := ui.ButtonBase(c.Key("inventory-toggle-"+r.Key)).Label(title).Description(inventoryRowState(r)+"; "+inventoryPhase(r.Phase)).Role(ui.RoleDisclosure).Expanded(open).FillWidth().MinWidth(0).AlignItems(ui.Stretch).Padding(design.Sp(3), design.Sp(3.5)).Cursor(ui.CursorPointer)
		if uniformHeight {
			button.Grow(1)
		}
		if button.Hovered() || open {
			if glassActive {
				hoverWash, _, selWash := settingsGlassListWash(t.IsDark())
				if open {
					button.Background(selWash)
				} else {
					button.Background(hoverWash)
				}
			} else {
				button.Background(t.Accent.Alpha(0.5))
			}
		}
		button.Children(func() {
			inner := ui.Column(c).FillWidth().MinWidth(0).Gap(2).PassThrough()
			if uniformHeight {
				inner.Grow(1)
			}
			inner.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).Grow(1).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
					// 0.3.13 marks only the exceptions: a pill for a disabled, failed or
					// preset-provided entry, nothing for an ordinary enabled one.
					if r.Enabled != "enabled" || r.Phase == "failed" || len(r.ProvidedBy) > 0 {
						ui.Text(c, inventoryRowState(r)).Shrink(0).Padding(1, 8).Radius(999).Background(fonttheme.Color(modelColors(c).SelectorFill)).Label(inventoryPhase(r.Phase)).
							FontSize(fonttheme.InterfaceSize(11)).FixedLineHeight(fonttheme.InterfaceSize(17)).FontWeight(500).TextColor(inventoryPhaseColor(c, r.Phase, fonttheme.Color(modelColors(c).Secondary)))
					}
					inventoryCardChevron(c, open, muted)
				})
				if desc != "" {
					text := ui.Text(c, desc).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
					if !open {
						text.MaxLines(2)
					}
				}
				subtitle := strings.TrimPrefix(r.EntryID, "include:")
				if subtitle != "" && subtitle != title {
					if uniformHeight {
						ui.Box(c).Grow(1).MinHeight(0)
					}
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
			ui.Column(c).FillWidth().MinWidth(0).Padding(10, 16, 12, 16).Gap(6).Background(chrome.FrostInset(fonttheme.Default().Active(c.Theme().Dark), c.Theme().Dark, c.Theme().Surface)).Children(func() {
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
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Label:   title,
		Content: func() {
			inventoryChevron(c, open, muted)
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(fg).SingleLine()
		},
	}).Role(ui.RoleDisclosure).Expanded(open).MinWidth(0).Padding(0)
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
