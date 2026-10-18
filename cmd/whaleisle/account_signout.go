package main

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/accountlogout"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

// Only account-page clicks call this flow. It has no application shutdown,
// Host-stop, task-cancellation, settings-save, or credential-file side effects.
func (s *mainState) runAccountSignOut(win *mygo.Window, action func(context.Context, any)) {
	_, offset := time.Now().Zone()
	locale := "zh-CN"
	if i18n.Locale() == i18n.LocaleEN {
		locale = "en-US"
	}
	metadata := api.AccountClientMetadata{Version: "whaleisle", Locale: locale, TimezoneOffsetSeconds: float64(offset)}
	go func() {
		action(context.Background(), metadata)
		if s.accountSignOut.Snapshot().Completed {
			s.mu.Lock()
			store, sessions := s.usage, s.sessions
			s.mu.Unlock()
			if store != nil {
				sid := ""
				if sessions != nil {
					sid = sessions.Snapshot().SelectedID
				}
				store.Refresh(context.Background(), sid)
			}
		}
		if win != nil {
			win.Invalidate()
		}
	}()
}

func (s *mainState) accountSignOutAccess() accountlogout.State { return s.accountSignOut.Snapshot() }
