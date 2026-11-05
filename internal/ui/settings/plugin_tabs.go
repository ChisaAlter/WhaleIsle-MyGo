package settings

import (
	"sort"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// PluginSettingsTab is a settings.plugins.tab contribution. Render receives
// active=false for a previously visited hidden panel so its owner can retain
// local drafts without leaving a native guest visible or accepting input.
type PluginSettingsTab struct {
	ID, Label string
	Order     int
	Render    func(*ui.Context, bool)
}

func pluginSettingsTabs(c *ui.Context, a *PluginsAccess, fg, muted ui.Color) {
	tabs := append([]PluginSettingsTab(nil), a.Tabs...)
	hasInventory := false
	for _, tab := range tabs {
		hasInventory = hasInventory || tab.ID == "all"
	}
	if !hasInventory && a.Inventory != nil {
		tabs = append(tabs, PluginSettingsTab{ID: "all", Label: pluginsText("全部插件", "All plugins"), Order: 10, Render: func(c *ui.Context, active bool) { pluginInventoryPane(c, a.Inventory, fg, muted) }})
	}
	sort.SliceStable(tabs, func(i, j int) bool { return tabs[i].Order < tabs[j].Order })
	if len(tabs) == 0 {
		return
	}
	root := ui.Column(c.Key("settings-plugin-tabs")).FillWidth().MinWidth(0).Gap(12)
	selected := ui.Local(root, "selected", func() string { return tabs[0].ID })
	visited := ui.Local(root, "visited", func() map[string]bool { return map[string]bool{} })
	found := false
	for _, tab := range tabs {
		found = found || tab.ID == *selected
	}
	if !found {
		*selected = tabs[0].ID
	}
	(*visited)[*selected] = true
	root.Children(func() {
		if len(tabs) > 1 {
			ui.Row(c).FillWidth().Wrap().Gap(8).Role(ui.RoleTabList).Label(pluginsText("插件设置页签", "Plugin settings tabs")).Children(func() {
				for _, tab := range tabs {
					active := tab.ID == *selected
					button := ui.Button(c.Key("plugin-settings-tab/"+tab.ID), tab.Label).Role(ui.RoleTab).Selected(active).Height(30).Padding(0, 12).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(13))
					if active {
						button.Background(c.Theme().SurfaceHover).TextColor(fg)
					}
					if button.Clicked() {
						*selected = tab.ID
						(*visited)[tab.ID] = true
					}
				}
			})
		}
		for _, tab := range tabs {
			if !(*visited)[tab.ID] || tab.Render == nil {
				continue
			}
			active := tab.ID == *selected
			panel := ui.Column(c.Key("plugin-settings-panel/" + tab.ID)).FillWidth().MinWidth(0)
			if !active {
				panel.Absolute().Height(0).MaxHeight(0).Invisible().Inert(true).Disabled(true).Clip()
			}
			panel.Children(func() { tab.Render(c, active) })
		}
	})
}
