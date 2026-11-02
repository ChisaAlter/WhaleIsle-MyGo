package main

import (
	"context"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
)

// startupSelectionState belongs to one attached Host, not an events reconnect.
// A user draft, route or navigation permanently retires that initial opportunity.
type startupSelectionState struct {
	store              *session.Store
	binding            uint64
	boot               int
	panel, gesture     uint64
	observed, consumed bool
}

func (s *mainState) startupSelectionAllowedLocked() bool {
	if s.quitting || s.marketOpen || s.scheduleOpen || s.assistantOpen || s.settingsOpen || s.launcherOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen {
		return false
	}
	for _, draft := range s.composerDrafts {
		if workspaceSessionDraftOccupied(draft) || workspaceSessionDraftRevision(draft) != 0 {
			return false
		}
	}
	return true
}

// claimStartupSelection is called only on the UI owner. Even a blocked draft or
// route consumes this binding before it can be cleared by a later refresh.
func (s *mainState) claimStartupSelection(win *mygo.Window, sessions *session.Store, receipt session.InitialWorkspaceSession) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions != sessions || s.win != win || receipt.Binding == 0 {
		return false
	}
	initial := &s.startupSelection
	if !initial.observed || initial.store != sessions || initial.binding != receipt.Binding {
		*initial = startupSelectionState{store: sessions, binding: receipt.Binding, boot: s.bootStatus.Generation, panel: s.assistantConversationGeneration, gesture: s.workspaceSessionOpen.generation, observed: true}
	}
	if initial.consumed {
		return false
	}
	if initial.boot != s.bootStatus.Generation || initial.panel != s.assistantConversationGeneration || initial.gesture != s.workspaceSessionOpen.generation || !s.startupSelectionAllowedLocked() {
		initial.consumed = true
		return false
	}
	if !receipt.Ready {
		return false
	}
	initial.consumed = true
	return true
}

// maybeStartupBlankSession runs from Window.Update, never from mainView while
// mainState.mu is held. The existing capture/finish transaction owns all RPC and
// draft/navigation fences; this coordinator merely supplies its one-time trigger.
func (s *mainState) maybeStartupBlankSession(win *mygo.Window, sessions *session.Store) {
	if s == nil || win == nil || sessions == nil {
		return
	}
	receipt := sessions.InitialWorkspaceSession()
	if !s.claimStartupSelection(win, sessions, receipt) {
		return
	}
	op, err := s.captureWorkspaceSessionOpen(win, sessions, receipt.HostGeneration, receipt.WorkspaceID)
	if err != nil || op == nil || !op.current() {
		return
	}
	go func() {
		id, err := op.target.Create(context.Background())
		if !win.IsDestroyed() {
			win.Update(func() { op.finish(id, err) })
		}
	}()
}
