package main

import (
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func startupSelectionFixture() (*mainState, *mygo.Window, *session.Store, session.InitialWorkspaceSession) {
	win := &mygo.Window{}
	store := session.New()
	s := &mainState{win: win, sessions: store, composerDrafts: map[string]*composer.DraftState{"": composer.NewDraftState("")}}
	return s, win, store, session.InitialWorkspaceSession{Binding: 4, HostGeneration: 9, WorkspaceID: "default"}
}

func TestStartupSelectionWaitsThenConsumesOnlyOneInitialOpportunity(t *testing.T) {
	s, win, store, receipt := startupSelectionFixture()
	if s.claimStartupSelection(win, store, receipt) {
		t.Fatal("claimed without authority")
	}
	receipt.Ready = true
	if !s.claimStartupSelection(win, store, receipt) {
		t.Fatal("fresh empty workspace did not claim")
	}
	for _, change := range []string{"same", "events-reconnect", "navigation-token"} {
		if change == "events-reconnect" {
			receipt.HostGeneration++
		}
		if change == "navigation-token" {
			receipt.NavigationToken++
		}
		if s.claimStartupSelection(win, store, receipt) {
			t.Fatalf("%s rearmed startup", change)
		}
	}
	receipt.Binding++
	if !s.claimStartupSelection(win, store, receipt) {
		t.Fatal("new attached Host could not initialize")
	}
}

func TestStartupSelectionPreservesDraftsAndDoesNotCreateAfterErase(t *testing.T) {
	for _, kind := range []string{"text", "attachments", "editing", "sending", "preparing", "reply", "typed-and-erased", "other-draft"} {
		t.Run(kind, func(t *testing.T) {
			s, win, store, receipt := startupSelectionFixture()
			draft := s.composerDrafts[""]
			switch kind {
			case "text":
				draft.Text = "keep my draft"
			case "attachments":
				draft.Attachments = []session.ComposerAttachment{{Path: "own.png"}}
			case "editing":
				draft.EditSeq = 3
			case "sending":
				draft.Sending = true
			case "preparing":
				draft.Preparing = 1
			case "reply":
				draft.GroupReply = &composer.GroupReplyDraft{ThreadID: "reply"}
			case "typed-and-erased":
				draft.Text = "typed"
				draft.Client.Sync()
				draft.Text = ""
				draft.Client.Sync()
			case "other-draft":
				s.composerDrafts["other"] = composer.NewDraftState("other remains")
			}
			before := string(draft.SaveDraft())
			if s.claimStartupSelection(win, store, receipt) || string(draft.SaveDraft()) != before {
				t.Fatal("pending startup changed a draft")
			}
			s.composerDrafts = map[string]*composer.DraftState{"": composer.NewDraftState("")}
			receipt.Ready = true
			if s.claimStartupSelection(win, store, receipt) {
				t.Fatal("erasing local draft rearmed delayed creation")
			}
		})
	}
}

func TestStartupSelectionRetiresOnRoutesNavigationAndHostRestart(t *testing.T) {
	for _, route := range []string{"market", "schedule", "assistant", "settings", "launcher", "usage", "im", "remote", "webview", "browser", "quitting", "panel-aba", "gesture", "boot"} {
		t.Run(route, func(t *testing.T) {
			s, win, store, receipt := startupSelectionFixture()
			if s.claimStartupSelection(win, store, receipt) {
				t.Fatal("not ready")
			}
			switch route {
			case "market":
				s.marketOpen = true
			case "schedule":
				s.scheduleOpen = true
			case "assistant":
				s.assistantOpen = true
			case "settings":
				s.settingsOpen = true
			case "launcher":
				s.launcherOpen = true
			case "usage":
				s.usageOpen = true
			case "im":
				s.imOpen = true
			case "remote":
				s.remoteOpen = true
			case "webview":
				s.webviewOpen = true
			case "browser":
				s.browserPreviewOpen = true
			case "quitting":
				s.quitting = true
			case "panel-aba":
				s.assistantConversationGeneration += 2
			case "gesture":
				s.workspaceSessionOpen.generation++
			case "boot":
				s.bootStatus.Generation++
			}
			receipt.Ready = true
			if s.claimStartupSelection(win, store, receipt) {
				t.Fatal("startup stole foreground")
			}
			s.marketOpen, s.scheduleOpen, s.assistantOpen, s.settingsOpen, s.launcherOpen, s.usageOpen, s.imOpen, s.remoteOpen, s.webviewOpen, s.browserPreviewOpen, s.quitting = false, false, false, false, false, false, false, false, false, false, false
			if s.claimStartupSelection(win, store, receipt) {
				t.Fatal("returning to conversation rearmed startup")
			}
		})
	}
}
