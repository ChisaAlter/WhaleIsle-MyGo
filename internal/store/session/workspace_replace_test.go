package session

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPublishedWorkspaceReplaceUsesCompleteAdmittedRegistry(t *testing.T) {
	s := New()
	s.cwd = "/owned/retained"
	s.items = []Item{{SessionID: "retained-session"}, {SessionID: "hidden-session", Pinned: true}}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"hidden","path":"/owned/hidden","title":"Hidden","sessionIds":["hidden-session"]},{"workspaceId":"retained","path":"/owned/retained","title":"Retained","sessionIds":["retained-session"]}],"scratchCwd":"/owned/scratch","pinnedSessionIds":["hidden-session"],"archivedSessionIds":["hidden-session"]}}`))
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"replace","value":{"items":[{"workspaceId":"retained","path":"/owned/retained","title":"Retained renamed","sessionIds":["retained-session"]}],"scratchCwd":"/owned/new-scratch","pinnedSessionIds":["retained-session"],"archivedSessionIds":[]}}`))
	v := s.Snapshot()
	if !v.WorkspaceReady || v.WorkspaceError != "" || len(v.Workspaces) != 1 || v.Workspaces[0].WorkspaceID != "retained" || v.Workspaces[0].Title != "Retained renamed" {
		t.Fatalf("complete replacement was not adopted: %+v", v.Workspaces)
	}
	if v.ScratchCwd != "/owned/new-scratch" || !reflect.DeepEqual(v.PinnedSessionIDs, []string{"retained-session"}) || len(v.ArchivedSessionIDs) != 0 {
		t.Fatalf("replacement kept stale scratch/pins/archive: %q %v %v", v.ScratchCwd, v.PinnedSessionIDs, v.ArchivedSessionIDs)
	}
	if s.WorkspaceID() != "retained" || !v.Items[0].Pinned || v.Items[1].Pinned {
		t.Fatal("replacement did not update configured identity or pin flags")
	}
	// A replacement reports admitted visibility, not deletion of a conversation.
	s.Select("retained-session")
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"replace","value":{"items":[],"scratchCwd":"/owned/new-scratch","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	if v = s.Snapshot(); len(v.Workspaces) != 0 || v.SelectedID != "retained-session" || len(v.Items) != 2 {
		t.Fatal("visibility replacement was mistaken for Session deletion")
	}
}
