package usage

import (
	"context"
	"whaleisle/internal/dsh"
)

// NoticeState contains event revisions, not bearer/expiry credential material.
type NoticeState struct{ Expired, SignInRequired, Generation uint64 }

// OnAccountNotice belongs to the existing events stream; canceled generations
// cannot publish after account/Host replacement. It never starts sign-in.
func (s *Store) OnAccountNotice(ctx context.Context, client *dsh.Client, emit dsh.EventsEmit) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.client != client || client == nil || ctx.Err() != nil {
		s.unlockPublish()
		return
	}
	switch emit.Event {
	case "deepseek-account/session-expired":
		s.notices.Expired++
	case "deepseek-account/model-sign-in-required":
		s.notices.SignInRequired++
	default:
		s.unlockPublish()
		return
	}
	s.unlockPublish()
	s.fire()
}
