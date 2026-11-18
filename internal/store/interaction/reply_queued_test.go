package interaction

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

// The Host rejects a second answer to a continued call with REPLY_QUEUED while
// the reply it already holds waits for admission. That answer reached the Host,
// so the card must not report a failed delivery and must not offer a retry that
// cannot converge until the agent admits the queued reply.
func TestAnswerReplyQueuedIsQueuedNotFailed(t *testing.T) {
	var calls atomic.Int32
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
		calls.Add(1)
		if body.Payload.Args.AgentID != "A" || body.Payload.Args.CallID != "call-1" || len(body.Payload.Args.Answer.Answers) != 1 {
			t.Errorf("wrong answer route: %+v", body.Payload.Args)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": body.RPCID,
			"result": map[string]any{"ok": false, "error": map[string]any{
				"code":    "REPLY_QUEUED",
				"message": "a reply is already queued for this question",
			}},
		})
	}))
	defer srv.Close()
	s := New()
	s.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "A")
	s.ResolveQuestions(id)
	awaitQuestionFixture(t, func() bool { return calls.Load() == 1 })
	awaitQuestionFixture(t, func() bool {
		snap := s.SnapshotForAgent("A")
		return len(snap.Pending) == 0 || !snap.Pending[0].Sending
	})

	snap := s.SnapshotForAgent("A")
	if len(snap.Pending) != 0 {
		t.Fatalf("a queued reply stayed editable as a failed delivery: %+v", snap.Pending)
	}
	if snap.QuestionStatus != "queued" {
		t.Fatalf("status = %q, want the queued notice", snap.QuestionStatus)
	}
	if calls.Load() != 1 {
		t.Fatalf("answer RPC ran %d times, want one", calls.Load())
	}

	// The fence holds: the projection still lists the call as continued until the
	// queued reply is admitted, and that must not reopen the card.
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	s.ResolveQuestions(id)
	if calls.Load() != 1 || len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("queued answer before its inbox echo was answerable again")
	}
	s.SyncProjectedQuestions("A", continuedFixture, json.RawMessage(`{"next-step":[{"source":{"kind":"user-question-reply","callId":"call-1"}}],"next-turn":[]}`))
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("queued answer regained an editable card")
	}
	s.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"call-1"}]}`), emptyInboxFixture)
	if got := s.SnapshotForAgent("A"); len(got.Pending) != 0 || calls.Load() != 1 {
		t.Fatal("admitted answer reopened the card or replayed the RPC")
	} else if got.QuestionStatus != "" {
		t.Fatalf("queued notice outlived admission: %q", got.QuestionStatus)
	}
}
