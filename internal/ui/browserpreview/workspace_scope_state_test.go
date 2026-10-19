package browserpreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
)

func TestBrowserWorkspacePartitions(t *testing.T) {
	makeScope := func(owner WorkspaceOwner) browserWorkspace {
		t.Helper()
		s, err := makeBrowserWorkspace(owner)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	owner := WorkspaceOwner{Authority: "supervised-dsh-home:fixture", SessionID: "session-a", Ready: true, Cwd: "/host/project", Title: "One"}
	first := makeScope(owner)
	owner.Title, owner.SessionID = "Renamed", "session-b"
	if reopened := makeScope(owner); first.key != reopened.key || first.needsNewTab(reopened) {
		t.Fatal("authoritative cwd partition changed after rename/session switch")
	}
	// Existing cwd cookie directories retain their exact pre-session-fallback key.
	encoded, _ := json.Marshal([]string{"browser-workspace-v1", owner.Authority, "workspace", owner.Cwd})
	sum := sha256.Sum256(encoded)
	if first.key != hex.EncodeToString(sum[:]) {
		t.Fatal("existing workspace cookie partition was migrated")
	}
	owner.Cwd = "/host/other"
	other := makeScope(owner)
	owner.Cwd, owner.Authority = "/host/project", "supervised-dsh-home:other"
	otherHost := makeScope(owner)
	if first.key == other.key || first.key == otherHost.key || !first.needsNewTab(other) {
		t.Fatal("workspace/Host partition was reused")
	}
	owner.Authority, owner.Cwd, owner.SessionID = "supervised-dsh-home:fixture", "", "session-a"
	sessionA := makeScope(owner)
	owner.Title = "No directory session"
	if sessionA.key != makeScope(owner).key {
		t.Fatal("session title changed partition")
	}
	owner.SessionID = "session-b"
	if sessionA.key == makeScope(owner).key {
		t.Fatal("directoryless sessions shared cookies")
	}
	owner.Cwd, owner.SessionID = "shared", "session-a"
	if sessionA.key == makeScope(owner).key || sessionA.key == first.key {
		t.Fatal("session fallback collided with cwd identity")
	}
	owner.Cwd = `C:\Work\Project\`
	win := makeScope(owner)
	owner.Cwd = "c:/work/project"
	if win.key == makeScope(owner).key {
		t.Fatal("opaque Host cwd inferred desktop case/separator aliases")
	}
	owner.Cwd = "/host/Project"
	if first.key == makeScope(owner).key {
		t.Fatal("distinct Host cwd spellings were merged")
	}
	if len(first.key) != 64 || first.key == "" || sessionA.label == "" {
		t.Fatal("partition identity/disclosure missing")
	}
	for _, invalid := range []WorkspaceOwner{
		{Authority: "desktop", SessionID: "s", Cwd: "/project"},
		{Authority: "desktop", Ready: true, Cwd: "/project"},
		{SessionID: "s", Ready: true, Cwd: "/project"},
		{Authority: "desktop", SessionID: "s", Ready: true, Err: errors.New("membership failed")},
	} {
		if _, err := makeBrowserWorkspace(invalid); err == nil {
			t.Fatalf("untrusted/not-ready owner accepted: %+v", invalid)
		}
	}
	oldKey := first.key
	_ = first.needsNewTab(other)
	if first.key != oldKey {
		t.Fatal("tab ownership mutated during comparison")
	}
}
