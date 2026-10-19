package session

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

func TestAcceptedFollowEOFRestartsOnlyItsCurrentSelection(t *testing.T) {
	for _, change := range []string{"current", "different-session", "same-session-generation", "canceled"} {
		t.Run(change, func(t *testing.T) {
			opens := make(chan dsh.ClientFrame, 1)
			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.CloseNow()
				var open dsh.ClientFrame
				if wsjson.Read(r.Context(), conn, &open) != nil {
					return
				}
				opens <- open
				_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: json.RawMessage(`{"type":"snapshot","cursor":7,"records":[{"type":"event","event":{"type":"user/message","seq":7,"data":{"content":[{"type":"text","text":"retained A"}]}}}]}`)})
				<-release
				_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameEnd, StreamID: open.StreamID})
				var cancel dsh.ClientFrame
				_ = wsjson.Read(r.Context(), conn, &cancel)
			}))
			defer srv.Close()
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			defer mux.Close()
			s := New()
			s.selectedID = "A"
			s.historyLoading = true
			generation := s.followGeneration
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan bool, 1)
			go func() { done <- s.runFollowConnection(ctx, mux, "A", generation) }()
			var open dsh.ClientFrame
			select {
			case open = <-opens:
			case <-ctx.Done():
				t.Fatal("offline follow did not open")
			}
			var payload struct {
				Args struct {
					Request struct {
						Address         json.RawMessage                     `json:"address"`
						AssistantStream bool                                `json:"assistantStream"`
						MaxMessages     int                                 `json:"maxMessages"`
						TurnWindow      struct{ MinMessages, MinTurns int } `json:"turnWindow"`
					} `json:"request"`
				} `json:"args"`
			}
			if err := json.Unmarshal(open.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			q := payload.Args.Request
			if open.Endpoint != "session/follow" || string(q.Address) != `{"kind":"session","sessionId":"A"}` || !q.AssistantStream || q.MaxMessages != 500 || q.TurnWindow.MinMessages != 50 || q.TurnWindow.MinTurns != 2 {
				t.Fatalf("wrong opening follow contract: %s", open.Payload)
			}
			deadline := time.Now().Add(2 * time.Second)
			for !s.Snapshot().HistoryReady {
				if time.Now().After(deadline) {
					t.Fatal("snapshot not accepted")
				}
				time.Sleep(time.Millisecond)
			}
			s.mu.Lock()
			switch change {
			case "different-session":
				s.selectSessionLocked("B")
				s.messages = []Message{{ID: "B", Body: "new B"}}
			case "same-session-generation":
				s.invalidateFollowLocked()
				s.messages = []Message{{ID: "new-A", Body: "new A"}}
			case "canceled":
				cancel()
			}
			s.mu.Unlock()
			close(release)
			select {
			case restart := <-done:
				if restart != (change == "current") {
					t.Fatalf("EOF restart=%v after %s", restart, change)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("EOF did not settle")
			}
			if change == "current" {
				v := s.Snapshot()
				if len(v.Messages) != 1 || v.Messages[0].Body != "retained A" || !v.HistoryReady {
					t.Fatal("accepted EOF discarded retained baseline")
				}
			}
		})
	}
}

func TestFollowRetiredBeforeRetryDoesNotDialOrOpen(t *testing.T) {
	s := New()
	s.selectedID = "B"
	// A stale reconnect must leave before touching even a nil transport.
	if s.runFollowConnection(context.Background(), nil, "A", s.followGeneration) {
		t.Fatal("retired target restarted")
	}
}
