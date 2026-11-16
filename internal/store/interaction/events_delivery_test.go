package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func deliveryAnswer(t *testing.T, s *Store, owner *EventsDeliveryOwner, wf dsh.EventsWaterfall) <-chan *dsh.EventResultOutcome {
	t.Helper()
	done := make(chan *dsh.EventResultOutcome, 1)
	go func() {
		out, err := s.HandleWaterfallForDelivery(owner, wf)
		if err != nil {
			t.Errorf("waterfall: %v", err)
		}
		done <- out
	}()
	return done
}

func deliveryReceive(t *testing.T, done <-chan *dsh.EventResultOutcome) *dsh.EventResultOutcome {
	t.Helper()
	select {
	case out := <-done:
		if out == nil {
			t.Fatal("waterfall returned no outcome")
		}
		return out
	case <-time.After(2 * time.Second):
		t.Fatal("retired delivery did not release its answer waiter")
		return nil
	}
}

func TestEventsDeliveryReplayRetainsQuestionDraftAndReplacesWaiter(t *testing.T) {
	s := New()
	defer s.Detach()
	client, mux := &dsh.Client{}, &dsh.Mux{}
	s.AttachClient(client, mux)
	oldCtx, oldCancel := context.WithCancel(context.Background())
	defer oldCancel()
	old := s.BeginEventsDelivery(oldCtx, client, mux)
	wf := dsh.EventsWaterfall{EventID: "same", AgentID: "A", Event: EventUserQuestions, Request: json.RawMessage(`{"callId":"call-replay","questions":[{"id":"q","question":"Answer"}]}`)}
	oldAnswer := deliveryAnswer(t, s, old, wf)
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
	s.SetQuestionCustom("same", "q", "kept\n中文 😀")
	// MaintainMux can cancel the previous attempt before its FollowEvents
	// finally reaches Retire. Beginning the replay must retire it immediately.
	oldCancel()
	fresh := s.BeginEventsDelivery(context.Background(), client, mux)
	defer s.RetireEventsDelivery(fresh)
	retired := deliveryReceive(t, oldAnswer)
	if retired.Delivery == nil || retired.Delivery.Valid() || len(s.Snapshot().Pending) != 0 {
		t.Fatal("old delivery retained UI or HTTP authority")
	}
	newAnswer := deliveryAnswer(t, s, fresh, wf)
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
	if got := *s.QuestionCustomPtr("same", "q"); got != "kept\n中文 😀" {
		t.Fatalf("replay discarded the callId draft: %q", got)
	}
	// Neither an old finally, old Cancel, nor an old HTTP acknowledgement may
	// erase the fresh same-eventId card.
	s.RetireEventsDelivery(old)
	s.OnCancelForDelivery(old, dsh.EventsCancel{EventID: "same"})
	s.OnResultForDelivery(old, dsh.EventResult{EventID: "same", Outcome: *retired}, nil)
	if len(s.Snapshot().Pending) != 1 {
		t.Fatal("old callback removed replay")
	}
	lateHandler := deliveryReceive(t, deliveryAnswer(t, s, old, wf))
	if lateHandler.Delivery == nil || lateHandler.Delivery.Valid() {
		t.Fatal("old handler joined the new answer waiter")
	}
	s.ResolveQuestions("same")
	out := deliveryReceive(t, newAnswer)
	if out.Delivery == nil || !out.Delivery.Valid() {
		t.Fatal("fresh replay cannot deliver")
	}
	var answer userQuestionsAnswer
	if err := json.Unmarshal(out.Value, &answer); err != nil || len(answer.Answers) != 1 || answer.Answers[0].Custom == nil || *answer.Answers[0].Custom != "kept\n中文 😀" {
		t.Fatalf("wrong replay answer: %s (%v)", out.Value, err)
	}
	// A genuine current Host cancellation still closes this request.
	s.OnCancelForDelivery(fresh, dsh.EventsCancel{EventID: "same"})
	if len(s.Snapshot().Pending) != 0 || out.Delivery.Valid() {
		t.Fatal("real Host Cancel did not revoke delivery")
	}
}

