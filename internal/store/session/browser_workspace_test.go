package session

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/dsh"
)

func TestBrowserWorkspaceScopeUsesAuthoritativeDirectMembership(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.items = []Item{{SessionID: "member", Cwd: "/different-cwd"}, {SessionID: "orphan", Cwd: "/workspace"}, {SessionID: "child", ParentID: "member", Cwd: "/workspace"}}
	s.selectedID = "member"
	if cwd, _, sid, ready, err := s.BrowserWorkspaceScope(); cwd != "" || sid != "member" || ready || err != nil {
		t.Fatal("loading registry admitted a guessed workspace")
	}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"w","path":"/workspace","title":"Authoritative","sessionIds":["member"]}],"scratchCwd":"","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	if cwd, title, sid, ready, err := s.BrowserWorkspaceScope(); cwd != "/workspace" || title != "Authoritative" || sid != "member" || !ready || err != nil {
		t.Fatalf("direct membership ignored: %q %q %q %v %v", cwd, title, sid, ready, err)
	}
	for _, id := range []string{"orphan", "child"} {
		s.selectedID = id
		cwd, title, sid, ready, err := s.BrowserWorkspaceScope()
		if cwd != "" || title != "" || sid != id || !ready || err != nil {
			t.Fatalf("cwd/parent incorrectly became membership for %s", id)
		}
	}
	s.workspaceErr = errors.New("registry unavailable")
	if cwd, _, sid, ready, err := s.BrowserWorkspaceScope(); cwd != "" || sid != "child" || ready || err == nil {
		t.Fatal("failed registry admitted session fallback before readiness")
	}
	s.workspaceErr = nil
	s.client = nil
	if _, _, _, ready, _ := s.BrowserWorkspaceScope(); ready {
		t.Fatal("detached Host retained ready browser partition")
	}
}
