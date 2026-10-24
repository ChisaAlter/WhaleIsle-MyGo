package hostterminal

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRecoveryAdoptsExactExistingProcessWithoutCreateOrInput(t *testing.T) {
	s := new(Store)
	r := fixture()
	s.bind(r, r, true, nil)
	defer s.bind(nil, nil, false, nil)
	s.Show("session", true)
	owner := s.Snapshot("session").Owner
	saved := &RecoveryTarget{Authority: "host-a", SessionID: "session", TerminalID: "term", Title: "Saved terminal"}
	if s.AdoptRecovery(owner+1, "host-a", "session", saved) || s.AdoptRecovery(owner, "host-b", "session", saved) || s.AdoptRecovery(owner, "host-a", "other", saved) {
		t.Fatal("foreign recovery accepted")
	}
	if r.count("terminal/environment") != 0 || r.count("terminal/create") != 0 {
		t.Fatal("foreign descriptor contacted Host")
	}
	if !s.AdoptRecovery(owner, "host-a", "session", saved) {
		t.Fatal("exact recovery rejected")
	}
	_ = attach(t, s, r, 4)
	if r.count("terminal/create") != 0 || r.count("terminal/write") != 0 {
		t.Fatal("recovery created a replacement or replayed input")
	}
	if !s.StartFresh(owner, "session") || r.count("terminal/create") != 0 {
		t.Fatal("fresh display allocated over a recovered descriptor")
	}
	snapshot := s.RecoverySnapshot("host-a", "session")
	if snapshot == nil || snapshot.TerminalID != "term" || snapshot.SessionID != "session" || snapshot.Authority != "host-a" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for field := range fields {
		if field != "authority" && field != "sessionId" && field != "terminalId" && field != "title" {
			t.Fatalf("runtime capability/output persisted: %s", field)
		}
	}
	if s.AdoptRecovery(owner, "host-a", "session", saved) {
		t.Fatal("repaint repeated discovery/attach")
	}
}

func TestMissingRecoveryNeverAllocatesAndFreshFailureIsConsumedOnce(t *testing.T) {
	t.Run("missing descriptor", func(t *testing.T) {
		s := new(Store)
		r := fixture()
		r.list = nil
		s.bind(r, r, true, nil)
		defer s.bind(nil, nil, false, nil)
		s.Show("session", true)
		owner := s.Snapshot("session").Owner
		if !s.AdoptRecovery(owner, "host", "session", &RecoveryTarget{Authority: "host", SessionID: "session", TerminalID: "gone"}) {
			t.Fatal("recovery not started")
		}
		v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.Error != "" })
		if v.ActiveID != "gone" || v.Phase != "failed" || r.count("terminal/create") != 0 || r.count("terminal/write") != 0 {
			t.Fatalf("missing descriptor replaced: %+v", v)
		}
		if !s.StartFresh(owner, "session") || r.count("terminal/create") != 0 {
			t.Fatal("missing recovery turned into fresh create")
		}
	})
	t.Run("fresh failure", func(t *testing.T) {
		s := new(Store)
		r := fixture()
		r.createErr = errors.New("allocation response lost")
		s.bind(r, r, true, nil)
		defer s.bind(nil, nil, false, nil)
		owner := s.Snapshot("session").Owner
		if s.StartFresh(owner, "session") {
			t.Fatal("hidden session consumed fresh create")
		}
		s.Show("session", true)
		if !s.StartFresh(owner, "session") {
			t.Fatal("fresh explicit occurrence rejected")
		}
		v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.Error != "" })
		if r.count("terminal/create") != 1 || v.ActiveID == "" {
			t.Fatalf("uncertain identity lost: %+v", v)
		}
		for range 3 {
			if !s.StartFresh(owner, "session") {
				t.Fatal("consumed marker was not acknowledged")
			}
		}
		if r.count("terminal/create") != 1 || s.Snapshot("session").ActiveID != v.ActiveID {
			t.Fatal("repaint retried uncertain allocation")
		}
	})
}
