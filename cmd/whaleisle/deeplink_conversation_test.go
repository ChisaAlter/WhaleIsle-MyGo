package main

import (
	"testing"
	"whaleisle/internal/desktop/deeplink"
)

func TestConversationDeepLinksLeaveToolPages(t *testing.T) {
	for _, action := range []deeplink.Action{deeplink.ActionSession, deeplink.ActionAgents} {
		t.Run(string(action), func(t *testing.T) {
			s := mainState{imOpen: true, remoteOpen: true, marketOpen: true,
				webviewOpen: true, browserPreviewOpen: true, assistantOpen: true,
				toolsMenuOpen: true, launcherOpen: true, settingsOpen: true,
				usageOpen: true, scheduleOpen: true}
			s.applyDeeplink(deeplink.Target{Action: action, SessionID: "requested-session"})
			if s.imOpen || s.remoteOpen || s.marketOpen || s.webviewOpen || s.browserPreviewOpen || s.assistantOpen || s.toolsMenuOpen || s.launcherOpen || s.settingsOpen || s.usageOpen || s.scheduleOpen {
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
		webviewOpen: true, browserPreviewOpen: true, assistantOpen: true}
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionOpen})
	if !s.imOpen || !s.remoteOpen || !s.marketOpen || !s.webviewOpen || !s.browserPreviewOpen || !s.assistantOpen {
		t.Fatal("raising the window unexpectedly navigated away")
	}
}
