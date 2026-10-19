package session

import (
	"net/http"
	"testing"

	"whaleisle/internal/dsh"
)

// Open is exercised after a synthetic completed admission. There is no Mux and
// this transport fails any unexpected HTTP action: no Host/model is contacted.
func TestCreatorIntroductionAcknowledgmentOwnership(t *testing.T) {
	calls := 0
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://creator-fixture.invalid"}, HTTP: &http.Client{Transport: presetReadingTransport(func(*http.Request) (*http.Response, error) {
		calls++
		t.Error("unexpected HTTP action")
		return nil, http.ErrServerClosed
	})}}
	s := New()
	s.Attach(client, nil)
	s.SetWorkspaceCwd("/fixture")
	open := func(applied bool) string {
		t.Helper()
		c := NewCreatorTask(s, nil)
		c.ObserveVisible(true)
		c.Prepare()
		prepared := c.Snapshot()
		if !prepared.Review {
			t.Fatalf("could not prepare: %+v", prepared)
		}
		c.mu.Lock()
		c.state.Review = false
		c.state.Attempted, c.state.Created, c.state.Applied = true, true, applied
		c.mu.Unlock()
		if !c.Open(prepared.Ticket, applied) {
			t.Fatal("could not open retained task")
		}
		return prepared.SessionID
	}
	firstID := open(true)
	first := s.AgentPresetSelection().Introduction
	epoch := s.Snapshot().ComposerHostGeneration
	if first == 0 {
		t.Fatal("applied Creator did not announce its receiving Session")
	}
	for _, wrong := range []struct {
		host uint64
		id   string
		cue  uint64
	}{{epoch + 1, firstID, first}, {epoch, "other", first}, {epoch, firstID, first + 1}} {
		s.AcknowledgeAgentPresetIntroduction(wrong.host, wrong.id, wrong.cue)
		if s.AgentPresetSelection().Introduction != first {
			t.Fatal("foreign acknowledgment consumed active cue", wrong)
		}
	}
	secondID := open(true)
	second := s.AgentPresetSelection().Introduction
	if second == 0 || second == first || secondID == firstID {
		t.Fatal("later Creator reused cue identity")
	}
	s.AcknowledgeAgentPresetIntroduction(epoch, firstID, first)
	if s.AgentPresetSelection().Introduction != second {
		t.Fatal("old Session completion consumed the new cue")
	}
	s.AcknowledgeAgentPresetIntroduction(epoch, secondID, second)
	if s.AgentPresetSelection().Introduction != 0 {
		t.Fatal("current cue was not consumed")
	}
	s.mu.Lock()
	s.selectSessionLocked("")
	s.mu.Unlock()
	if s.AgentPresetSelection().Introduction != 0 {
		t.Fatal("consumed cue leaked into the unbound hero")
	}
	unappliedID := open(false)
	if s.AgentPresetSelection().Introduction != 0 || s.Snapshot().SelectedID != unappliedID {
		t.Fatal("uncertain preset recovery invented a Creator announcement")
	}
	oldID := open(true)
	oldCue := s.AgentPresetSelection().Introduction
	s.Attach(&dsh.Client{Endpoint: client.Endpoint, HTTP: client.HTTP}, nil)
	if s.AgentPresetSelection().Introduction != 0 {
		t.Fatal("Host replacement retained old cue")
	}
	newID := open(true)
	newCue := s.AgentPresetSelection().Introduction
	newEpoch := s.Snapshot().ComposerHostGeneration
	if newEpoch == epoch || newCue == 0 {
		t.Fatal("replacement Host has no distinct ownership")
	}
	s.AcknowledgeAgentPresetIntroduction(epoch, oldID, oldCue)
	s.AcknowledgeAgentPresetIntroduction(epoch, newID, newCue)
	if s.AgentPresetSelection().Introduction != newCue {
		t.Fatal("old Host receipt consumed replacement cue")
	}
	s.AcknowledgeAgentPresetIntroduction(newEpoch, newID, newCue)
	if s.AgentPresetSelection().Introduction != 0 || calls != 0 {
		t.Fatal("acknowledgment wrote externally or failed to settle")
	}
}
