package main

import (
	"whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

type accountStatusNoticeState struct {
	store *usage.Store
	seen  usage.NoticeState
}

// syncAccountStatusNotices writes the account's own state where the person is
// working: the conversation's status line. A signed-out account or a model that
// needs a sign-in is a state of the account, not a bar over the page, and each
// sentence names the next step. The Account page carries the same state with the
// button that acts on it.
func (s *mainState) syncAccountStatusNotices() {
	s.mu.Lock()
	store, sessions := s.usage, s.sessions
	s.mu.Unlock()
	if store == nil || sessions == nil {
		return
	}
	notices := store.Notices()
	s.mu.Lock()
	state := &s.accountStatusNotices
	if state.store != store {
		*state = accountStatusNoticeState{store: store}
	}
	previous := state.seen
	state.seen = notices
	s.mu.Unlock()
	if notices.Expired > previous.Expired {
		sessions.SetLastAction(i18n.T(i18n.KeyCmdYouHaveSignedOutOfYourAccountPleaseL))
	}
	if notices.SignInRequired > previous.SignInRequired {
		sessions.SetLastAction(i18n.T(i18n.KeyCmdModelUnavailablePleaseSignInAndTryAga))
	}
}
