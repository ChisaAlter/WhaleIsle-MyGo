package session

import "testing"

func TestWithoutProjectionKeysDetachesAnEditedCopy(t *testing.T) {
	s := New()
	snap := s.Snapshot()
	if !snap.ProjectionKey().Valid() || !snap.HistoryProjectionKey().Valid() || !snap.RosterProjectionKey().Valid() {
		t.Fatal("a store view carries no keys")
	}

	edited := snap.WithoutProjectionKeys()
	if edited.ProjectionKey().Valid() || edited.HistoryProjectionKey().Valid() || edited.RosterProjectionKey().Valid() {
		t.Fatal("an edited copy still claims a store revision")
	}
	// The receiver is a copy: the view it came from keeps its keys.
	if !snap.ProjectionKey().Valid() || !snap.HistoryProjectionKey().Valid() || !snap.RosterProjectionKey().Valid() {
		t.Fatal("WithoutProjectionKeys cleared the original view")
	}
}
