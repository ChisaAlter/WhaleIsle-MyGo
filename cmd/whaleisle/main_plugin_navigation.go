package main

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/settings"
)

// An elected extension may replace client navigation/body, never the native
// caption or the system layers deferred by mainView. Explicit native launcher
// and settings routes remain available above that replacement.
func (s *mainState) paintPluginRootClient(c *ui.Context, ready, nativeRoute bool, colors chrome.TitleBarColors) bool {
	if !ready || nativeRoute {
		return false
	}
	render := s.pluginRootRenderer()
	if render == nil {
		return false
	}
	s.mu.Lock()
	directoryPicker, sessions := s.dirPickOpen, s.sessions
	s.mu.Unlock()
	// These dialogs are painted by the native conversation route, unlike the
	// global confirmation/exit layers. An elected root must yield to them.
	if directoryPicker || sessions != nil && sessions.Snapshot().WorkspaceNavigation != nil {
		return false
	}
	frame.PluginRootCaption(c, colors)
	ui.Scroll(c.Key("plugin-root-client-body")).Label("Plugin root client body").FillWidth().Grow(1).MinHeight(0).MinWidth(0).
		Padding(0, s.padReservedWidth(c), 0, 0).Children(func() {
		if !render(c) {
			// The source was withdrawn after layout selection. Rebuild rather
			// than painting stale content or a second native caption this frame.
			c.Invalidate()
		}
	})
	return true
}

func (s *mainState) paintPluginShellOverlayLayer(c *ui.Context) {
	s.mu.Lock()
	win := s.win
	masked := s.settingsOpen || s.launcherOpen || s.closingOverlay || s.confirmAsk != nil || s.dirPickOpen || s.quitting
	s.mu.Unlock()
	if masked || win == nil || !win.IsVisible() || win.IsMinimized() {
		return
	}
	ui.Overlay(c, func() { s.renderPluginShellOverlay(c) })
}

func (s *mainState) pluginRichFrameVisible() bool {
	s.mu.Lock()
	win := s.win
	masked := s.launcherOpen || s.closingOverlay || s.confirmAsk != nil || s.dirPickOpen || s.quitting
	s.mu.Unlock()
	// Settings may itself contain the selected plugin's rich controls. The
	// per-frame seats and native clip hide unpainted main/other-settings guests.
	return !masked && win != nil && win.IsVisible() && !win.IsMinimized()
}

func (s *mainState) selectedPluginMainView() string {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	selected := ""
	if p != nil {
		selected = p.selected
	}
	s.mu.Unlock()
	if selected == "" {
		return ""
	}
	for _, view := range s.pluginNativeViews() {
		if view.Handle == selected && (view.Slot == "main" || view.Slot == "sidebar.page") {
			return selected
		}
	}
	return ""
}

func (s *mainState) withPluginSettingsSections(m settings.Model) settings.Model {
	s.mu.Lock()
	selected := s.settingsEmbeddedSection
	s.mu.Unlock()
	for _, row := range s.pluginNativeViews() {
		if row.Slot != "settings.section" && row.Slot != "settings.remote.tab" {
			continue
		}
		if row.Handle == "" || strings.TrimSpace(row.Label) == "" {
			continue
		}
		handle := row.Handle
		id := settings.SectionID("plugin/native-settings/" + handle)
		m.ContributedSections = append(m.ContributedSections, settings.ContributedSection{ID: id, Label: row.Label, OnSelect: func() {
			if selected != id {
				s.leaveSettingsSection(m, selected)
			}
			s.mu.Lock()
			s.settingsEmbeddedSection = id
			win := s.win
			s.mu.Unlock()
			s.openPluginNativeView(handle)
			if win != nil {
				win.Invalidate()
			}
		}})
		if selected == id {
			m.ActiveSection = id
			m.SectionContent = func(c *ui.Context) {
				if !s.renderPluginNativeSlotView(c, handle) {
					ui.Text(c, resumeText("正在读取插件设置…", "Loading plugin settings…")).FillWidth()
				}
			}
		}
	}
	return m
}
func (s *mainState) openPluginMainView(handle string) {
	for _, view := range s.pluginNativeViews() {
		if view.Handle == handle && (view.Slot == "main" || view.Slot == "sidebar.page") && strings.TrimSpace(view.Label) != "" {
			s.revealMainConversation()
			s.openPluginNativeView(handle)
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			if win != nil {
				win.Invalidate()
			}
			return
		}
	}
}
func (s *mainState) bindPluginNativeNavigation(m *frame.Model) {
	m.NativePages = append([]pluginui.NativeView(nil), s.pluginNativeViews()...)
	m.OnOpenNativePage = s.openPluginMainView
	if selected := s.selectedPluginMainView(); selected != "" {
		m.ActivePanel = selected
	}
}
