package interaction

import (
	"context"
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
		_, _ = s.HandleWaterfall(dsh.EventsWaterfall{EventID: "live", Event: EventUserQuestions, AgentID: "A", Request: json.RawMessage(`{"wait":{"callId":"call-1"},"questions":[{"id":"q","question":"Pick","options":[{"label":"A"},{"label":"B"}]}]}`)})
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

func TestQuestionReceiptUsesActualLegacyOrProjectedProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, request string
		projected     bool
	}{
		{"legacy", `{"callId":"call-1","questions":[{"id":"q","question":"Answer"}]}`, false},
		{"timed-indefinite", `{"wait":{"callId":"call-1"},"questions":[{"id":"q","question":"Answer"}]}`, true},
		{"timed", `{"wait":{"callId":"call-1","timed":true},"questions":[{"id":"q","question":"Answer"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Detach()
			client, mux := &dsh.Client{}, &dsh.Mux{}
			s.AttachClient(client, mux)
			owner := s.BeginEventsDelivery(context.Background(), client, mux)
			done := deliveryAnswer(t, s, owner, dsh.EventsWaterfall{EventID: "live", AgentID: "A", Event: EventUserQuestions, Request: json.RawMessage(tc.request)})
			awaitQuestionFixture(t, func() bool { return len(s.SnapshotForAgent("A").Pending) == 1 })
			s.SetQuestionCustom("live", "q", "owned answer")
			s.ResolveQuestions("live")
			out := deliveryReceive(t, done)
			s.OnResultForDelivery(owner, dsh.EventResult{EventID: "live", Outcome: *out}, nil)
			pending := s.SnapshotForAgent("A").Pending
			if !tc.projected {
				if len(pending) != 0 {
					t.Fatal("legacy question waited for a projection the Host never publishes")
				}
				return
			}
			if len(pending) != 1 || !pending[0].Sending {
				t.Fatal("projected question settled on HTTP receipt alone")
			}
			s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1"}]}`), emptyInboxFixture, 4, 4)
			if len(s.SnapshotForAgent("A").Pending) != 0 {
				t.Fatal("Host projection failed to settle timed question")
			}
		})
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
