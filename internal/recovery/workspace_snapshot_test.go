package recovery

import (
	"encoding/json"
	"testing"
)

func TestWorkspacePaneSnapshotsAndRejectedUpdatesCannotAliasStoredState(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Update(func(v *State) { v.Layout.WorkspacePanes = json.RawMessage(`{"version":1,"active":"main"}`) }); err != nil {
		t.Fatal(err)
	}
	snapshot := s.Snapshot()
	snapshot.Layout.WorkspacePanes[0] = '!'
	if !json.Valid(s.Snapshot().Layout.WorkspacePanes) {
		t.Fatal("snapshot changed stored workspace layout")
	}
	err = s.Update(func(v *State) { v.Layout.WorkspacePanes[0] = '!' })
	if err == nil {
		t.Fatal("invalid layout unexpectedly saved")
	}
	if !json.Valid(s.Snapshot().Layout.WorkspacePanes) {
		t.Fatal("rejected update corrupted the resident recovery state")
	}
}
