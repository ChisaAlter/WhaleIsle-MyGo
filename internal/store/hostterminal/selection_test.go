package hostterminal

import "testing"

func TestHostTerminalSelectionRequiresCurrentDisplayedIdentity(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 1)
	o := s.Snapshot("session").Owner
	s.mu.Lock()
	// The Host makes this stream read-only: input is revoked while
	// local selection stays available.
	a.writable = false
	s.mu.Unlock()
	if s.Snapshot("session").Writable || !s.CanCopySelection(o, "session", "term", a) {
		t.Fatal("read-only selection rejected")
	}
	for _, target := range []struct {
		o           uint64
		session, id string
		a           *Attachment
	}{
		{o + 1, "session", "term", a}, {o, "other", "term", a}, {o, "session", "other", a}, {o, "session", "term", nil},
	} {
		if s.CanCopySelection(target.o, target.session, target.id, target.a) {
			t.Fatal("stale selection accepted", target)
		}
	}
	s.mu.Lock()
	a.entry.closing = true
	s.mu.Unlock()
	if s.CanCopySelection(o, "session", "term", a) {
		t.Fatal("closing selection accepted")
	}
	s.mu.Lock()
	a.entry.closing = false
	s.mu.Unlock()
	s.Show("", false)
	if s.CanCopySelection(o, "session", "term", a) {
		t.Fatal("hidden selection accepted")
	}
	s.Show("session", true)
	if s.CanCopySelection(o, "session", "term", a) {
		t.Fatal("replaced attachment accepted")
	}
	s.bind(nil, nil, false, nil)
	if s.CanCopySelection(o, "session", "term", a) {
		t.Fatal("replaced Host accepted")
	}
}
