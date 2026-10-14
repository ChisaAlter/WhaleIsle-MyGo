package session

import (
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestDeleteResultRemovesRowsBeforeArchiveEcho(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "root"}, {SessionID: "child"}, {SessionID: "fork"}}
	s.selectedID = "fork"
	s.archivedSessionIDs = []string{"root", "child", "other"}
	s.listLoading = true
	s.applyDeleteResultLocked(api.SessionDeleteValue{DeletedSessionIds: []string{"root", "child"}, ArchivedSessionIds: []string{"other"}})
	if len(s.items) != 1 || s.items[0].SessionID != "fork" || s.selectedID != "fork" {
		t.Fatalf("items=%+v selected=%q", s.items, s.selectedID)
	}
	if len(s.archivedSessionIDs) != 1 || s.archivedSessionIDs[0] != "other" {
		t.Fatalf("archive=%v", s.archivedSessionIDs)
	}
	if len(s.listMutations) != 2 {
		t.Fatalf("pending list lost delete tombstones: %v", s.listMutations)
	}
}
