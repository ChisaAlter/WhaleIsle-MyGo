package main

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
)

func TestPhoneGitRegisteredCandidates(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(root, "project")
	os.Mkdir(other, 0700)
	rows := phoneGitRegisteredRoots([]session.WorkspaceRow{{WorkspaceID: "w", Path: root, Title: "Work"}, {WorkspaceID: "bad", Path: "/"}, {WorkspaceID: "missing", Path: filepath.Join(root, "absent")}}, []projectflow.Project{{ID: "p", Title: "Project", CanonicalWorkingDirectory: other}})
	if len(rows) != 2 || rows[0].ID != "host:w" || rows[1].ID != "project:p" {
		t.Fatalf("unverified candidate: %+v", rows)
	}
	for _, r := range rows {
		if r.Selected {
			t.Fatal("candidate granted by default")
		}
	}
	s := &mainState{}
	s.setPhoneGitWorkspace("arbitrary:/", true)
	if len(s.phoneGitWorkspaces.Roots) != 0 {
		t.Fatal("unlisted path granted")
	}
}
func TestPhoneGitWorkspaceOwnerLossRevokes(t *testing.T) {
	s := &mainState{phoneGitWorkspaces: phoneGitWorkspaceScope{Owner: "old-host", Roots: map[string]bool{"/old": true}}}
	if rows := s.phoneGitWorkspaceRows(); len(rows) != 0 {
		t.Fatal("unowned Host supplied candidates")
	}
	if len(s.phoneGitWorkspaces.Roots) != 0 {
		t.Fatal("old grant survived owner loss")
	}
	s.phoneGitWorkspaces = phoneGitWorkspaceScope{Owner: "host", Roots: map[string]bool{"/old": true}}
	s.revokePhoneGit()
	if len(s.phoneGitWorkspaces.Roots) != 0 {
		t.Fatal("global revoke retained grants")
	}
}
