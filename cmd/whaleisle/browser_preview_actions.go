package main

import (
	"strings"
	settingsstore "whaleisle/internal/store/settings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/composer"
)

func (s *mainState) newBrowserController(win *mygo.Window) *browserpreviewui.Controller {
	browser := browserpreviewui.NewController(win)
	browser.SetDeveloperTools(func() bool {
		s.mu.Lock()
		store := s.settings
		s.mu.Unlock()
		return store == nil || settingsstore.DeveloperToolsEnabled(store.Snapshot())
	})
	browser.SetActionTarget(s.browserActionTarget)
	browser.SetWorkspaceOwner(s.browserWorkspaceOwner)
	browser.SetPageFailureHandler(s.onEmbeddedPageFailure)
	browser.SetDocumentLoader(s.workspaceDocumentReady, s.readWorkspaceDocument)
	return browser
}

// browserActionTarget runs at the click, not when the browser tab was opened.
// Every callback retains the initiating store, session, window and Host identity.
func (s *mainState) browserActionTarget() browserpreviewui.ActionTarget {
	s.mu.Lock()
	win, sessions, host := s.win, s.sessions, s.bootStatus
	quitting := s.quitting
	s.mu.Unlock()
	if quitting || win == nil || sessions == nil {
		return browserpreviewui.ActionTarget{}
	}
	snap := sessions.Snapshot()
	id, cwd := snap.SelectedID, snap.SelectedCwd
	var draft *composer.DraftState
	if id != "" {
		s.mu.Lock()
		draft = s.composerDraftLocked(id)
		s.mu.Unlock()
	}
	valid := func() bool {
		s.mu.Lock()
		current := !s.quitting && s.win == win && s.sessions == sessions &&
			s.bootStatus.Generation == host.Generation && s.bootStatus.State == host.State &&
			s.bootStatus.Endpoint.Origin == host.Endpoint.Origin &&
			s.bootStatus.Endpoint.Cookie == host.Endpoint.Cookie &&
			(draft == nil || s.composerDrafts[id] == draft)
		s.mu.Unlock()
		if !current || win.IsDestroyed() {
			return false
		}
		currentSession := sessions.Snapshot()
		return currentSession.SelectedID == id && currentSession.SelectedCwd == cwd
	}
	target := browserpreviewui.ActionTarget{Valid: valid}
	if draft == nil || draft.Sending || draft.EditSeq != 0 || draft.EditSessionID != "" {
		return target
	}
	target.Append = func(text string) bool {
		if !valid() || draft.Sending || draft.Preparing > 0 ||
			draft.EditSeq != 0 || draft.EditSessionID != "" ||
			(draft.TargetID != "" && draft.TargetID != id) ||
			draft.Client == nil || strings.TrimSpace(text) == "" {
			return false
		}
		// Do not terminate an in-progress IME composition.
		if _, marked := draft.Client.MarkedRange(); marked {
			return false
		}
		if draft.Text != "" {
			text = "\n\n" + text
		}
		end := ui.UTF16Len(draft.Text)
		draft.Client.ReplaceText(&ui.TextInputRange{Start: end, End: end}, text)
		draft.Error = ""
		// No submit, upload, send, session selection or focus change here.
		return true
	}
	target.AppendAnnotation = func(text, screenshotPath string) bool {
		// This callback is invoked only by the native review button. The path
		// was created by the bounded local capture, never supplied by the page.
		if screenshotPath == "" || !valid() || !target.Append(text) {
			return false
		}
		s.addComposerFiles(win, sessions, id, draft, []string{screenshotPath})
		return true
	}
	return target
}
