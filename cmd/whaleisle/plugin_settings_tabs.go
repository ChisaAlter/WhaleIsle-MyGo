package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// pluginSettingsTabs shares the global navigation source, not the Plugins
// manager or a Session source. The original Settings section owns tab drafts.
func (s *mainState) pluginSettingsTabs() []settings.PluginSettingsTab {
	s.mu.Lock()
	navigation := s.nativePlugins.navigation
	if navigation == nil || navigation.owner == nil {
		s.mu.Unlock()
		return nil
	}
	owner := navigation.owner
	s.mu.Unlock()
	var tabs []settings.PluginSettingsTab
	for _, entry := range owner.SettingsPluginTabs() {
		row := entry
		tabs = append(tabs, settings.PluginSettingsTab{ID: row.ID, Label: row.Label, Order: row.Order, Render: func(c *ui.Context, active bool) {
			s.mu.Lock()
			current := s.nativePlugins.navigation == navigation && navigation.owner == owner
			s.mu.Unlock()
			if !current {
				return
			}
			if !owner.RenderSettingsPluginTab(c, row.Handle, active) && active {
				ui.Text(c, pluginCopy("正在读取插件设置…", "Loading plugin settings…")).FillWidth()
			}
		}})
	}
	return tabs
}
