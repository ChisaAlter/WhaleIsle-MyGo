package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestMuxOverflowClosesOnlyThatStream(t *testing.T) {
	cancelled := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx := r.Context()
		var slow, fast ClientFrame
		if wsjson.Read(ctx, c, &slow) != nil || wsjson.Read(ctx, c, &fast) != nil {
			return
		}
		for i := 0; i < streamInboxFrames+64; i++ {
			if wsjson.Write(ctx, c, ServerFrame{Type: FrameItem, StreamID: slow.StreamID, Value: json.RawMessage(`1`)}) != nil {
				return
			}
		}
		_ = wsjson.Write(ctx, c, ServerFrame{Type: FrameItem, StreamID: fast.StreamID, Value: json.RawMessage(`"fast"`)})
		for {
			var f ClientFrame
			if wsjson.Read(ctx, c, &f) != nil {
				return
			}
			switch {
			case f.Type == FrameCancel && f.StreamID == slow.StreamID:
				select {
				case cancelled <- f.StreamID:
				default:
				}
			case f.Type == FrameOpen:
				_ = wsjson.Write(ctx, c, ServerFrame{Type: FrameItem, StreamID: f.StreamID, Value: json.RawMessage(`"third"`)})
			}
		}
	}))
	defer srv.Close()
	var lost atomic.Int32
	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}, OnCarrierLost: func(error) { lost.Add(1) }}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	slow, err := mux.Open(ctx, "session/follow", nil)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := mux.Open(ctx, "workspace/follow", nil)
	if err != nil {
		t.Fatal(err)
	}
	if raw, err := fast.Recv(ctx); err != nil || string(raw) != `"fast"` {
		t.Fatalf("other stream after overflow: %s %v", raw, err)
	}
	if _, err := slow.Recv(ctx); !errors.Is(err, ErrStreamOverflow) || !Resubscribable(err) {
		t.Fatalf("overflowed stream: %v, want ErrStreamOverflow", err)
	}
	select {
	case <-cancelled:
	case <-ctx.Done():
		t.Fatal("the Host was never told to stop the overflowed stream")
	}
	third, err := mux.Open(ctx, "speech/follow", nil)
	if err != nil {
		t.Fatalf("carrier should survive one stream's overflow: %v", err)
	}
	if raw, err := third.Recv(ctx); err != nil || string(raw) != `"third"` {
		t.Fatalf("new stream on surviving carrier: %s %v", raw, err)
	}
	if lost.Load() != 0 {
		t.Fatal("overflow must not report the carrier lost")
	}
}

func rpcOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{"items": []any{}}},
		})
	})
}

// resetRequest aborts the connection with an RST, as Node does when it
// destroys a socket whose upgrade route is not registered yet.
func resetRequest(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	hj, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("hijack unsupported")
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		t.Fatal(err)
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetLinger(0)
	}
	_ = conn.Close()
}
