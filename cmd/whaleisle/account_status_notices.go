package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/store/usage"
)

type accountStatusNoticeState struct {
	store *usage.Store
	seen  usage.NoticeState
}

// Always drawn at the application root, independently of the Settings panel.
func (s *mainState) syncAccountStatusNotices(c *ui.Context) {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	if store == nil {
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
		c.AddToast(s.accountStatusToast(store, notices, "account-session-expired", resumeText("登录信息已失效，请重新登录", "You have signed out of your account, please log in again.")))
	}
	if notices.SignInRequired > previous.SignInRequired {
		c.AddToast(s.accountStatusToast(store, notices, "account-model-sign-in-required", resumeText("当前模型暂不可用，请登录后再试", "Model unavailable. Please sign in and try again.")))
	}
}

// The action only navigates to the existing account controls. It never starts
// authentication, and a replaced account or Host cannot use an old notice.
func (s *mainState) accountStatusToast(store *usage.Store, notices usage.NoticeState, id, description string) ui.Toast {
	s.mu.Lock()
	client, generation := s.dshClient, s.bootStatus.Generation
	s.mu.Unlock()
	return ui.Toast{ID: id, Description: description, Action: resumeText("前往登录", "Sign in"), OnAction: func() {
		s.mu.Lock()
		current := !s.quitting && s.usage == store && s.dshClient == client && s.bootStatus.Generation == generation
		s.mu.Unlock()
		if !current || store.Notices() != notices {
			return
		}
		s.selectMainPanel("")
		s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "account"})
	}}
}