func TestEventsDeliveryLateHTTPReceiptCannotSettleReplayedApproval(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://offline.invalid"}, HTTP: &http.Client{Transport: questionFixtureTransport(func(r *http.Request) (*http.Response, error) {
				var envelope struct {
					RPCID string `json:"rpcId"`
				}
				if e := json.NewDecoder(r.Body).Decode(&envelope); e != nil {
					return nil, e
				}
				close(entered)
				<-release // Deliberately model an already-completed late receipt.
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": map[string]any{"ok": true}})
				return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
			})}}
			s := New()
			mux := &dsh.Mux{}
			s.AttachClient(client, mux)
			defer s.Detach()
			old := s.BeginEventsDelivery(context.Background(), client, mux)
			wf := dsh.EventsWaterfall{EventID: "approval", AgentID: "A", Event: EventApproval, Request: json.RawMessage(`{"toolName":"fixture"}`)}
			oldAnswer := deliveryAnswer(t, s, old, wf)
			awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
			s.ResolveAllow("approval")
			out := deliveryReceive(t, oldAnswer)
			receipt := make(chan error, 1)
			go func() {
				receipt <- client.PostEventResult(context.Background(), dsh.EventResult{ClientID: "old", EventID: "approval", Outcome: *out})
			}()
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("offline receipt not entered")
			}
			fresh := s.BeginEventsDelivery(context.Background(), client, mux)
			defer s.RetireEventsDelivery(fresh)
			newAnswer := deliveryAnswer(t, s, fresh, wf)
			awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
			close(release)
			var err error
			select {
			case err = <-receipt:
			case <-time.After(2 * time.Second):
				t.Fatal("offline receipt not released")
			}
			if out.Delivery.Valid() || out.Delivery.Context.Err() == nil {
				t.Fatal("old in-flight HTTP context remained authorized")
			}
			if s.OnResultForDelivery(old, dsh.EventResult{EventID: "approval", Outcome: *out}, err) {
				t.Fatal("old failure requested a retry")
			}
			pending := s.Snapshot().Pending
			if len(pending) != 1 || pending[0].Sending || pending[0].Error != "" {
				t.Fatalf("late old HTTP mutated replay: %+v", pending)
			}
			s.ResolveDeny("approval")
			current := deliveryReceive(t, newAnswer)
			// Owner alone is insufficient: a mismatched response proof is stale.
			s.OnResultForDelivery(fresh, dsh.EventResult{EventID: "approval", Outcome: *out}, errors.New("old response"))
			pending = s.Snapshot().Pending
			if len(pending) != 1 || !pending[0].Sending || pending[0].Error != "" {
				t.Fatal("old proof changed current owner's decision receipt")
			}
			s.OnResultForDelivery(fresh, dsh.EventResult{EventID: "approval", Outcome: *current}, nil)
			if len(s.Snapshot().Pending) != 0 {
				t.Fatal("current decision did not settle")
			}
		})
	}
}

func TestEventsDeliveryDuplicateEventIDPumpExitsCleanly(t *testing.T) {
	framesDelivered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			http.NotFound(w, r)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if err := wsjson.Read(r.Context(), c, &open); err != nil {
			return
		}
		_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{
			Type:     dsh.FrameItem,
			StreamID: open.StreamID,
			Value:    json.RawMessage(`{"type":"ready","clientId":"c1","host":{"home":"H"}}`),
		})
		for range 2 {
			_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{
				Type:     dsh.FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"waterfall","event":"approval/request","eventId":"dup-1","agentId":"A","request":{"toolName":"bash"}}`),
			})
		}
		close(framesDelivered)
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()

	s := New()
	defer s.Detach()
	s.AttachClient(client, mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := s.BeginEventsDelivery(ctx, client, mux)

	done := make(chan error, 1)
	go func() {
		done <- client.FollowEvents(ctx, mux, dsh.EventsFuncs{
			Waterfall: func(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
				return s.HandleWaterfallForDelivery(owner, wf)
			},
			Result: func(res dsh.EventResult, err error) bool {
				return s.OnResultForDelivery(owner, res, err)
			},
			Cancel: func(ev dsh.EventsCancel) {
				s.OnCancelForDelivery(owner, ev)
			},
			Retire: func() {
				s.RetireEventsDelivery(owner)
			},
		})
	}()

	select {
	case <-framesDelivered:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for duplicate waterfall frames")
	}
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("FollowEvents pump hung in tasks.Wait() on duplicate EventID")
	}
}

func TestEmptyEventIDDoesNotWipeCrossSessionPending(t *testing.T) {
	s := New()
	defer s.Detach()
	client, mux := &dsh.Client{}, &dsh.Mux{}
	s.AttachClient(client, mux)
	owner := s.BeginEventsDelivery(context.Background(), client, mux)
	defer s.RetireEventsDelivery(owner)

	wf1 := dsh.EventsWaterfall{EventID: "ev-session-a", AgentID: "A", Event: EventApproval, Request: json.RawMessage(`{"toolName":"bash"}`)}
	wf2 := dsh.EventsWaterfall{EventID: "ev-session-b", AgentID: "B", Event: EventUserQuestions, Request: json.RawMessage(`{"callId":"c-b","questions":[{"id":"q1","question":"Keep?"}]}`)}
	ans1 := deliveryAnswer(t, s, owner, wf1)
	ans2 := deliveryAnswer(t, s, owner, wf2)
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 2 })

	// Empty EventID frames / results must be rejected and never trigger dropPending("").
	s.OnCancel(dsh.EventsCancel{EventID: ""})
	s.OnCancelForDelivery(owner, dsh.EventsCancel{EventID: "   "})
	if s.OnResult(dsh.EventResult{EventID: ""}, nil) {
		t.Fatal("empty result EventID requested retry")
	}
	if s.OnResultForDelivery(owner, dsh.EventResult{EventID: ""}, errors.New("boom")) {
		t.Fatal("empty delivery result EventID requested retry")
	}
	outEmpty, err := s.HandleWaterfallForDelivery(owner, dsh.EventsWaterfall{EventID: "", AgentID: "C", Event: EventApproval})
	if err != nil || outEmpty == nil || (outEmpty.Delivery != nil && outEmpty.Delivery.Valid()) {
		t.Fatalf("empty waterfall EventID unexpected outcome=%+v err=%v", outEmpty, err)
	}

	snap := s.Snapshot()
	if len(snap.Pending) != 2 || len(snap.QuestionCards) != 1 {
		t.Fatalf("empty EventID cleared unrelated pending/question state: pending=%+v cards=%+v", snap.Pending, snap.QuestionCards)
	}

	s.RetireEventsDelivery(owner)
	_ = deliveryReceive(t, ans1)
	_ = deliveryReceive(t, ans2)
}
