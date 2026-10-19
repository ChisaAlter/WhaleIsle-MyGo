package interaction

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

type questionFixtureTransport func(*http.Request) (*http.Response, error)

func (f questionFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLiveQuestionReceiptWaitsForHostAndContinuedRetainsDraft(t *testing.T) {
	s := New()
	done := make(chan struct{})
	go func() {
		_, _ = s.HandleWaterfall(dsh.EventsWaterfall{EventID: "live", Event: EventUserQuestions, AgentID: "A", Request: json.RawMessage(`{"callId":"call-1","questions":[{"id":"q","question":"Pick","options":[{"label":"A"},{"label":"B"}]}]}`)})
		close(done)
	}()
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
	open := json.RawMessage(strings.Replace(string(continuedFixture), `"continued"`, `"open"`, 1))
	s.SyncProjectedQuestions("A", open, emptyInboxFixture, 1, 1)
	s.SetQuestionSingleSelect("live", "q", "B")
	s.ResolveQuestions("live")
	<-done
	s.OnResult(dsh.EventResult{EventID: "live"}, nil)
	if pending := s.Snapshot().Pending; len(pending) != 1 || !pending[0].Sending || pending[0].Continued {
		t.Fatal("HTTP receipt settled the request")
	}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 2, 2)
	pending := s.Snapshot().Pending
	if len(pending) != 1 || !pending[0].Continued || pending[0].Sending || *s.QuestionSingleSelectPtr(pending[0].EventID, "q") != "B" {
		t.Fatal("actual continuation lost answerability/draft")
	}
	s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1"}]}`), emptyInboxFixture, 3, 3)
	if len(s.Snapshot().Pending) != 0 {
		t.Fatal("confirmed closed question retained")
	}
	s.OnResult(dsh.EventResult{EventID: "live"}, nil)
	if len(s.Snapshot().Pending) != 0 {
		t.Fatal("late receipt reopened closed question")
	}
}

func TestLateQuestionReceiptDoesNotHideAlreadyDiscardedQueuedReply(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: questionFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		var request map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&request)
		close(started)
		<-release
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": true}})
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw))), Request: r}, nil
	})}}
	s := New()
	s.AttachClient(client, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 10, 10)
	id := s.Snapshot().Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "B")
	s.ResolveQuestions(id)
	<-started
	s.SyncProjectedQuestions("A", continuedFixture, json.RawMessage(`{"next-step":[{"source":{"kind":"user-question-reply","callId":"call-1"}}],"next-turn":[]}`), 10, 11)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 10, 12)
	s.ResolveQuestions(id)
	if calls.Load() != 1 {
		t.Fatal("second answer escaped the in-flight owner")
	}
	close(release)
	awaitQuestionFixture(t, func() bool { p := s.Snapshot().Pending; return len(p) == 1 && !p[0].Sending })
	if *s.QuestionSingleSelectPtr(id, "q") != "B" || s.SnapshotForAgent("A").QuestionStatus != "" {
		t.Fatal("late receipt suppressed discarded draft")
	}
}
