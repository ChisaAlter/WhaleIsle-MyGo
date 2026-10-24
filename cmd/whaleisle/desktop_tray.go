package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) setupDesktopTray() {
	// Windows and macOS use explicit right-click popup so an attached native
	// menu cannot consume the left-click restore action. AppIndicator on Linux
	// requires a persistent menu; primary activation depends on indicator and panel support.
	options := mygo.TrayOptions{Icon: trayIcon(), ToolTip: i18n.T(i18n.KeyAppName)}
	if runtime.GOOS == "linux" {
		options.Menu = s.desktopTrayMenu()
	}
	tray, err := mygo.NewTray(options)
	if err != nil {
		log.Println("tray:", err)
		s.showMainWindow() // also undo hidden login startup when the tray failed
		return
	}
	s.mu.Lock()
	s.desktopTray = tray
	s.trayHideNotice.markerPath = filepath.Join(paths.UserDataDir(), "tray-hide-acknowledged")
	s.mu.Unlock()
	tray.OnClick(s.showMainWindow)
	tray.OnRightClick(func() {
		if runtime.GOOS == "linux" {
			s.refreshDesktopTray()
			return
		}
		tray.SetMenu(s.desktopTrayMenu())
		tray.PopUpMenu()
		tray.SetMenu(nil)
	})
	if s.desktopPet != nil {
		s.desktopPet.OnEnabledChanged = s.refreshDesktopTray
	}
}

func (s *mainState) refreshDesktopTray() {
	s.mu.Lock()
	tray := s.desktopTray
	quitting := s.quitting
	s.mu.Unlock()
	if tray != nil && !quitting && runtime.GOOS == "linux" {
		tray.SetMenu(s.desktopTrayMenu())
	}
}

func (s *mainState) closeDesktopTray() {
	s.mu.Lock()
	tray := s.desktopTray
	s.desktopTray = nil
	notice := s.trayHideNotification
	s.trayHideNotification = nil
	s.mu.Unlock()
	if notice != nil {
		notice.Close()
	}
	if tray != nil {
		tray.Destroy()
	}
}

func (s *mainState) desktopTrayMenu() *mygo.Menu {
	s.mu.Lock()
	canRestart := s.sup != nil && !s.quitting && !s.launcherImportBusy && !s.taskRestartPending
	p := s.desktopPet
	canToggle := p != nil && !s.quitting
	s.mu.Unlock()
	enabled := false
	if p != nil && p.Store != nil {
		enabled = p.Store.DesktopPet().Enabled
	}
	return buildDesktopTrayMenu(desktopTrayActions{
		open:     s.showMainWindow,
		launcher: func() { s.openDesktopTrayPage(deeplink.ActionLauncher) },
		settings: func() { s.openDesktopTrayPage(deeplink.ActionSettings) },
		market:   func() { s.openDesktopTrayPage(deeplink.ActionMarket) },
		restart: func() {
			s.mu.Lock()
			sup := s.sup
			quitting := s.quitting
			s.mu.Unlock()
			if sup == nil || quitting {
				return
			}
			s.openDesktopTrayPage(deeplink.ActionLauncher)
			go s.restartUnlessImport(sup)
		},
		quit:       s.quitDesktopTray,
		pet:        s.setDesktopTrayPet,
		canRestart: canRestart, canTogglePet: canToggle, petEnabled: enabled,
	})
}

func (s *mainState) openDesktopTrayPage(action deeplink.Action) {
	s.mu.Lock()
	if s.quitting {
		s.mu.Unlock()
		return
	}
	// The existing deep-link routes own the actual panels and backends. Clear
	// other route flags first so a hidden remote/IM pane cannot mask the target.
	s.imOpen = false
	s.remoteOpen = false
	s.marketOpen = false
	s.toolsMenuOpen = false
	s.mu.Unlock()
	s.applyDeeplink(deeplink.Target{Action: action})
}

func (s *mainState) quitDesktopTray() {
	s.mu.Lock()
	win, sup, quitting := s.win, s.sup, s.quitting
	s.mu.Unlock()
	if quitting {
		return
	}
	// Parent confirmation must be visible, including when invoked from the tray.
	s.showMainWindow()
	s.requestQuitProtected(win, sup)
}

func (s *mainState) setDesktopTrayPet(enabled bool) {
	s.mu.Lock()
	p, quitting := s.desktopPet, s.quitting
	s.mu.Unlock()
	if p == nil || quitting {
		return
	}
	if err := p.SetEnabled(enabled); err != nil {
		s.openPetSettings()
	}
	s.refreshDesktopTray()
}

func (s *mainState) hideToDesktopTray(win *mygo.Window, sup *harness.Supervisor) {
	s.mu.Lock()
	ready := s.desktopTray != nil
	quitting, closeToTray := s.quitting, s.closeToTray
	s.mu.Unlock()
	if desktopCloseDecision(quitting, closeToTray, ready) != desktopCloseHide {
		// Closing was requested as Hide, so never silently reinterpret failure as
		// permission to stop background work. Explicit Quit still uses task checks.
		s.showMainWindow()
		result, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning,
			Title:   i18n.T(i18n.KeyCmdTrayUnavailable),
			Message: i18n.T(i18n.KeyCmdTheSystemTrayCouldNotBeCreatedTheWin),
			Buttons: []string{i18n.T(i18n.KeyCmdKeepWindow), i18n.T(i18n.KeyCmdQuitApplication)}, DefaultButton: 0, CancelButton: 0})
		if err == nil && result.Button == 1 {
			s.requestQuitProtected(win, sup)
		}
		return
	}
	err := s.trayHideNotice.close(win.Hide, func() error {
		if !systemNotificationsSupported() {
			return nil
		}
		n := mygo.NewNotification(mygo.NotificationOptions{ID: "whaleisle-tray-hide", Title: i18n.T(i18n.KeyCmdMinimizedToTray),
			Body: i18n.T(i18n.KeyCmdClickTheTrayIconToReopenTheWindowOr), Silent: true})
		n.OnClick(s.showMainWindow)
		s.mu.Lock()
		s.trayHideNotification = n
		s.mu.Unlock()
		return n.Show()
	})
	if err != nil {
		log.Println("tray hide notice:", err)
	}
}
