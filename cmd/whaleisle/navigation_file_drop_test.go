package main

import (
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func TestNavigationFileDraftScope(t *testing.T) {
	owner := session.New()
	s := &mainState{sessions: owner}
	m := s.navigationFileDropModel(frame.Model{Store: owner})
	if m.Draft == nil || m.OnAttachComposerFiles == nil {
		t.Fatal("chat lacked file target")
	}
	s.marketOpen = true
	if v := s.navigationFileDropModel(frame.Model{Store: owner}); v.Draft != nil || v.OnAttachComposerFiles != nil {
		t.Fatal("market borrowed chat draft")
	}
	// A queued old target cannot insert into a different Host/store owner.
	s.marketOpen = false
	s.sessions = session.New()
	m.OnAttachComposerFiles([]string{"/not-a-real-file"})
	if m.ComposerState.Error != "" || *m.Draft != "" {
		t.Fatal("stale owner executed intake")
	}
}
