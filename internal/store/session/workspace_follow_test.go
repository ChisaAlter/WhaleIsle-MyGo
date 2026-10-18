package session

import (
	"encoding/json"
	"testing"
)

// WorkspaceFollowFrame belongs to the workspace registry, not session/list.
func TestApplyWorkspaceFollowUnknownFrameNoPanic(t *testing.T) {
	s := New()
	for _, raw := range []string{`{"type":"future","payload":{"x":1}}`, `null`, `{`, `{"type":"baseline","value":null}`, `{"type":"pinned","pinnedSessionIds":null}`, `{"type":"archived","archivedSessionIds":null}`} {
		s.applyWorkspaceFollowFrame(json.RawMessage(raw))
	}
}
