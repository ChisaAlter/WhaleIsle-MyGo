package session

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
)

func awaitCoalescedQuestion(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("question state did not settle")
		}
		time.Sleep(time.Millisecond)
	}
}

// Apply real producer control frames without painting their intermediate
// snapshots. The only UI sync below sees the final, empty inbox.
func TestQuestionControlQueueDiscardCoalescingPreservesAttemptOwnership(t *testing.T) {
	for _, phase := range []string{"while-request-pending", "after-receipt", "before-click", "still-queued-at-receipt"} {
		t.Run(phase, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
				var request struct {
					RPCID string `json:"rpcId"`
				}
				_ = json.NewDecoder(r.Body).Decode(&request)
				close(started)
				<-release
				raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": true}})
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw))), Request: r}, nil
			})}}
			source := New()
			source.client = client
			source.selectedID = "A"
			face := json.RawMessage(`{"active":[{"callId":"call-1","state":"continued","questions":[{"id":"q","question":"Pick","options":[{"label":"A"},{"label":"B"}]}]}],"settled":[]}`)
			source.applyQuestionFaceLocked("A", "userQuestions", face, 10)
			source.applyQuestionFaceLocked("A", "inbox", json.RawMessage(`{"next-step":[],"next-turn":[]}`), 10)
			inter := interaction.New()
			inter.AttachClient(client, nil)
			paint := func() {
				source.mu.Lock()
				p := source.questionProjectionLocked()
				source.mu.Unlock()
				inter.SyncProjectedQuestionSnapshot("A", p.Questions, p.Inbox, p.QuestionsSeq, p.InboxSeq, p.QueueSeen, source.QuestionQueueObservation)
			}
			queue := func() {
				source.mu.Lock()
				defer source.mu.Unlock()
				source.applyQuestionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"inbox","seq":11,"value":{"next-step":[{"source":{"kind":"user-question-reply","callId":"call-1"}}],"next-turn":[]}}`))
			}
			discard := func() {
				source.mu.Lock()
				defer source.mu.Unlock()
				source.applyQuestionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"inbox","seq":12,"value":{"next-step":[],"next-turn":[]}}`))
			}
			paint()
			id := inter.Snapshot().Pending[0].EventID
			inter.SetQuestionSingleSelect(id, "q", "B")
			if phase == "before-click" {
				queue()
				discard()
			}
			inter.ResolveQuestions(id)
			<-started
			if phase == "while-request-pending" || phase == "still-queued-at-receipt" {
				queue()
				if phase != "still-queued-at-receipt" {
					discard()
				}
			}
			close(release)
			if phase == "while-request-pending" {
				awaitCoalescedQuestion(t, func() bool { p := inter.Snapshot().Pending; return len(p) == 1 && !p[0].Sending })
			} else {
				awaitCoalescedQuestion(t, func() bool { return inter.SnapshotForAgent("A").QuestionStatus == "submitted" })
				if len(inter.Snapshot().Pending) != 0 {
					t.Fatal("receipt exposed an answer still queued or not yet observed")
				}
				if phase == "before-click" {
					paint()
					if len(inter.Snapshot().Pending) != 0 {
						t.Fatal("pre-click queue observation was attributed to the new attempt")
					}
					return
				}
				if phase == "after-receipt" {
					queue()
				}
				discard()
				paint()
			}
			if got := inter.Snapshot(); len(got.Pending) != 1 || got.Pending[0].Sending || got.QuestionCards[id].Rows[0].Selected[0] != "B" {
				t.Fatal("coalesced discard did not restore the retained draft")
			}
		})
	}
}

func TestQuestionQueueObservationsAreClonedPrunedAndHostScoped(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "A"
	s.applyQuestionValuesLocked(json.RawMessage(`{"userQuestions":{"active":[{"callId":"q"}],"settled":[]},"inbox":{"next-step":[{"source":{"kind":"user-question-reply","callId":"q"}}],"next-turn":[]}}`), 1)
	s.applyQuestionFaceLocked("A", "inbox", json.RawMessage(`{"next-step":[],"next-turn":[]}`), 2)
	copy := s.questionProjectionLocked()
	if copy.QueueSeen["q"] == 0 {
		t.Fatal("coalesced queued observation lost")
	}
	copy.QueueSeen["q"] = 0
	if s.questionProjectionLocked().QueueSeen["q"] == 0 {
		t.Fatal("mutable UI observation map")
	}
	s.applyQuestionFaceLocked("A", "userQuestions", json.RawMessage(`{"active":[],"settled":[{"callId":"q"}]}`), 3)
	if len(s.questionProjectionLocked().QueueSeen) != 0 {
		t.Fatal("closed call accumulated observation history")
	}
	s.Attach(&dsh.Client{}, nil)
	if len(s.questionProjectionLocked().QueueSeen) != 0 {
		t.Fatal("new Host inherited queued observation")
	}
}
