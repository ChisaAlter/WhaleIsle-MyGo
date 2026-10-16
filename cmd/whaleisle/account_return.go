package main

import (
	"context"
	"log"
	"strings"

	desktopwebview "whaleisle/internal/desktop/webview"
)

type accountReturnState struct {
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
	if strings.TrimSpace(url) == "" {
		return
	}
	s.mu.Lock()
	s.accountReturn.external, s.accountReturn.leftWindow = true, false
	s.mu.Unlock()
	if err := desktopwebview.OpenExternal(url); err != nil {
		s.mu.Lock()
		s.accountReturn.external, s.accountReturn.leftWindow = false, false
		s.mu.Unlock()
		log.Printf("account: open top-up: %v", err)
	}
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
