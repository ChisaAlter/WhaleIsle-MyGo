package main

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/accountlogin"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

// One account owner follows authorization and hosts sign-out confirmations even
// while Settings is closed or showing another section.
func (s *mainState) syncResidentAccount(c *ui.Context) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	bound := ui.Local(c.Root(), "resident-account-client", func() *dsh.Client { return client })
	if *bound != client {
		s.accountLogin.Finish()
		*bound = client
	}
	s.mu.Lock()
	store, preferences, win, themeID, sessions := s.usage, s.settings, s.win, s.themeID, s.sessions
	s.mu.Unlock()
	if store != nil && client != nil {
		if preferences != nil {
			store.ConfigureBonusAcknowledgements(preferences.NamespaceValue("ui-settings-account"))
		}
		account := store.Snapshot()
		code := account.AttemptErrorCode
		s.accountLogin.Observe(api.AccountView{Status: account.AccountStatus, Attempt: api.AccountViewAttempt{
			Id: account.AttemptID, Phase: account.AttemptPhase, AuthorizeUrl: &account.AuthorizeURL, ErrorCode: &code,
		}}, s.openAccountAuthorizationBrowser)
	}
	login := s.accountLogin.Snapshot()
	reported := ui.Local(c.Root(), "resident-account-login-error", func() string { return "" })
	if login.Error != "" && *reported != login.Error {
		c.AddToast(ui.Toast{ID: "account-signin-error", Description: login.Error})
	}
	*reported = login.Error
	review := s.accountSignOut.Snapshot()
	colors := theme.Resolve(themeID).Active(mygo.Theme.IsDark())
	if store != nil {
		v := store.Snapshot()
		settings.AccountSignInDialog(c, &settings.AccountAccess{Status: v.AccountStatus, ClientReady: v.ClientReady, SignInBusy: login.Busy, AttemptPhase: v.AttemptPhase, AttemptErrorCode: v.AttemptErrorCode, SignInAttemptID: v.AttemptID, AuthorizeURL: v.AuthorizeURL, Err: login.Error, OnSignIn: s.retryAccountSignIn, OnCopyAuthorize: s.copyAccountAuthorization, OnCancelSignIn: func() { s.cancelAccountSignIn(client, v.AttemptID, store, sessions) }}, login.Visible, s.accountLogin.Finish, s.openAccountAPIKeySettings)
	}
	settings.AccountSignOutDialog(c, &settings.AccountAccess{
		SignOut: review,
		OnConfirmSignOut: func() {
			s.runAccountSignOut(win, func(ctx context.Context, meta any) { s.accountSignOut.Confirm(ctx, meta, review.ReviewID) })
		},
		OnCancelSignOut: func() { s.accountSignOut.Cancel(review.ReviewID) },
		OnCheckSignOut: func() {
			s.runAccountSignOut(win, func(ctx context.Context, meta any) { s.accountSignOut.Check(ctx, meta, review.ReviewID) })
		},
	}, chrome.Color(colors.FG), chrome.Color(colors.Muted))
}

func (s *mainState) copyAccountAuthorization(raw string) {
	s.mu.Lock()
	store, win := s.usage, s.win
	s.mu.Unlock()
	if store == nil {
		return
	}
	view := store.Snapshot()
	if !accountlogin.Active(view.AttemptPhase) || raw == "" || raw != view.AuthorizeURL {
		return
	}
	target, err := s.accountAuthorizationURL(raw)
	if err == nil {
		err = mygo.Clipboard.Write(transfer.TextData(target))
	}
	s.reportAccountAction(err)
	message := "Sign-in link copied."
	if i18n.Locale() == i18n.LocaleZH {
		message = "已复制登录链接。"
	}
	if err != nil {
		message = "Could not copy the sign-in link. " + err.Error()
		if i18n.Locale() == i18n.LocaleZH {
			message = "复制登录链接失败：" + err.Error()
		}
	}
	if win != nil {
		s.notifyUser(win, message)
	}
}

func (s *mainState) accountAuthorizationURL(raw string) (string, error) {
	s.mu.Lock()
	themeID := s.themeID
	s.mu.Unlock()
	return accountlogin.AuthorizeURL(raw, theme.ResolveWithSystem(themeID, mygo.Theme.IsDark()).Scheme)
}

func (s *mainState) openAccountAuthorizationBrowser(raw string) error {
	target, err := s.accountAuthorizationURL(raw)
	if err != nil {
		return err
	}
	return desktopwebview.OpenExternal(target)
}
