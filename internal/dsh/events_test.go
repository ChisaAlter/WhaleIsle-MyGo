package dsh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestFollowEventsDuplicateWaterfallDoesNotClobberPrimaryCancel(t *testing.T) {
	primaryPosting := make(chan struct{})
	primaryCanceled := make(chan struct{})
	dupFinished := make(chan struct{})
	var callNum atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == EventsResultPath && r.Method == http.MethodPost:
			body, rpcID := decodeEventResultRPC(t, r)
			if body.Outcome.Kind == "result" {
				// Primary attempt blocks in PostEventResult until EventCancel cancels its ctx.
				close(primaryPosting)
				<-r.Context().Done()
				close(primaryCanceled)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  rpcID,
				"result": map[string]any{"ok": true},
			})
			close(dupFinished)
		case r.URL.Path == MuxPath:
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			var open ClientFrame
			if err := wsjson.Read(r.Context(), c, &open); err != nil {
				return
			}
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"ready","clientId":"c1","host":{"home":"H"}}`),
			})
			wfJSON := json.RawMessage(`{"type":"waterfall","event":"approval/request","eventId":"dup-1","agentId":"a1","request":{"x":1}}`)
			// Send primary waterfall, then duplicate with the same EventID.
			_ = wsjson.Write(r.Context(), c, ServerFrame{Type: FrameItem, StreamID: open.StreamID, Value: wfJSON})
			<-primaryPosting
			_ = wsjson.Write(r.Context(), c, ServerFrame{Type: FrameItem, StreamID: open.StreamID, Value: wfJSON})
			// Wait for the duplicate attempt to finish and clean up its own token.
			<-dupFinished
			// Now send EventCancel: it must still cancel the primary PostEventResult!
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"cancel","eventId":"dup-1"}`),
			})
			<-primaryCanceled
			_ = wsjson.Write(r.Context(), c, ServerFrame{Type: FrameEnd, StreamID: open.StreamID})
			time.Sleep(50 * time.Millisecond)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	mux := &Mux{Endpoint: client.Endpoint, PingInterval: -1}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.FollowEvents(ctx, mux, EventsFuncs{
		Waterfall: func(wf EventsWaterfall) (*EventResultOutcome, error) {
			if callNum.Add(1) == 1 {
				return &EventResultOutcome{Kind: "result", Value: json.RawMessage(`{"ok":true}`)}, nil
			}
			return &EventResultOutcome{Kind: "next"}, nil
		},
	})
	if err != nil {
		t.Fatalf("FollowEvents: %v", err)
	}
	select {
	case <-primaryCanceled:
	default:
		t.Fatal("primary waterfall PostEventResult was not canceled by EventCancel")
	}
	_ = mux.Close()
}

func BenchmarkEventsDispatchEmit(b *testing.B) {
	raw := json.RawMessage(`{"type":"emit","event":"session/updated","args":[{"id":"s1","seq":42}]}`)
	sess := &EventsSession{}
	handler := EventsFuncs{Emit: func(EventsEmit) {}}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sess.dispatch(ctx, handler, raw); err != nil {
			b.Fatal(err)
		}
	}
}
