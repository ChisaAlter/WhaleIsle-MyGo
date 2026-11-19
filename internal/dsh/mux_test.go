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
	resultDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == EventsResultPath && r.Method == http.MethodPost:
			body, rpcID := decodeEventResultRPC(t, r)
			resultBody.Store(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": rpcID, "result": map[string]any{"ok": true}})
			close(resultDone)
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
			<-resultDone
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

func TestMuxStreamWatcherExitsOnFrameEnd(t *testing.T) {
	var cancelCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			Type:     FrameEnd,
			StreamID: open.StreamID,
		})
		for {
			var f ClientFrame
			if err := wsjson.Read(r.Context(), c, &f); err != nil {
				return
			}
			if f.Type == FrameCancel {
				cancelCount.Add(1)
			}
		}
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	if err := mux.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()

	stream, err := mux.Open(context.Background(), "session/follow", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(context.Background()); !errors.Is(err, ErrStreamEnd) {
		t.Fatalf("Recv = %v, want ErrStreamEnd", err)
	}
	select {
	case <-stream.streamDone:
	case <-time.After(time.Second):
		t.Fatal("streamDone was not closed after FrameEnd")
	}
	time.Sleep(30 * time.Millisecond)
	if n := cancelCount.Load(); n != 0 {
		t.Fatalf("unexpected FrameCancel count %d after normal FrameEnd", n)
	}
}

func TestMuxOnCarrierLostCanCloseWithoutDeadlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		_ = c.CloseNow()
	}))
	defer srv.Close()

	closedFromCB := make(chan struct{})
	var mux *Mux
	mux = &Mux{
		Endpoint: Endpoint{Origin: srv.URL},
		OnCarrierLost: func(error) {
			_ = mux.Close()
			close(closedFromCB)
		},
	}
	if err := mux.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closedFromCB:
	case <-time.After(3 * time.Second):
		t.Fatal("OnCarrierLost deadlocked calling mux.Close()")
	}
}

func TestMuxConcurrentRecvAndCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			var f ClientFrame
			if err := wsjson.Read(r.Context(), c, &f); err != nil {
				return
			}
			if f.Type == FrameOpen {
				_ = wsjson.Write(r.Context(), c, ServerFrame{Type: FrameEnd, StreamID: f.StreamID})
			}
		}
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	if err := mux.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()

	for range 25 {
		s, err := mux.Open(context.Background(), "session/follow", nil)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.Cancel()
		}()
		_, _ = s.Recv(context.Background())
		<-done
	}
}

func TestMuxCancelSurvivesStreamContextCancel(t *testing.T) {
	cancels := make(chan string, 4)
	paused := make(chan struct{})
	resume := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		c.SetReadLimit(64 << 20)
		for range 2 {
			var open ClientFrame
			if err := wsjson.Read(r.Context(), c, &open); err != nil {
				return
			}
		}
		close(paused)
		<-resume
		for {
			var f ClientFrame
			if err := wsjson.Read(r.Context(), c, &f); err != nil {
				return
			}
			if f.Type == FrameCancel {
				select {
				case cancels <- f.StreamID:
				default:
				}
			}
		}
	}))
	defer srv.Close()

	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	if err := mux.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()

	watch, err := mux.Open(context.Background(), "account/watch", nil)
	if err != nil {
		t.Fatal(err)
	}
	follow, err := mux.Open(context.Background(), "job/follow", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-paused

	// A write that is still in flight when its stream context is cancelled is the
	// shape that used to take the whole carrier down: the library keeps the
	// context armed for the length of the write and closes the socket when it ends.
	followCtx, followCancel := context.WithCancel(context.Background())
	defer followCancel()
	big := map[string]any{"blob": strings.Repeat("x", 4<<20)}
	writeDone := make(chan error, 1)
	go func() { writeDone <- follow.SendItem(followCtx, big) }()
	time.Sleep(100 * time.Millisecond)
	followCancel()
	time.Sleep(200 * time.Millisecond)
	close(resume)

	if err := <-writeDone; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("SendItem with a cancelled context = %v", err)
	}

	watch.Cancel()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case id := <-cancels:
			if id == watch.id {
				return
			}
		case <-deadline:
			t.Fatal("sibling cancel frame never reached the Host: one stream's cancelled context took the shared carrier down")
		}
	}
}
