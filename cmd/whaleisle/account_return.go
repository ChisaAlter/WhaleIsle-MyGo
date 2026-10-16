package main

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/usage"

	desktopwebview "whaleisle/internal/desktop/webview"
)

type accountReturnState struct {
	err                string
	external           bool
	leftWindow         bool
	embedded           string
	embeddedGeneration int
}

func (s *mainState) refreshReturnedAccount() {
	s.mu.Lock()
	usage, sessions := s.usage, s.sessions
	s.mu.Unlock()
	if usage == nil {
		return
	}
	sid := ""
	if sessions != nil {
		sid = sessions.Snapshot().SelectedID
	}
	// A new account epoch prevents a pre-return profile/wallet request from
	// replacing this post-operation read when it eventually completes.
	usage.Refresh(context.Background(), sid)
}

func (s *mainState) openAccountTopUp(url string) {
	s.openAccountLink(url, true, desktopwebview.OpenExternal)
}
func (s *mainState) openAccountAuthorization(url string) {
	s.openAccountLink(url, false, desktopwebview.OpenExternal)
}
func (s *mainState) reportAccountAction(err error) {
	s.mu.Lock()
	s.accountReturn.err = ""
	if err != nil {
		s.accountReturn.err = err.Error()
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) openAccountLink(url string, topUp bool, open func(string) error) {
	if strings.TrimSpace(url) == "" {
		return
	}
	s.mu.Lock()
	if topUp {
		s.accountReturn.external, s.accountReturn.leftWindow = true, false
	}
	s.mu.Unlock()
	err := open(url)
	if err != nil {
		s.mu.Lock()
		if topUp {
			s.accountReturn.external, s.accountReturn.leftWindow = false, false
		}
		s.mu.Unlock()
		label := resumeText("无法打开授权页面", "Could not open authorization page")
		if topUp {
			label = resumeText("无法打开充值页面", "Could not open top-up page")
		}
		err = fmt.Errorf("%s: %w", label, err)
	}
	s.reportAccountAction(err)
}

func (s *mainState) cancelAccountSignIn(client *dsh.Client, attemptID string, usageStore *usage.Store, sessions *session.Store) {
	go func() {
		var out api.AccountView
		err := client.Call(context.Background(), "account/cancelSignIn", map[string]any{"attemptId": attemptID}, &out)
		s.dshMu.Lock()
		if s.dshClient != client {
			s.dshMu.Unlock()
			return
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", resumeText("取消登录失败", "Could not cancel sign-in"), err)
		} else {
			s.accountLogin.Finish()
		}
		s.reportAccountAction(err)
		s.dshMu.Unlock()
		if usageStore != nil {
			sid := ""
			if sessions != nil {
				sid = sessions.Snapshot().SelectedID
			}
			usageStore.Refresh(context.Background(), sid)
		}
	}()
}

func (s *mainState) accountWindowBlurred() {
	s.mu.Lock()
	if s.accountReturn.external {
		s.accountReturn.leftWindow = true
	}
	s.mu.Unlock()
}

func (s *mainState) returnFromAccountExternal() {
	s.mu.Lock()
	returned := s.accountReturn.external && s.accountReturn.leftWindow
	if returned {
		s.accountReturn.external, s.accountReturn.leftWindow = false, false
	}
	s.mu.Unlock()
	if returned {
		s.refreshReturnedAccount()
	}
}

func (s *mainState) openAccountUsage(url, source string) {
	if strings.TrimSpace(url) == "" {
		return
	}
	s.mu.Lock()
	s.settingsOpen, s.usageOpen = false, false
	s.webviewOpen = true
	s.assistantConversationGeneration++
	s.assistantOpen = false
	s.accountReturn.embedded = ""
	win := s.win
	s.mu.Unlock()
	s.openWebViewURL(url)
	s.mu.Lock()
	if s.webviewHost != nil && s.webviewHostErr == "" {
		s.accountReturn.embedded = source
		s.accountReturn.embeddedGeneration = s.bootStatus.Generation
	}
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) returnFromAccountPage() {
	s.mu.Lock()
	source := s.accountReturn.embedded
	if s.accountReturn.embeddedGeneration != s.bootStatus.Generation {
		source = ""
	}
	s.accountReturn.embedded = ""
	if source == "settings" {
		s.settingsOpen = true
		s.settingsAccountActive = true
	} else if source == "usage" {
		s.usageOpen = true
	}
	s.mu.Unlock()
	if source != "" {
		s.refreshReturnedAccount()
	}
}

func (s *mainState) clearAccountReturn() {
	s.mu.Lock()
	s.accountReturn = accountReturnState{}
	s.mu.Unlock()
}
