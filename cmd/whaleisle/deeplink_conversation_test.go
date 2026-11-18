package main

import (
	"testing"
	"whaleisle/internal/desktop/deeplink"
)

func TestConversationDeepLinksLeaveToolPages(t *testing.T) {
	for _, action := range []deeplink.Action{deeplink.ActionSession, deeplink.ActionAgents} {
		t.Run(string(action), func(t *testing.T) {
			s := mainState{imOpen: true, remoteOpen: true, marketOpen: true,
				webviewOpen: true, browserPreviewOpen: true,
				toolsMenuOpen: true, maintenanceOpen: true, settingsOpen: true,
				usageOpen: true, scheduleOpen: true}
			s.assistantService().SetOpen(true)
			s.applyDeeplink(deeplink.Target{Action: action, SessionID: "requested-session"})
			if s.imOpen || s.remoteOpen || s.marketOpen || s.webviewOpen || s.browserPreviewOpen || s.assistantService().Open() || s.toolsMenuOpen || s.maintenanceOpen || s.settingsOpen || s.usageOpen || s.scheduleOpen {
				t.Fatal("conversation target remains hidden by a previous page")
			}
			if action == deeplink.ActionAgents && (!s.filesPaneOpen || s.rightPaneTab != "agents") {
				t.Fatal("agent target did not reveal the agents pane")
			}
		})
	}
}

func TestPlainOpenLinkKeepsCurrentToolPage(t *testing.T) {
	s := mainState{imOpen: true, remoteOpen: true, marketOpen: true,
		webviewOpen: true, browserPreviewOpen: true}
	s.assistantService().SetOpen(true)
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionOpen})
	if !s.imOpen || !s.remoteOpen || !s.marketOpen || !s.webviewOpen || !s.browserPreviewOpen || !s.assistantService().Open() {
		t.Fatal("raising the window unexpectedly navigated away")
	}
}
