package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktopcomponents "whaleisle/internal/desktop/components"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
	// One milestone per user request, so a cancelled quit is still visible to the
	// next run's update evidence (update-journal.js quit-requested).
	s.recordUpdateAction(desktopupdate.JournalActionQuitRequested)
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
	win := s.win
	cancelPreparation := s.launcherStartCancel
	s.mu.Unlock()
	if cancelPreparation != nil {
		cancelPreparation()
	}
	s.closeWhaleBridgeAlerts()
	// The management window is a separate top-level window; quitting must not
	// leave it behind its own webview child.
	s.closeWhaleBridgeWindow()
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
	// Native order (index.js:1434-1437): the window shows the closing layer
	// before the local Harness service stops, and the paint can never hold the
	// quit.
	s.stopWithClosingOverlay(win, sup)
	s.mu.Lock()
	s.quitCleanupFinished = true
	s.mu.Unlock()
}

// closingOverlayPaintBudget bounds one closing-layer paint. It is the native
// host deadline around insertCSS/executeJavaScript
// (reference/electron/src/main/closing-overlay.js:142-157): a suspended UI
// thread or a gone renderer must never hold the quit.
const closingOverlayPaintBudget = 500 * time.Millisecond

// paintClosingOverlayLayer paints the closing layer from the window's view.
// mainView defers it first so it runs last and covers chat, dialogs, settings
// and the launcher; without a quit in progress it paints nothing. colors is the
// palette that frame resolved, so the layer matches the window it covers.
func (s *mainState) paintClosingOverlayLayer(c *ui.Context, colors theme.Colors) {
	s.mu.Lock()
	closing := s.closingOverlay
	s.mu.Unlock()
	if !closing {
		return
	}
	chrome.DrawClosingOverlay(c, chrome.ClosingOverlay{
		Title:  i18n.T(i18n.KeyClosingTitle),
		Detail: i18n.T(i18n.KeyClosingDetail),
		Colors: colors,
	})
}

// closingOverlayFrameRequest posts one closing-layer frame to the window's UI
// thread; done reports that the thread built it. Restore, show and focus run on
// that thread as well, so a window that never answers cannot block the caller
// before the deadline. A nil window has no surface and no request.
func closingOverlayFrameRequest(win *mygo.Window) func(done func()) {
	if win == nil {
		return nil
	}
	return func(done func()) {
		win.Update(func() {
			// A quit from the tray surfaces the window first, as the native
			// showClosingOverlay restores, shows and focuses before painting.
			if win.IsMinimized() {
				win.Restore()
			}
			if !win.IsVisible() {
				win.Show()
			}
			win.Focus()
			// Queued behind this frame's request: the UI thread reached the
			// closing state and asked for the repaint.
			win.Update(done)
		})
	}
}

// waitForClosingFrame runs request and waits at most budget for its frame. The
// request is posted from its own goroutine, so neither a wedged UI thread nor a
// post that never returns can hold the caller past the deadline. False means no
// frame arrived in time; shutdown proceeds regardless.
func waitForClosingFrame(request func(done func()), budget time.Duration) bool {
	if request == nil || budget <= 0 {
		return false
	}
	done := make(chan struct{}, 1)
	go request(func() {
		select {
		case done <- struct{}{}:
		default:
		}
	})
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// showClosingOverlayForQuit marks the closing layer and waits for one painted
// frame. The mark is in place before the first wait, so a frame that arrives
// late still shows the layer.
func (s *mainState) showClosingOverlayForQuit(request func(done func()), budget time.Duration) bool {
	s.mu.Lock()
	s.closingOverlay = true
	s.mu.Unlock()
	return waitForClosingFrame(request, budget)
}

// overlayThenStop runs the closing layer and then the shutdown step, whatever
// the layer did: a missing window, a suspended UI thread or a failed paint.
func overlayThenStop(overlay, stop func()) {
	if overlay != nil {
		overlay()
	}
	if stop != nil {
		stop()
	}
}

// stopWithClosingOverlay is the native quit order: surface and paint the
// closing layer, then stop the local Harness service. The paint is best effort
// and bounded, so shutdown always continues.
func (s *mainState) stopWithClosingOverlay(win *mygo.Window, sup *harness.Supervisor) {
	request := closingOverlayFrameRequest(win)
	overlayThenStop(func() {
		s.showClosingOverlayForQuit(request, closingOverlayPaintBudget)
	}, func() {
		if sup != nil {
			sup.Stop()
		}
	})
}
