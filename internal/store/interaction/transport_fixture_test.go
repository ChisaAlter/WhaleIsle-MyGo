package interaction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// This fixture talks only to an in-process fake gateway. No Host or tool runs.
func TestInteractionDeliveryFixture(t *testing.T) {
	frames := make(chan string, 8)
	stop := make(chan struct{})
	var mu sync.Mutex
	var posts []dsh.EventResult
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.EventsResultPath {
			var body dsh.EventResult
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			posts = append(posts, body)
			n := len(posts)
			mu.Unlock()
			if body.EventID == "q" && n == 1 {
				http.Error(w, "temporary failure", 503)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path != dsh.MuxPath {
			http.NotFound(w, r)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), conn, &open) != nil {
			return
		}
		go func() {
			for {
				var frame dsh.ClientFrame
				if wsjson.Read(r.Context(), conn, &frame) != nil {
					return
				}
				if frame.Endpoint == "userQuestions/attachWait" {
					_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: json.RawMessage(`{"remainingMs":0}`)})
				}
			}
		}()
		for {
			select {
			case raw := <-frames:
				if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: json.RawMessage(raw)}) != nil {
					return
				}
			case <-stop:
				return
			}
		}
	}))
	defer srv.Close()
	defer close(stop)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.AttachClient(client, mux)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- client.FollowEvents(ctx, mux, dsh.EventsFuncs{Waterfall: s.HandleWaterfall, Cancel: s.OnCancel, Result: s.OnResult})
	}()
	wait := func(check func() bool) {
		t.Helper()
		for !check() {
			select {
			case <-ctx.Done():
				t.Fatal("fixture timed out")
			case <-time.After(time.Millisecond):
			}
		}
	}
	frames <- `{"type":"ready","clientId":"client-A","host":{"home":"fixture"}}`
	frames <- `{"type":"waterfall","event":"user-questions/request","eventId":"q","agentId":"A","request":{"callId":"call-q","questions":[{"id":"choice","options":[{"label":"Yes"}]}]}}`
	frames <- `{"type":"waterfall","event":"approval/request","eventId":"cancel-me","agentId":"B","request":{"toolName":"fixture"}}`
	wait(func() bool { return len(s.Snapshot().Pending) == 2 })
	if len(s.SnapshotForAgent("A").Pending) != 1 || len(s.SnapshotForAgent("B").Pending) != 1 || len(s.SnapshotForAgent("").Pending) != 0 {
		t.Fatal("conversation ownership lost")
	}
	// Cancellation must arrive while a different prompt still waits for input.
	frames <- `{"type":"cancel","eventId":"cancel-me"}`
	wait(func() bool { return len(s.SnapshotForAgent("B").Pending) == 0 })
	s.SetQuestionCustom("q", "choice", "custom answer")
	s.ResolveQuestions("q")
	s.ResolveQuestions("q")
	wait(func() bool { p := s.SnapshotForAgent("A").Pending; return len(p) == 1 && p[0].Error != "" })
	if len(s.Snapshot().QuestionCards["q"].Rows) != 1 || !s.QuestionsCanSubmit("q") {
		t.Fatal("failed delivery discarded draft")
	}
	s.Retry("q")
	wait(func() bool { return len(s.Snapshot().Pending) == 0 })
	mu.Lock()
	got := append([]dsh.EventResult(nil), posts...)
	mu.Unlock()
	if len(got) != 2 || got[0].ClientID != "client-A" || got[0].EventID != "q" || string(got[0].Outcome.Value) != string(got[1].Outcome.Value) {
		t.Fatalf("retry identity or response changed: %#v", got)
	}
	var answer userQuestionsAnswer
	if err := json.Unmarshal(got[1].Outcome.Value, &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Answers) != 1 || answer.Answers[0].Custom == nil || *answer.Answers[0].Custom != "custom answer" || len(answer.Answers[0].Selected) != 0 {
		t.Fatalf("answer %#v", answer)
	}
	frames <- `{"type":"waterfall","event":"approval/request","eventId":"deny","agentId":"A","request":{"toolName":"fixture"}}`
	wait(func() bool { return len(s.Snapshot().Pending) == 1 })
	s.ResolveDeny("deny")
	wait(func() bool { return len(s.Snapshot().Pending) == 0 })
	mu.Lock()
	last := posts[len(posts)-1]
	mu.Unlock()
	if string(last.Outcome.Value) != `"rejected"` || s.Snapshot().AutoAllow {
		t.Fatalf("denial or permission changed: %#v", last)
	}
	frames <- `{"type":"waterfall","event":"approval/request","eventId":"allow","agentId":"A","request":{"toolName":"fixture","displayReason":{"en":"localized reason"}}}`
	wait(func() bool { return len(s.Snapshot().Pending) == 1 })
	if s.Snapshot().Pending[0].Detail != "localized reason" {
		t.Fatal("localized approval reason lost")
	}
	s.ResolveAllow("allow")
	wait(func() bool { return len(s.Snapshot().Pending) == 0 })
	mu.Lock()
	last = posts[len(posts)-1]
	mu.Unlock()
	if string(last.Outcome.Value) != `"allowed-once"` {
		t.Fatalf("allow %#v", last)
	}
	frames <- `{"type":"waterfall","event":"user-questions/request","eventId":"timed","agentId":"A","request":{"wait":{"callId":"timed-call","timed":true},"questions":[{"id":"q","question":"Timed"}]}}`
	wait(func() bool { mu.Lock(); defer mu.Unlock(); return posts[len(posts)-1].EventID == "timed" })
	wait(func() bool { return len(s.Snapshot().Pending) == 0 })
	mu.Lock()
	last = posts[len(posts)-1]
	mu.Unlock()
	if last.Outcome.Error == nil || last.Outcome.Error.Code != "ASK_TIMED_OUT" {
		t.Fatalf("timeout %#v", last)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event loop did not stop")
	}
}
