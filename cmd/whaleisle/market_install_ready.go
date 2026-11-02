package main

import (
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
)

// marketInstallReadyFor answers whether the marketplace can install. The probe
// resolves the node binary through PATH (a stat per directory) and inspects the
// runtime, and the render path asks for it on every pass: tmp/stalls recorded
// the syscall storm. The answer only changes when the runtime is installed, so
// the last answer is served and refreshed in the background once a second.
var marketInstallReadyProbe backgroundProbe[bool]

func (s *mainState) marketInstallReadyFor(nodeBin string) bool {
	return marketInstallReadyProbe.get(nodeBin, time.Second, func() bool { return desktopmarket.InstallReady(nodeBin) }, s.invalidateMainWindow)
}

// invalidateMainWindow asks the main window for a frame from any goroutine.
func (s *mainState) invalidateMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
