package interaction

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

var continuedFixture = json.RawMessage(`{"active":[{"callId":"call-1","state":"continued","questions":[{"id":"q","question":"Pick","options":[{"label":"A"},{"label":"B"}]}]}],"settled":[]}`)
var emptyInboxFixture = json.RawMessage(`{"next-step":[],"next-turn":[]}`)

func awaitQuestionFixture(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("question condition not reached")
		}
		time.Sleep(time.Millisecond)
	}
}
func TestContinuedQuestionsRestoreAndRequireInbox(t *testing.T) {
	s := New()
	s.SyncProjectedQuestions("A", continuedFixture, nil)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("recovery assumed an unloaded inbox was empty")
	}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	snap := s.SnapshotForAgent("A")
	if len(snap.Pending) != 1 || !snap.Pending[0].Continued || snap.Pending[0].CallID != "call-1" {
		t.Fatalf("continued recovery: %+v", snap)
	}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 1 || len(s.SnapshotForAgent("B").Pending) != 0 {
		t.Fatal("duplicate or cross-session card")
	}
	s.SyncProjectedQuestions("B", json.RawMessage(`{"active":[{"callId":"call-1","state":"open","questions":[{"id":"q","question":"Pick"}]}],"settled":[]}`), emptyInboxFixture)
	if len(s.SnapshotForAgent("B").Pending) != 0 {
		t.Fatal("projection created an RPC answer path for an open waterfall")
	}
}
func TestContinuedQueuedAndSettledRepliesCannotResubmit(t *testing.T) {
	for _, lane := range []string{"next-step", "next-turn"} {
		t.Run(lane, func(t *testing.T) {
			s := New()
			s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
			inbox, _ := json.Marshal(map[string]any{lane: []any{map[string]any{"source": map[string]any{"kind": "user-question-reply", "callId": "call-1"}}}})
			s.SyncProjectedQuestions("A", continuedFixture, inbox)
			if len(s.SnapshotForAgent("A").Pending) != 0 {
				t.Fatal("queued late answer was editable")
			}
			s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
			if len(s.SnapshotForAgent("A").Pending) != 1 {
				t.Fatal("discarded queued answer did not become answerable")
			}
			s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1","answers":[]}]}`), emptyInboxFixture)
			if len(s.SnapshotForAgent("A").Pending) != 0 {
				t.Fatal("settled question survived")
			}
		})
	}
}
func TestContinuedWaterfallTimeoutRestoresDraftWithoutDoubleAnswer(t *testing.T) {
	s := New()
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "B")
	done := make(chan dsh.EventResultOutcome, 1)
	go func() {
		out, _ := s.HandleWaterfall(dsh.EventsWaterfall{EventID: "live", Event: EventUserQuestions, AgentID: "A", Request: json.RawMessage(`{"callId":"call-1","questions":[{"id":"q","question":"Pick","options":[{"label":"A"},{"label":"B"}]}]}`)})
		done <- *out
	}()
	awaitQuestionFixture(t, func() bool { p := s.SnapshotForAgent("A").Pending; return len(p) == 1 && p[0].EventID == "live" })
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 1 {
		t.Fatal("projection duplicated live waterfall")
	}
	s.resolve("live", dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Code: "ASK_TIMED_OUT"}})
	<-done
	s.OnResult(dsh.EventResult{EventID: "live"}, nil)
	snap := s.SnapshotForAgent("A")
	if len(snap.Pending) != 1 || !snap.Pending[0].Continued {
		t.Fatalf("timed out question lost: %+v", snap.Pending)
	}
	if got := *s.QuestionSingleSelectPtr(snap.Pending[0].EventID, "q"); got != "B" {
		t.Fatalf("draft lost during continuation: %q", got)
	}
}
func TestLiveWaterfallAnswerIsNotReplayedAsLateAnswer(t *testing.T) {
	s := New()
	done := make(chan struct{})
	go func() {
		_, _ = s.HandleWaterfall(dsh.EventsWaterfall{EventID: "live", Event: EventUserQuestions, AgentID: "A", Request: json.RawMessage(`{"callId":"call-1","questions":[{"id":"q","options":[{"label":"A"}]}]}`)})
		close(done)
	}()
	awaitQuestionFixture(t, func() bool { return len(s.SnapshotForAgent("A").Pending) == 1 })
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	s.SetQuestionSingleSelect("live", "q", "A")
	s.ResolveQuestions("live")
	<-done
	s.OnResult(dsh.EventResult{EventID: "live"}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("delivered waterfall answer gained a second RPC submission")
	}
}
func TestLateQuestionAnswerRPCFailureRetryAndQueuedEcho(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					AgentID, CallID string
					Answer          userQuestionsAnswer
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		n := calls.Add(1)
		if n == 1 {
			<-release
		}
		if body.Payload.Args.AgentID != "A" || body.Payload.Args.CallID != "call-1" || len(body.Payload.Args.Answer.Answers) != 1 {
			t.Errorf("wrong answer route: %+v", body.Payload.Args)
		}
		result := map[string]any{"ok": true, "value": true}
		if n == 1 {
			result = map[string]any{"ok": false, "error": map[string]any{"code": "TRANSIENT", "message": "retry"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": result})
	}))
	defer srv.Close()
	s := New()
	s.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "A")
	s.ResolveQuestions(id)
	s.ResolveQuestions(id)
	awaitQuestionFixture(t, func() bool { return calls.Load() == 1 })
	close(release)
	awaitQuestionFixture(t, func() bool {
		p := s.SnapshotForAgent("A").Pending
		return len(p) == 1 && p[0].Error != "" && !p[0].Sending
	})
	s.Retry(id)
	awaitQuestionFixture(t, func() bool { return s.SnapshotForAgent("A").QuestionStatus == "submitted" })
	s.ResolveQuestions(id)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if calls.Load() != 2 || len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("accepted answer was repeated before inbox echo")
	}
	s.SyncProjectedQuestions("A", continuedFixture, json.RawMessage(`{"next-step":[{"source":{"kind":"user-question-reply","callId":"call-1"}}],"next-turn":[]}`))
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("queued response regained editable card")
	}
	s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1"}]}`), emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("settled response regained editable card")
	}
}
func TestLateQuestionAnswerResultIgnoredAfterHostReplacement(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(started)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": true}})
	}))
	defer srv.Close()
	s := New()
	s.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "A")
	s.ResolveQuestions(id)
	<-started
	s.Detach()
	s.AttachClient(&dsh.Client{}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	close(release)
	// The new Host owns a new recovery epoch and an independent editable card.
	if got := s.SnapshotForAgent("A"); len(got.Pending) != 1 || got.QuestionStatus != "" || got.Pending[0].Sending {
		t.Fatalf("stale submission leaked across Host: %+v", got)
	}
}
func TestRecoveredQuestionDenyDoesNotInventWaterfall(t *testing.T) {
	s := New()
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.ResolveDeny(id)
	if len(s.SnapshotForAgent("A").Pending) != 1 || len(s.waiters) != 0 {
		t.Fatal("late-question dismissal changed live waterfall state")
	}
}

