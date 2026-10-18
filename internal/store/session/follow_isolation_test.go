package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// Delay the real WebSocket handshake, rather than relying on scheduling/sleeps.
// This reproduces an A Dial/Open that completes after B is already rendered.
func TestFollowLateOpenCannotClearNewSelection(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), c, &open) != nil {
			return
		}
		if open.Endpoint != "session/follow" {
			t.Errorf("wrong endpoint: %s", open.Endpoint)
			return
		}
		for _, raw := range []string{
			`{"type":"snapshot","cursor":90,"hasMore":false,"records":[{"type":"event","event":{"type":"user/message","seq":90,"time":1,"data":{"content":[{"type":"text","text":"late A"}]}}}]}`,
			`{"type":"event","event":{"type":"plan/enter","seq":91,"time":1,"data":{}}}`,
			`{"type":"assistant-stream","frame":{"type":"chunk","chunk":"late A delta"}}`,
		} {
			if wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: json.RawMessage(raw)}) != nil {
				return
			}
		}
		_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameEnd, StreamID: open.StreamID})
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), c, &frame) != nil {
				return
			}
		}
	}))
	defer srv.Close()
	defer releaseOnce.Do(func() { close(release) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := New()
	s.selectedID = "A"
	generation := s.followGeneration
	finished := make(chan struct{})
	go func() { defer close(finished); s.runFollow(ctx, mux, "A", generation) }()
	wait(t, started)
	s.Select("B")
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":2,"hasMore":true,"records":[{"type":"event","event":{"type":"user/message","seq":2,"time":1,"data":{"content":[{"type":"text","text":"B"}]}}}]}`))
	s.mu.Lock()
	s.planBoard = PlanBoard{PlanMode: true, Todos: []PlanTodo{{Content: "B plan"}}}
	s.mu.Unlock()
	before := s.Snapshot()
	releaseOnce.Do(func() { close(release) })
	wait(t, finished)
	got := s.Snapshot()
	if !reflect.DeepEqual(got.Messages, before.Messages) || !reflect.DeepEqual(got.PlanBoard, before.PlanBoard) || got.HistoryHasMore != before.HistoryHasMore || s.historyThroughSeq != 2 {
		t.Fatalf("late A Open/frame mutated B: messages=%+v plan=%+v history=%v through=%v", got.Messages, got.PlanBoard, got.HistoryHasMore, s.historyThroughSeq)
	}
}

func TestFollowSameSessionGenerationAndCanceledFramesAreDiscarded(t *testing.T) {
	for _, kind := range []string{"same-session-restart", "A-B-A", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			s := New()
			s.Select("A")
			generation := s.followGeneration
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "same-session-restart":
				s.mu.Lock()
				s.invalidateFollowLocked()
				s.mu.Unlock()
			case "A-B-A":
				s.Select("B")
				s.Select("A")
			case "canceled":
				cancel()
			}
			s.mu.Lock()
			s.messages = []Message{{ID: "new", Body: "new A"}}
			s.planBoard = PlanBoard{PlanMode: true, Todos: []PlanTodo{{Content: "new A"}}}
			s.historyThroughSeq = 7
			s.historyHasMore = true
			s.mu.Unlock()
			before := s.Snapshot()
			for _, raw := range []string{
				`{"type":"snapshot","records":[]}`, `{"type":"event","event":{"type":"turn/start","seq":90,"time":1,"data":{}}}`,
				`{"type":"assistant-stream","frame":{"type":"start"}}`, `{"type":"assistant-stream","frame":{"type":"chunk","chunk":"old"}}`,
			} {
				if s.applyFollowFrameForTarget(ctx, "A", generation, json.RawMessage(raw)) {
					t.Fatal("stale/canceled target was accepted")
				}
			}
			if got := s.Snapshot(); !reflect.DeepEqual(got, before) || s.historyThroughSeq != 7 {
				t.Fatalf("old generation mutated latest A: %+v", got)
			}
		})
	}
}

func TestFollowSnapshotUsesWireHistoryCursorAndHasMore(t *testing.T) {
	s := New()
	s.historyThroughSeq = 900
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":12,"hasMore":false,"records":[{"type":"event","event":{"type":"user/message","seq":8,"data":{"content":[{"type":"text","text":"hi"}]}}}]}`))
	if got := s.Snapshot(); got.HistoryHasMore || s.historyThroughSeq != 12 {
		t.Fatalf("wire opening history not authoritative: %+v cursor=%v", got, s.historyThroughSeq)
	}
	before := s.Snapshot()
	for _, raw := range []string{`{`, `null`, `{"type":"snapshot","records":null}`, `{"type":"event","event":null}`, `{"type":"future"}`} {
		s.applyFollowFrame(json.RawMessage(raw))
	}
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("malformed follow changed state: %+v", got)
	}
}
