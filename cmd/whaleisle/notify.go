package main

import (
	"fmt"
	"log"
	"strings"

	whaleislecomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
)

// notifyUser answers a user action that cannot run, or reports what it did, in
// the conversation's status line: the shell's own inline status, which stays
// until another action speaks. It never queues a toast, so a refusal cannot
// appear as a bar over the content the person is reading.
//
// Before the shell owns a session store there is no status line to write to, so
// the message joins the startup log the diagnostics export carries.
func (s *mainState) notifyUser(win *mygo.Window, text string) {
	if s == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.mu.Lock()
	sessions := s.sessions
	if sessions == nil {
		s.appendBootLogLocked(text)
	}
	s.mu.Unlock()
	if sessions != nil {
		sessions.SetLastAction(text)
		return
	}
	if win != nil {
		win.Update(func() {})
	}
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

// revealInFileManager reports a failed reveal instead of leaving the menu click
// silent. The path is in the message because it is what the user can act on.
func (s *mainState) revealInFileManager(win *mygo.Window, path string) {
	if err := whaleislecomponents.OpenInFileManager(path); err != nil {
		s.notifyUser(win, fmt.Sprintf(i18n.T(i18n.KeyCmdRevealInFileManagerFailed), path))
	}
}

// saveFailedToast names the action that failed and the cause a person can act
// on. The Host's own wording goes to the log, never into the sentence the page
// shows.
func saveFailedToast(action string, err error) string {
	if err != nil {
		log.Printf("%s: %v", action, err)
	}
	return errcopy.Under(action+i18n.T(i18n.KeyCmdFailed), err)
}