func TestContinuedFreshReconnectCanRestoreDiscardedSubmission(t *testing.T) {
	s := New()
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 10, 10)
	s.mu.Lock()
	p := s.questionAgentLocked("A")
	p.submitted["call-1"] = submittedQuestion{asOfSeq: 10}
	s.reconcileProjectedQuestionsLocked("A")
	s.mu.Unlock()
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 10, 10)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("same pre-submission snapshot re-enabled answer")
	}
	// A later complete baseline proves the call is still continued and no reply remains.
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 20, 20)
	if len(s.SnapshotForAgent("A").Pending) != 1 {
		t.Fatal("fresh reconnect failed to recover discarded reply")
	}
	s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1"}]}`), emptyInboxFixture, 21, 21)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 10, 10)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("stale UI snapshot reopened settled question")
	}
}

func TestLateQuestionNegativeRPCResultPreservesDraftAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": false}})
	}))
	defer srv.Close()
	s := New()
	s.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "B")
	s.ResolveQuestions(id)
	awaitQuestionFixture(t, func() bool {
		p := s.SnapshotForAgent("A").Pending
		return len(p) == 1 && !p[0].Sending && p[0].Error != ""
	})
	if got := *s.QuestionSingleSelectPtr(id, "q"); got != "B" {
		t.Fatal("negative reply erased draft")
	}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 1 || s.questionAgentLocked("A").answered["call-1"] {
		t.Fatal("negative reply was fabricated into an answered projection")
	}
	s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[]}`), emptyInboxFixture)
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("authoritative removal did not settle failed card")
	}
}
