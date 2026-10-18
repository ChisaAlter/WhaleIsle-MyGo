package botnotify

import "testing"

func TestReadCandidatesBaselineAndVisibleSession(t *testing.T) {
	rows := []ContactActivity{
		{ID: "legacy", SessionID: "a", ActivitySeq: 8},
		{ID: "visible", SessionID: "selected", ActivitySeq: 12, LastSeenSeq: 9, ReadInitialized: true},
		{ID: "other", SessionID: "b", ActivitySeq: 12, LastSeenSeq: 9, ReadInitialized: true},
		{ID: "seen", SessionID: "selected", ActivitySeq: 12, LastSeenSeq: 12, ReadInitialized: true},
		{ID: "unavailable", ActivitySeq: 0, Error: "inspection unavailable"},
	}
	got := ReadCandidates(rows, "selected")
	if len(got) != 2 || got[0].ID != "legacy" || got[1].ID != "visible" {
		t.Fatalf("wrong marks %+v", got)
	}
	if rows[0].ReadInitialized {
		t.Fatal("optimistic local write")
	}
}
