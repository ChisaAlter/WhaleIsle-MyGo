package main

import "testing"

func TestGlobalSessionNavigationRevealsConversation(t *testing.T) {
	s := &mainState{
		launcherOpen: true, settingsOpen: true, usageOpen: true, imOpen: true,
		remoteOpen: true, webviewOpen: true, browserPreviewOpen: true,
		marketOpen: true, scheduleOpen: true, assistantOpen: true, toolsMenuOpen: true,
		settingsEmbeddedSection: "models", launcherWorkspaceDraft: "unsaved workspace",
		marketInstallSpec: "unsaved package",
	}
	s.revealMainConversation()
	if s.launcherOpen || s.settingsOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen || s.marketOpen || s.scheduleOpen || s.assistantOpen || s.toolsMenuOpen || s.settingsEmbeddedSection != "" {
		t.Fatal("exclusive page still masks the globally selected conversation")
	}
	if s.launcherWorkspaceDraft != "unsaved workspace" || s.marketInstallSpec != "unsaved package" {
		t.Fatal("route change discarded a resident draft")
	}
}
