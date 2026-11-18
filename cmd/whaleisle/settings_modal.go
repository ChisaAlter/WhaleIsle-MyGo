package main

// Actual navigation disposes its old surface. Only a transient native modal
// obscures the current surface without revoking its document or account lease.
func (s *mainState) syncSettingsNativeSurfaces() {
	s.syncMainRouteResources()
	s.syncInstalledPluginView()
	asstOpen := s.assistantService().Open()
	s.mu.Lock()
	mask := s.exclusiveSurfaceMaskLocked(asstOpen)
	open := mask.transientModalMasked()
	host, prior := s.webviewHost, s.settingsOverlayHost
	show := s.webviewOpen || s.browserPreviewOpen
	if open {
		s.settingsOverlayHost = host
	} else {
		s.settingsOverlayHost = nil
	}
	browser := s.browserController
	s.mu.Unlock()
	if open {
		if host != nil {
			host.SetVisible(false)
		}
		if browser != nil {
			for _, tab := range browser.Tabs {
				if tab.Host != nil {
					tab.Host.SetVisible(false)
				}
			}
		}
	} else if prior != nil && prior == host && show && host.Alive() {
		host.SetVisible(true)
	}
}
