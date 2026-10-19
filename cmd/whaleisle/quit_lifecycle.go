package main

import (
	"log"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Claim on the calling thread before scheduling disk/Host work. Repeated close
// requests leave the original save, task inspection and confirmation in charge.
func (s *mainState) requestQuitProtected(win *mygo.Window, sup *harness.Supervisor) {
	s.mu.Lock()
	if s.quitRequestPending || s.quitting {
		s.mu.Unlock()
		return
	}
	s.quitRequestPending = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		did := s.quitProtected(win, sup)
		s.mu.Lock()
		s.quitRequestPending = false
		s.mu.Unlock()
		if did {
			s.quitAfterCleanup()
			return
		}
		if win != nil {
			win.Update(func() {})
		}
	}()
}

// A repeated approved caller may find cleanup already in progress. The owner
// will request the final quit after it finishes; no caller closes windows early.
func (s *mainState) quitAfterCleanup() {
	s.mu.Lock()
	finished := s.quitCleanupFinished
	s.mu.Unlock()
	if finished {
		mygo.App.Quit()
	}
}

// Only an approved protected commit reaches final cleanup. Callers keep slow
// shutdown on their worker; controller/pet state is owned by the UI thread.
func (s *mainState) finalizeQuit(sup *harness.Supervisor) {
	s.mu.Lock()
	if s.quitting {
		s.mu.Unlock()
		return
	}
	s.quitting = true
	s.mu.Unlock()
	s.closeWhaleBridgeAlerts()
	s.closeDesktopTray()
	s.cancelComposerVoice()
	if err := desktopcomponents.ShutdownGeneric(paths.UserDataDir()); err != nil {
		log.Println("Components shutdown:", err)
	}
	mygo.RunOnMain(func() {
		if s.desktopPet != nil {
			s.desktopPet.Close()
		}
	})
	s.stopPhoneRuntime()
	s.closeWebViewHost()
	s.stopDSH()
	s.closeTerminal()
	mygo.RunOnMain(func() {
		if s.browserController != nil {
			s.browserController.Close()
		}
	})
	if sup != nil {
		sup.Stop()
	}
	s.mu.Lock()
	s.quitCleanupFinished = true
	s.mu.Unlock()
}
