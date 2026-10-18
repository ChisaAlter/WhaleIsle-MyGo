package main

// Native child windows cannot be covered by a canvas modal. Hide only their
// surfaces; preserve the owned page, documents, account lease and workspace.
func (s *mainState) syncSettingsNativeSurfaces() {
	s.mu.Lock()
	open, host, prior := s.settingsOpen, s.webviewHost, s.settingsOverlayHost
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
