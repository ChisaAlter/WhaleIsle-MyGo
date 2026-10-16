package main

import (
	"github.com/egoist/mygo/ui"
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
		c.AddToast(ui.Toast{ID: "account-session-expired", Description: resumeText("登录信息已失效，请重新登录", "You have signed out of your account, please log in again.")})
	}
	if notices.SignInRequired > previous.SignInRequired {
		c.AddToast(ui.Toast{ID: "account-model-sign-in-required", Description: resumeText("当前模型暂不可用，请登录后再试", "Model unavailable. Please sign in and try again.")})
	}
}
