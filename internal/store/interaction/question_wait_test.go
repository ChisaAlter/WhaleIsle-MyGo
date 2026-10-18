package interaction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// Editing before a claim opens must hold it, including a zero-remaining opening.
// The old unconditional attachWait timer incorrectly returns ASK_TIMED_OUT.
func TestTimedQuestionEditingRetainsForegroundClaim(t *testing.T) {
	opened := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var frame dsh.ClientFrame
		if wsjson.Read(r.Context(), conn, &frame) != nil {
			return
		}
		if frame.Endpoint != "userQuestions/attachWait" {
			t.Errorf("endpoint %s", frame.Endpoint)
			return
		}
		_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: json.RawMessage(`{"remainingMs":0}`)})
		close(opened)
		for {
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
		}
	}))
	defer server.Close()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: server.URL}}
	defer mux.Close()
	s := New()
	s.pending = []Pending{{EventID: "event", AgentID: "A", CallID: "call", Event: EventUserQuestions}}
	s.questionDrafts["event"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "q", Question: "Explain"}}})
	waiter := make(chan dsh.EventResultOutcome, 1)
	s.waiters["event"] = waiter
	s.SetQuestionCustom("event", "q", "I am still answering")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.attachQuestionWait(ctx, mux, "A", "call", "event")
	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("claim never opened")
	}
	select {
	case got := <-waiter:
		t.Fatalf("edited question timed out: %+v", got)
	case <-time.After(75 * time.Millisecond):
	}
	s.ResolveQuestions("event")
	select {
	case got := <-waiter:
		if got.Kind != "result" {
			t.Fatalf("answer lost: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("edited question no longer answerable")
	}
}

func TestQuestionWaitFocusPreservesRemainingTime(t *testing.T) {
	now := time.Unix(100, 0)
	w := questionWaitState{active: true, deadline: now.Add(5 * time.Second)}
	if !w.focus(true, now.Add(2*time.Second)) {
		t.Fatal("focus did not pause")
	}
	paused := w.view(now.Add(10 * time.Second))
	if paused.Running || paused.RemainingMs != 3000 || paused.State != "focused" {
		t.Fatalf("paused %+v", paused)
	}
	w.focus(false, now.Add(10*time.Second))
	resumed := w.view(now.Add(11 * time.Second))
	if !resumed.Running || resumed.RemainingMs != 2000 {
		t.Fatalf("resumed %+v", resumed)
	}
}

func TestQuestionWaitTakeTimeAndEditsAreRequestScoped(t *testing.T) {
	s := New()
	req := userQuestionsRequest{Questions: []userQuestionItem{{ID: "q", Options: []userQuestionOption{{Label: "A"}}}}}
	for _, id := range []string{"a", "b"} {
		s.pending = append(s.pending, Pending{EventID: id, AgentID: id, CallID: id, Event: EventUserQuestions})
		s.questionDrafts[id] = newDraftSession(req)
		s.questionDrafts[id].wait = questionWaitState{active: true, deadline: time.Now().Add(time.Minute)}
	}
	s.SyncQuestionFocus("a", "a")
	s.SyncQuestionFocus("b", "b")
	if s.Snapshot().QuestionCards["a"].Wait.State != "counting" || s.Snapshot().QuestionCards["b"].Wait.State != "focused" {
		t.Fatal("focus crossed session switch")
	}
	s.TakeQuestionTime("wrong", "a")
	if s.Snapshot().QuestionCards["a"].Wait.State != "counting" {
		t.Fatal("stale owner held another request")
	}
	s.TakeQuestionTime("a", "a")
	s.HideQuestion("a")
	s.ReopenQuestion("a")
	if s.Snapshot().QuestionCards["a"].Wait.State != "waiting" {
		t.Fatal("hide/remount lost take-time")
	}
	s.SetQuestionOptionChecked("b", "q", "A", true)
	s.SyncQuestionFocus("", "")
	if s.Snapshot().QuestionCards["b"].Wait.State != "editing" {
		t.Fatal("first choice did not hold wait")
	}
}

func TestQuestionWaitUntouchedExpiresAndCancelledClaimDoesNot(t *testing.T) {
	setup := func() (*Store, chan dsh.EventResultOutcome) {
		s := New()
		s.pending = []Pending{{EventID: "e", AgentID: "A", CallID: "call", Event: EventUserQuestions}}
		s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "q"}}})
		ch := make(chan dsh.EventResultOutcome, 1)
		s.waiters["e"] = ch
		return s, ch
	}
	s, ch := setup()
	s.runQuestionWait(context.Background(), "A", "call", "e", 0)
	got := <-ch
	if got.Error == nil || got.Error.Code != "ASK_TIMED_OUT" {
		t.Fatalf("timeout %+v", got)
	}
	s, ch = setup()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.runQuestionWait(ctx, "A", "call", "e", 0)
	select {
	case <-ch:
		t.Fatal("cancelled claim settled")
	default:
	}
	if s.Snapshot().QuestionCards["e"].Wait != nil {
		t.Fatal("cancelled claim advertised countdown")
	}
}
