package main

import (
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// notifyUser queues a one-shot toast for the next painted frame. User actions
// that cannot run because this Host has no store yet call this instead of
// returning silently, so the click is always answered somewhere visible.
// Background timers must not call it: they are not the user asking.
func (s *mainState) notifyUser(win *mygo.Window, text string) {
	if s == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.mu.Lock()
	s.pendingToast = text
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

// takePendingToast returns the queued toast exactly once. mainView calls it at
// the start of every frame, so a burst of clicks shows the latest message and
// nothing repeats on the following frames.
func (s *mainState) takePendingToast() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	text := s.pendingToast
	s.pendingToast = ""
	s.mu.Unlock()
	return text
}

// refuseHostAction answers a user click that cannot run because this Host has
// no store yet. Background timers never call it.
func (s *mainState) refuseHostAction() {
	if s == nil {
		return
	}
	s.notifyUser(s.win, hostNotReadyToast())
}

// hostNotReadyToast is the shared wording for "no Host yet" refusals.
func hostNotReadyToast() string {
	return i18n.T(i18n.KeyCmdNotConnectedToTheHostYetTryAgainShor)
}

// saveFailedToast names the action that failed and keeps the Host's reason.
func saveFailedToast(action string, err error) string {
	reason := ""
	if err != nil {
		reason = strings.TrimSpace(err.Error())
	}
	if reason == "" {
		return action + i18n.T(i18n.KeyCmdFailed)
	}
	return action + i18n.T(i18n.KeyCmdFailed) + reason
}

// consumePendingToast paints the queued message once, before anything else in
// the frame draws.
func consumePendingToast(c *ui.Context) {
	if c == nil {
		return
	}
	if text := state.takePendingToast(); text != "" {
		c.Toast(text)
	}
}
