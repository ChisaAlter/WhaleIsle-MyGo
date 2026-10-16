package usage

import (
	"context"
	"whaleisle/internal/dsh"
)

// NoticeState contains event revisions, not bearer/expiry credential material.
type NoticeState struct{ Expired, SignInRequired uint64 }

func (s *Store) Notices() NoticeState { s.mu.Lock(); defer s.mu.Unlock(); return s.notices }

// OnAccountNotice belongs to the existing events stream; canceled generations
// cannot publish after account/Host replacement. It never starts sign-in.
func (s *Store) OnAccountNotice(ctx context.Context, client *dsh.Client, emit dsh.EventsEmit) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.client != client || client == nil || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	switch emit.Event {
	case "deepseek-account/session-expired":
		s.notices.Expired++
	case "deepseek-account/model-sign-in-required":
		s.notices.SignInRequired++
	default:
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	s.fire()
}
