package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
)

type nativePluginNavigationState struct {
	owner              *pluginui.Owner
	endpoint           harness.Endpoint
	generation         int
	selected           string
	error              string
	directoryPickID    string
	directoryPickOwner *pluginui.Owner
}

// syncPluginNativeNavigation keeps one hidden source for the current Host. It
// is destroyed with the native surface; no old view can acquire a new Host.
func (s *mainState) syncPluginNativeNavigation(visible bool) {
	defer func() {
		owner := s.pluginNavigationOwner()
		s.mu.Lock()
		sessions := s.sessions
		s.mu.Unlock()
		if owner != nil && sessions != nil {
			owner.SetNavigationContext(sessions.SelectedSessionID())
		}
	}()
	s.mu.Lock()
	p := s.nativePlugins.navigation
	if p == nil {
		p = &nativePluginNavigationState{}
		s.nativePlugins.navigation = p
	}
	st, win := s.bootStatus, s.win
	ready := visible && nativePluginHostEnabled() && st.State == harness.Ready && win != nil && win.IsVisible() && !win.IsMinimized()
	changed := p.generation != st.Generation || p.endpoint != st.Endpoint
	owner := p.owner
	if !ready || changed || (owner != nil && !owner.Alive()) {
		p.owner = nil
		p.selected = ""
		p.error = ""
	}
	p.generation, p.endpoint = st.Generation, st.Endpoint
	missing := ready && p.owner == nil && p.error == ""
	s.mu.Unlock()
	if owner != nil && (!ready || changed || !owner.Alive()) {
		owner.Close()
	}
	if !missing {
		return
	}
	owner, err := pluginui.New(pluginui.Options{Parent: win, Origin: st.Endpoint.Origin, Cookie: st.Endpoint.Cookie, Navigation: true, Changed: func() { win.Update(func() {}) }, OnFailure: s.onEmbeddedPageFailure, PermissionPolicy: desktopwebview.PermissionPolicy{SessionHost: true, NotificationsAllowed: systemNotificationsSupported}.Decide})
	s.mu.Lock()
	current := s.nativePlugins.navigation == p && s.bootStatus.State == harness.Ready && s.bootStatus.Generation == st.Generation && s.bootStatus.Endpoint == st.Endpoint
	if current {
		if err != nil {
			p.error = err.Error()
		} else {
			p.owner = owner
		}
	}
	s.mu.Unlock()
	if !current && owner != nil {
		owner.Close()
	}
}

func (s *mainState) pluginNativeViews() []pluginui.NativeView {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner == nil {
		return nil
	}
	return owner.Snapshot().Views
}
func (s *mainState) openPluginNativeView(handle string) {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner == nil {
		return
	}
	for _, view := range owner.Snapshot().Views {
		if view.Handle == handle {
			s.mu.Lock()
			if s.nativePlugins.navigation == p && p.owner == owner {
				p.selected = handle
			}
			s.mu.Unlock()
			owner.Select(pluginui.Subject{Kind: "view", Name: handle})
			return
		}
	}
}
func (s *mainState) closePluginNativeView() {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
		p.selected = ""
	}
	s.mu.Unlock()
	if owner != nil {
		owner.Select(pluginui.Subject{Kind: "navigation"})
	}
}
func (s *mainState) renderPluginNativeView(c *ui.Context) bool {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	selected := ""
	if p != nil {
		owner, selected = p.owner, p.selected
	}
	s.mu.Unlock()
	if owner == nil || selected == "" {
		return false
	}
	snapshot := owner.Snapshot()
	var selectedView *pluginui.NativeView
	for _, view := range snapshot.Views {
		if view.Handle == selected {
			v := view
			selectedView = &v
			break
		}
	}
	if selectedView == nil {
		s.closePluginNativeView()
		return false
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		pluginModuleFailures(c, owner)
		ui.Text(c, selectedView.Label).FontSize(20).FontWeight(600)
		if ui.Button(c.Key("plugin-native-back"), "Back").Clicked() {
			s.closePluginNativeView()
		}
		if !owner.RenderNativeSlot(c, selectedView.Slot, "", -1) {
			ui.Text(c, "Loading plugin view…").FillWidth()
		}
	})
	return true
}

// renderPluginNativeSlotView is the in-place settings/sidebar consumer. It does
// not add a center-page shell or move the native session header/composer.
func (s *mainState) renderPluginNativeSlotView(c *ui.Context, handle string) bool {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner == nil {
		return false
	}
	snapshot := owner.Snapshot()
	if snapshot.Subject == nil || snapshot.Subject.Kind != "view" || snapshot.Subject.Name != handle {
		return false
	}
	for _, view := range snapshot.Views {
		if view.Handle == handle {
			pluginModuleFailures(c, owner)
			return owner.RenderNativeSlot(c, view.Slot, "", -1)
		}
	}
	return false
}

func (s *mainState) renderPluginShellOverlay(c *ui.Context) {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner != nil {
		owner.RenderGlobalNativeSlot(c, "shell.overlay")
	}
}

// Only an actual elected third-party RootEntry can replace native client-area
// navigation/body. Go retains caption, system dialogs, exit and window control.
func (s *mainState) renderPluginRoot(c *ui.Context) bool {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner == nil {
		return false
	}
	return owner.RenderGlobalNativeSlot(c, "root")
}
func (s *mainState) pluginRootRenderer() func(*ui.Context) bool {
	s.mu.Lock()
	p := s.nativePlugins.navigation
	var owner *pluginui.Owner
	if p != nil {
		owner = p.owner
	}
	s.mu.Unlock()
	if owner == nil {
		return nil
	}
	return owner.GlobalNativeRenderer("root")
}
