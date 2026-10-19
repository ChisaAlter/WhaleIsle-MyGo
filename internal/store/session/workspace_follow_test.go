package session

import (
	"encoding/json"
	"errors"
	"testing"
)

// WorkspaceFollowFrame belongs to the workspace registry, not session/list.
func TestApplyWorkspaceFollowUnknownFrameNoPanic(t *testing.T) {
	s := New()
	for _, raw := range []string{`{"type":"future","payload":{"x":1}}`, `null`, `{`, `{"type":"baseline","value":null}`, `{"type":"pinned","pinnedSessionIds":null}`, `{"type":"archived","archivedSessionIds":null}`} {
		s.applyWorkspaceFollowFrame(json.RawMessage(raw))
	}
}

func TestWorkspaceBaselineRetiresOnlyItsPriorReaderError(t *testing.T) {
	for _, unrelated := range []bool{false, true} {
		s := New()
		s.workspaceErr = errors.New("workspace follow unavailable; reconnect and try again")
		s.lastAction = "workspace follow: " + s.workspaceErr.Error()
		if unrelated {
			s.lastAction = "rename failed"
		}
		s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[],"scratchCwd":"","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
		if !s.workspaceReady || s.workspaceErr != nil {
			t.Fatal("authoritative baseline did not recover workspace readiness")
		}
		if !unrelated && s.lastAction != "" || unrelated && s.lastAction != "rename failed" {
			t.Fatal("recovered reader left stale feedback or erased another action")
		}
	}
}
