package main

import "whaleisle/internal/ui/frame"

// bindPluginSessionNavigation offers only the explicit existing owner actions to
// the outer navigation shell. It never requests captures or renders outlets.
func (s *mainState) bindPluginSessionNavigation(m *frame.Model) {
	if s == nil || m == nil || m.Store == nil || m.WorkspacePage || frame.MainConversationChildActive(*m) {
		return
	}
	s.mu.Lock()
	p := s.nativePlugins.session
	if p == nil || !p.visible || p.sessions != m.Store || s.sessions != m.Store {
		s.mu.Unlock()
		return
	}
	owner := p.owner
	s.mu.Unlock()
	selected := m.Store.SelectedSessionID()
	current := func() bool {
		s.mu.Lock()
		valid := s.nativePlugins.session == p && p.visible && p.sessions == m.Store && s.sessions == m.Store && p.owner == owner
		s.mu.Unlock()
		return valid && m.Store.SelectedSessionID() == selected
	}
	m.SessionPluginsActive = owner != nil && owner.Alive() && s.pluginSessionCurrent(p)
	m.OnOpenSessionPlugins = func() {
		if current() {
			s.openPluginSession(p)
		}
	}
	m.OnCloseSessionPlugins = func() {
		if current() {
			s.closePluginSession(p)
		}
	}
}
