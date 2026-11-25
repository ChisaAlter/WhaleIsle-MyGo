package main

import "testing"

func TestGlobalSessionNavigationRevealsConversation(t *testing.T) {
	s := &mainState{
		maintenanceOpen: true, settingsOpen: true, usageOpen: true, imOpen: true,
		remoteOpen: true, webviewOpen: true, browserPreviewOpen: true,
		marketOpen: true, scheduleOpen: true,
		settingsEmbeddedSection: "models",
	}
	s.assistantService().SetOpen(true)
	s.launcherController().ResetWorkspaceDraft("unsaved workspace")
	s.marketCoordinator().SetInstallSpec("unsaved package")
	s.revealMainConversation()
	if s.maintenanceOpen || s.settingsOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen || s.marketOpen || s.scheduleOpen || s.assistantService().Open() || s.settingsEmbeddedSection != "" {
		t.Fatal("exclusive page still masks the globally selected conversation")
	}
	if s.launcherController().PanelState(nil, "").Drafts.Workspace != "unsaved workspace" || s.marketCoordinator().ViewSnapshot().Value.InstallSpec != "unsaved package" {
		t.Fatal("route change discarded a resident draft")
	}
}
