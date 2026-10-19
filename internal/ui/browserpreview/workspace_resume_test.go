package browserpreview

import (
	"errors"
	"testing"
)

func TestBrowserRecoveryDoesNotNavigateAndKeepsCheckpointOnRestoreError(t *testing.T) {
	owner := WorkspaceOwner{Authority: "fixture", SessionID: "session", Ready: true, Cwd: "/project"}
	scope, err := makeBrowserWorkspace(owner)
	if err != nil {
		t.Fatal(err)
	}
	saved := &WorkspaceSnapshot{Tabs: []WorkspaceTabSnapshot{{ID: 7, URL: "https://example.invalid/original", Draft: "https://example.invalid/unsubmitted", Partition: scope.key, Label: scope.label}}, Active: 12}
	b := NewController(nil)
	b.SetWorkspaceOwner(func() WorkspaceOwner { return owner })
	b.RestoreWorkspace(saved)
	tab := b.current()
	if tab == nil || b.Active != 0 || tab.Host != nil || tab.resumeURL != saved.Tabs[0].URL || tab.Draft != saved.Tabs[0].Draft {
		t.Fatalf("recovery created navigation/lost checkpoint: %+v", tab)
	}
	b.Select(0)
	b.resumeCurrent()
	if tab.Host != nil || tab.resumeURL != saved.Tabs[0].URL {
		t.Fatal("displaying restored tab navigated")
	}
	owner.Err = errors.New("workspace list unavailable")
	b.restoreBrowserAddress(tab)
	if tab.Host != nil || tab.ActionMessage == "" || tab.resumeURL != saved.Tabs[0].URL || tab.Draft != saved.Tabs[0].Draft {
		t.Fatal("restore failure consumed checkpoint/draft")
	}
	owner.Err = nil
	owner.Cwd = "/different-project"
	b.restoreBrowserAddress(tab)
	if len(b.Tabs) != 1 || tab.Host != nil || tab.workspace.key != scope.key || tab.resumeURL != saved.Tabs[0].URL {
		t.Fatal("restore repurposed a foreign profile")
	}
	checkpoint := b.WorkspaceSnapshot()
	if checkpoint.Tabs[0].URL != saved.Tabs[0].URL || checkpoint.Tabs[0].Draft != saved.Tabs[0].Draft || checkpoint.Tabs[0].Partition != scope.key {
		t.Fatalf("retry checkpoint lost: %+v", checkpoint)
	}
}
