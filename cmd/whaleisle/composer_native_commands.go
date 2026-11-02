package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) retireNativeComposerCommands() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, draft := range s.composerDrafts {
		draft.RetireNativeCommands()
	}
}

// Both command actions reuse their existing owner: the draft's ordinary file
// intake, or the selected session's feedback controller. No Host command or
// model prompt is sent by a bare local action.
func (s *mainState) nativeComposerCommandAction(win *mygo.Window, sessions *session.Store, draft *composer.DraftState, sourceID, cwd string, owner composer.NativeCommandOwner, name string) bool {
	if sessions == nil || draft == nil || sourceID != owner.SessionID ||
		!sessions.NativeCommandCurrent(owner, name) || draft.EditSeq > 0 || draft.Sending ||
		(draft.TargetID != "" && draft.TargetID != sourceID) {
		return false
	}
	s.mu.Lock()
	current := s.sessions == sessions && s.composerDrafts[sourceID] == draft
	s.mu.Unlock()
	if !current {
		return false
	}
	if isRoom, _ := s.assistantGroupMembers(sourceID); isRoom {
		return false
	}
	switch name {
	case "file":
		if draft.Preparing > 0 {
			return false
		}
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdAttachFilesImages), DefaultPath: cwd, Multiple: true})
		s.mu.Lock()
		current = s.sessions == sessions && s.composerDrafts[sourceID] == draft
		s.mu.Unlock()
		if !current || !sessions.NativeCommandCurrent(owner, name) {
			return false
		}
		if err != nil {
			draft.Error = err.Error()
			return true
		}
		s.addComposerFiles(win, sessions, sourceID, draft, paths)
		if win != nil {
			win.Update(func() {})
		}
		return true
	case "feedback":
		return sessions.OpenSessionFeedback(sourceID)
	default:
		return false
	}
}
