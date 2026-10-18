package main

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/recovery"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
)

// State-only fixtures: no Host, GUI, service, shell or PTY is started.
func TestWorkspaceSessionLayoutsFixture(t *testing.T) {
	s := &mainState{layout: frame.DefaultLayout()}
	s.switchWorkspaceSessionLocked("host-a", "/project", "one")
	first := s.layout.Workspace
	s.filesPaneOpen = true
	s.layout.Right = 620
	s.selectedFile, s.filePreview, s.fileEditDraft = "draft.go", "saved", "unsaved"
	s.fileEditing = true
	s.fileTabs.Workspaces = map[string]*files.WorkspaceTabs{"/project": {Active: "draft.go", Open: []*files.FileTab{{Path: "draft.go", Preview: "saved", Draft: "unsaved"}}}}
	first.Fullscreen = first.Active
	first.Root.Float = &frame.WorkspaceFloat{X: 44, Y: 55, Width: 410, Height: 310}
	s.switchWorkspaceSessionLocked("host-a", "/project", "two")
	second := s.layout.Workspace
	if second == first || s.selectedFile != "" || len(s.fileTabs.Workspaces) != 0 || s.filesPaneOpen || second.Fullscreen != "" {
		t.Fatal("new session inherited the previous resource layout or file adapters")
	}
	s.selectedFile, s.fileEditDraft = "second.txt", "second draft"
	s.switchWorkspaceSessionLocked("host-a", "/project", "one")
	if s.layout.Workspace != first || s.selectedFile != "draft.go" || s.fileEditDraft != "unsaved" || !s.filesPaneOpen || s.layout.Right != 620 {
		t.Fatal("same-session resource state did not resume")
	}
	if first.Root.Float == nil || first.Root.Float.X != 44 || first.Fullscreen != first.Active {
		t.Fatal("floating geometry/fullscreen did not remain session-owned")
	}
	s.switchWorkspaceSessionLocked("host-b", "/project", "one")
	if s.layout.Workspace == first || s.selectedFile != "" {
		t.Fatal("same path/session on another Host reused content")
	}
	s.switchWorkspaceSessionLocked("host-a", "/project", "one")
	raw := s.exportWorkspaceSessionsLocked()
	restarted := &mainState{layout: frame.DefaultLayout()}
	if err := restarted.restoreWorkspaceSessions(recovery.State{Layout: recovery.Layout{WorkspacePanes: raw}}); err != nil {
		t.Fatal(err)
	}
	restarted.switchWorkspaceSessionLocked("host-a", "/project", "one")
	if restarted.fileEditDraft != "unsaved" || restarted.selectedFile != "draft.go" || restarted.layout.Right != 620 {
		t.Fatal("durable session restore lost editor state")
	}
	open := restarted.fileTabs.Workspaces["/project"].Open
	if len(open) != 1 || open[0].Draft != "unsaved" || !open[0].Dirty() {
		t.Fatal("unsaved file buffer lost")
	}
	restarted.switchWorkspaceSessionLocked("host-a", "/project", "two")
	if restarted.fileEditDraft != "second draft" || restarted.selectedFile != "second.txt" {
		t.Fatal("inactive session was not persisted independently")
	}
	if second == restarted.layout.Workspace {
		t.Fatal("restart unexpectedly reused runtime resources")
	}
}

func TestWorkspaceSessionLegacyMigrationFixture(t *testing.T) {
	old := &mainState{layout: frame.DefaultLayout(), selectedFile: "legacy.txt", fileEditDraft: "legacy draft"}
	old.layout.Workspace = &frame.WorkspacePanes{}
	old.layout.Workspace.CaptureSessionProjection(old.workspaceProjectionLocked())
	saved := recovery.State{Workspace: "/old", SelectedSession: "saved", Layout: recovery.Layout{WorkspacePanes: old.layout.Workspace.Export(), RightOpen: true, RightTab: "files"}}
	s := &mainState{layout: frame.DefaultLayout()}
	if err := s.restoreWorkspaceSessions(saved); err != nil {
		t.Fatal(err)
	}
	s.switchWorkspaceSessionLocked("host", "/old", "other")
	if s.selectedFile != "" {
		t.Fatal("legacy tree assigned to wrong session")
	}
	if s.workspaceSessions.BySession[workspaceSessionKey("legacy", "/old", "saved")] == nil {
		t.Fatal("unselected legacy layout discarded")
	}
	s.switchWorkspaceSessionLocked("host", "/old", "saved")
	if s.selectedFile != "legacy.txt" || s.fileEditDraft != "legacy draft" || !s.filesPaneOpen {
		t.Fatal("legacy layout did not migrate to its saved session")
	}
	if s.workspaceSessions.BySession[workspaceSessionKey("legacy", "/old", "saved")] != nil {
		t.Fatal("legacy layout duplicated after migration")
	}
	for _, raw := range []string{`{"version":2,"bySession":{}}`, `{"version":1,"bySession":{"bad":null}}`, `{"version":1,"bySession":null}`} {
		before := s.layout.Workspace
		if err := s.restoreWorkspaceSessions(recovery.State{Layout: recovery.Layout{WorkspacePanes: json.RawMessage(raw)}}); err == nil {
			t.Fatalf("accepted invalid envelope %s", raw)
		}
		if s.layout.Workspace != before {
			t.Fatal("invalid recovery replaced current resources")
		}
	}
}
