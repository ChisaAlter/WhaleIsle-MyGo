package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// A frame the read loop already queued is Host data the caller must still see:
// losing it drops a snapshot the Host had delivered before its socket died.
func TestMuxStreamRecvKeepsFrameQueuedBeforeCarrierLost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open ClientFrame
		if wsjson.Read(r.Context(), c, &open) != nil {
			return
		}
		_ = wsjson.Write(r.Context(), c, ServerFrame{
			Type:     FrameItem,
			StreamID: open.StreamID,
			Value:    json.RawMessage(`{"type":"snapshot","cursor":11}`),
		})
		c.CloseNow()
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	defer mux.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	stream, err := mux.Open(ctx, "session/follow", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-stream.done:
	case <-ctx.Done():
		t.Fatal("carrier never died")
	}
	if len(stream.inbox) != 1 {
		t.Fatalf("queued frames = %d, want the snapshot the Host sent before it closed", len(stream.inbox))
	}
	raw, err := stream.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv dropped the frame queued before the carrier died: %v", err)
	}
	if string(raw) != `{"type":"snapshot","cursor":11}` {
		t.Fatalf("frame = %s", raw)
	}
	if _, err := stream.Recv(ctx); !errors.Is(err, ErrCarrierLost) {
		t.Fatalf("Recv after the drained frame = %v, want ErrCarrierLost", err)
	}
}
