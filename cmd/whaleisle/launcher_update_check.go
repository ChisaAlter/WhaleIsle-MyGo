package main

import (
	"context"
	"fmt"

	"github.com/egoist/mygo"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

// Network latency must not hold the launcher's input/render thread. Keep one
// check in flight and publish its result on the window's normal update queue.
func (s *mainState) checkLauncherUpdate(win *mygo.Window) {
	s.mu.Lock()
	if s.launcherUpdateCheckBusy || s.quitting {
		s.mu.Unlock()
		return
	}
	s.launcherUpdateCheckBusy = true
	s.launcherUpdate = resumeText("正在检查更新…", "Checking for updates…")
	s.mu.Unlock()
	route := s.updateDownloadRoute()
	go func() {
		res, err := desktopupdate.CheckForUpdatesVersionRoute(context.Background(), buildVersion, route)
		apply := func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.launcherUpdateCheckBusy = false
			if s.quitting {
				return
			}
			if err != nil {
				s.launcherUpdate = err.Error()
				s.launcherUpdateAvailable = false
				return
			}
			s.launcherUpdateAvailable = res.Status == desktopupdate.StatusUpdateAvailable
			if s.launcherUpdateAvailable {
				s.launcherUpdate = fmt.Sprintf(i18n.T(i18n.KeyLauncherUpdateAvailable), res.Version, buildVersion)
			} else {
				s.launcherUpdate = string(res.Status) + ": " + res.Message
				if res.ReleaseURL != "" {
					s.launcherUpdate += " · " + res.ReleaseURL
				}
			}
		}
		if win != nil {
			win.Update(apply)
		} else {
			apply()
		}
	}()
}
