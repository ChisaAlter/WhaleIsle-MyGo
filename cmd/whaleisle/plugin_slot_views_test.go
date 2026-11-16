package main

import (
	"testing"

	"whaleisle/internal/harness"
)

func TestSelectMainPanelAndRevealClearActivePluginWebViewState(t *testing.T) {
	s := &mainState{
		webviewOpen:           true,
		activePluginViewLabel: "QA Panel",
		bootStatus:            harness.Status{State: harness.Ready},
	}
	s.selectMainPanel("plugins")
	if s.webviewOpen || s.activePluginViewLabel != "" {
		t.Fatalf("selectMainPanel left webview state dirty: open=%v label=%q", s.webviewOpen, s.activePluginViewLabel)
	}

	s.webviewOpen = true
	s.activePluginViewLabel = "QA Panel"
	s.revealMainConversation()
	if s.webviewOpen || s.activePluginViewLabel != "" {
		t.Fatalf("revealMainConversation left webview state dirty: open=%v label=%q", s.webviewOpen, s.activePluginViewLabel)
	}
}
