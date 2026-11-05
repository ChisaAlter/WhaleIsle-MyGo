package main

import (
	"context"

	"github.com/egoist/mygo"
	"whaleisle/internal/accountlogin"
	"whaleisle/internal/accountlogout"
)

// Only account-page clicks call this flow. It has no application shutdown,
// Host-stop, task-cancellation, settings-save, or credential-file side effects.
func (s *mainState) runAccountSignOut(win *mygo.Window, action func(context.Context, any)) {
	accountlogin.SetClientVersion(buildVersion)
	metadata := accountlogin.ClientMetadata()
	go func() {
		action(context.Background(), metadata)
		if s.accountSignOut.Snapshot().Completed {
			s.mu.Lock()
			store, sessions := s.usage, s.sessions
			s.mu.Unlock()
			if store != nil {
				sid := ""
				if sessions != nil {
					sid = sessions.SelectedSessionID()
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
