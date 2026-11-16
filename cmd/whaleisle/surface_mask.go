package main

type mainExclusiveSurfaceState struct {
	quitting           bool
	launcherOpen       bool
	settingsOpen       bool
	closingOverlay     bool
	confirmOpen        bool
	dirPickOpen        bool
	webviewOpen        bool
	browserPreviewOpen bool
	marketOpen         bool
	scheduleOpen       bool
	assistantOpen      bool
	usageOpen          bool
	imOpen             bool
	remoteOpen         bool
	embeddedSection    bool
}

func (s *mainState) exclusiveSurfaceMaskLocked(assistantOpen ...bool) mainExclusiveSurfaceState {
	asstOpen := len(assistantOpen) > 0 && assistantOpen[0]
	return mainExclusiveSurfaceState{
		quitting:           s.quitting,
		launcherOpen:       s.launcherOpen,
		settingsOpen:       s.settingsOpen,
		closingOverlay:     s.closingOverlay,
		confirmOpen:        s.confirmAsk != nil,
		dirPickOpen:        s.dirPickOpen,
		webviewOpen:        s.webviewOpen,
		browserPreviewOpen: s.browserPreviewOpen,
		marketOpen:         s.marketOpen,
		scheduleOpen:       s.scheduleOpen,
		assistantOpen:      asstOpen,
		usageOpen:          s.usageOpen,
		imOpen:             s.imOpen,
		remoteOpen:         s.remoteOpen,
		embeddedSection:    s.settingsEmbeddedSection != "",
	}
}

func (s *mainState) exclusiveSurfaceMask() mainExclusiveSurfaceState {
	asstOpen := s.assistantService().Open()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exclusiveSurfaceMaskLocked(asstOpen)
}

func (m mainExclusiveSurfaceState) modalMasked() bool {
	return m.quitting || m.launcherOpen || m.settingsOpen || m.closingOverlay || m.confirmOpen || m.dirPickOpen
}

func (m mainExclusiveSurfaceState) shellOverlayMasked() bool {
	return m.modalMasked() || m.webviewOpen || m.browserPreviewOpen
}

func (m mainExclusiveSurfaceState) nativeOverlayMasked() bool {
	return m.modalMasked() || m.marketOpen || m.scheduleOpen || m.assistantOpen || m.usageOpen || m.imOpen || m.remoteOpen || m.embeddedSection
}
