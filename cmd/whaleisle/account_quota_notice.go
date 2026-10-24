package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type accountQuotaNoticeState struct {
	store    *session.Store
	id       uint64
	toast    uint64
	opened   bool
	signedIn bool
}

func (s *mainState) dismissAccountQuotaPage() {
	s.mu.Lock()
	state := s.accountQuotaNotice
	s.accountQuotaNotice = accountQuotaNoticeState{}
	s.mu.Unlock()
	if state.store != nil && state.opened {
		state.store.DismissQuotaNotice(state.id)
	}
}

func (s *mainState) closeAccountQuotaToast(c *ui.Context) {
	s.mu.Lock()
	shown := s.accountQuotaNotice.toast != 0
	s.accountQuotaNotice.toast = 0
	s.mu.Unlock()
	if shown {
		c.CloseToast("account-quota-notice")
	}
}

func (s *mainState) syncAccountQuotaNotice(c *ui.Context) {
	s.mu.Lock()
	sessions, store := s.sessions, s.usage
	s.mu.Unlock()
	if sessions == nil || store == nil {
		return
	}
	notice := sessions.QuotaNotice()
	account := store.Snapshot()
	signedIn := account.AccountStatus == "credential-stored"
	s.mu.Lock()
	state := &s.accountQuotaNotice
	if state.store != sessions || state.id != notice.ID {
		old := *state
		*state = accountQuotaNoticeState{store: sessions, id: notice.ID, signedIn: signedIn, toast: old.toast}
		if old.store != nil {
			old.store.ReleaseQuotaNotice(old.id)
		}
	}
	if state.signedIn && !signedIn {
		id := state.id
		shown := state.toast != 0
		if state.opened {
			s.accountReturn.external = false
			s.accountReturn.leftWindow = false
		}
		*state = accountQuotaNoticeState{}
		s.mu.Unlock()
		sessions.DismissQuotaNotice(id)
		if shown {
			c.CloseToast("account-quota-notice")
		}
		return
	}
	state.signedIn = signedIn
	opened := state.opened
	alreadyToasted := state.toast == notice.ID
	platformOpen := s.accountReturn.external || s.accountReturn.embedded != ""
	s.mu.Unlock()
	if notice.ID == 0 {
		s.closeAccountQuotaToast(c)
		return
	}
	// Keep the account claim alive during initial loading, like AccountQuotaNotice.
	if notice.Code == "ACCOUNT_QUOTA" && account.AccountStatus == "" && account.Err == "" && account.ClientReady {
		return
	}
	if notice.Code != "ACCOUNT_QUOTA" || !signedIn || account.TopUpURL == "" {
		if !alreadyToasted {
			s.mu.Lock()
			s.accountQuotaNotice.toast = notice.ID
			s.mu.Unlock()
			c.AddToast(ui.Toast{ID: "account-quota-notice", Description: i18n.T(i18n.KeyCmdRequestQuotaExhausted), OnClose: func() { sessions.DismissQuotaNotice(notice.ID) }})
		}
		return
	}
	s.closeAccountQuotaToast(c)
	if opened || platformOpen {
		return
	}
	s.mu.Lock()
	themeID := s.themeID
	s.mu.Unlock()
	colors := theme.Resolve(themeID).Active(mygo.Theme.IsDark())
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Width(460).MaxWidthPercent(92).Padding(20).Radius(14).Background(chrome.Color(colors.BG)).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyCmdNoBalanceAvailable)).Bold().FontSize(16).TextColor(chrome.Color(colors.FG))
				ui.Text(c, i18n.T(i18n.KeyCmdMygoCannotStartANewTaskWithThisAccou)).TextColor(chrome.Color(colors.FG))
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, i18n.T(i18n.KeyCmdCancel)).Clicked() {
						sessions.DismissQuotaNotice(notice.ID)
					}
					if ui.Button(c, i18n.T(i18n.KeyCmdTopUp)).Clicked() && sessions.HoldQuotaNotice(notice.ID) {
						s.openAccountTopUp(account.TopUpURL)
						s.mu.Lock()
						success := s.accountReturn.err == ""
						if success {
							s.accountQuotaNotice.opened = true
						}
						s.mu.Unlock()
						if !success {
							sessions.ReleaseQuotaNotice(notice.ID)
						}
					}
				})
				s.mu.Lock()
				errorText := s.accountReturn.err
				s.mu.Unlock()
				if errorText != "" {
					ui.Text(c, errorText).TextColor(chrome.Color(colors.FG))
				}
			})
		})
	})
	if !open {
		sessions.DismissQuotaNotice(notice.ID)
	}
}
