package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestClientFrameValidate(t *testing.T) {
	if err := (ClientFrame{Type: FrameOpen, StreamID: "a", Endpoint: "$events"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (ClientFrame{Type: FrameOpen, StreamID: "a"}).Validate(); err == nil {
		t.Fatal("expected error")
	}
	if err := (ServerFrame{Type: FrameError, StreamID: "a", Error: &StreamFailure{Code: "x", Message: "y"}}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMuxOpenItemEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != MuxPath {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.Header.Get("Cookie"); got != "dsh-auth=1" {
			t.Fatalf("cookie %q", got)
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.CloseNow()
		var open ClientFrame
		if err := wsjson.Read(r.Context(), c, &open); err != nil {
			t.Fatal(err)
		}
		if open.Type != FrameOpen || open.Endpoint != "session/follow" {
			t.Fatalf("open %#v", open)
		}
		_ = wsjson.Write(r.Context(), c, ServerFrame{
			Type:     FrameItem,
			StreamID: open.StreamID,
			Value:    json.RawMessage(`{"hello":1}`),
		})
		_ = wsjson.Write(r.Context(), c, ServerFrame{
			Type:     FrameEnd,
			StreamID: open.StreamID,
		})
		// Keep socket open briefly so client can finish.
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL, Cookie: "dsh-auth=1"}}
	ctx := context.Background()
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()

	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := stream.Recv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"hello":1}` {
		t.Fatalf("value %s", raw)
	}
	_, err = stream.Recv(ctx)
	if !errors.Is(err, ErrStreamEnd) {
		t.Fatalf("end = %v", err)
	}
}

func TestMuxStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open ClientFrame
		_ = wsjson.Read(r.Context(), c, &open)
		_ = wsjson.Write(r.Context(), c, ServerFrame{
			Type:     FrameError,
			StreamID: open.StreamID,
			Error:    &StreamFailure{Code: "DENIED", Message: "nope", Details: json.RawMessage(`{}`)},
		})
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	ctx := context.Background()
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	stream, err := mux.Open(ctx, "x/y", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = stream.Recv(ctx)
	var fail *StreamFailure
	if !errors.As(err, &fail) || fail.Code != "DENIED" {
		t.Fatalf("got %v", err)
	}
}

func TestFollowEventsWaterfallNext(t *testing.T) {
	var resultBody atomic.Value
	muxDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == EventsResultPath && r.Method == http.MethodPost:
			var body EventResult
			_ = json.NewDecoder(r.Body).Decode(&body)
			resultBody.Store(body)
			w.WriteHeader(http.StatusNoContent)
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
			if open.Endpoint != EventsEndpoint {
				t.Errorf("endpoint %s", open.Endpoint)
			}
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"ready","clientId":"c1","host":{"home":"H"}}`),
			})
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameItem,
				StreamID: open.StreamID,
				Value:    json.RawMessage(`{"type":"emit","event":"ping","args":[]}`),
			})
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameItem,
				StreamID: open.StreamID,
				Value: json.RawMessage(
					`{"type":"waterfall","event":"approval/request","eventId":"e1","agentId":"a1","request":{"x":1}}`,
				),
			})
			_ = wsjson.Write(r.Context(), c, ServerFrame{
				Type:     FrameEnd,
				StreamID: open.StreamID,
			})
			close(muxDone)
			time.Sleep(100 * time.Millisecond)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	mux := &Mux{Endpoint: client.Endpoint}
	var gotReady EventsReady
	var gotEmit EventsEmit
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := client.FollowEvents(ctx, mux, EventsFuncs{
		Ready: func(r EventsReady) { gotReady = r },
		Emit:  func(e EventsEmit) { gotEmit = e },
		// Waterfall nil → automatic next
	})
	if err != nil {
		t.Fatal(err)
	}
	<-muxDone
	if gotReady.ClientID != "c1" || gotReady.Host.Home != "H" {
		t.Fatalf("ready %#v", gotReady)
	}
	if gotEmit.Event != "ping" {
		t.Fatalf("emit %#v", gotEmit)
	}
	raw, _ := resultBody.Load().(EventResult)
	if raw.ClientID != "c1" || raw.EventID != "e1" || raw.Outcome.Kind != "next" {
		t.Fatalf("result %#v", raw)
	}
	_ = mux.Close()
}

func TestMuxURL(t *testing.T) {
	u, err := muxURL("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "ws://127.0.0.1:9") || !strings.HasSuffix(u, MuxPath) {
		t.Fatalf("url %s", u)
	}
}

func TestEventResultNext(t *testing.T) {
	r := EventResultNext("c", "e")
	if r.Outcome.Kind != "next" || r.ClientID != "c" {
		t.Fatalf("%#v", r)
	}
}
