package session

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
	"whaleisle/internal/dsh"
)

func TestSubmissionWatchConfirmsBackgroundSessionAndReleasesStream(t *testing.T) {
	opened := make(chan dsh.ClientFrame, 1)
	release := make(chan struct{})
	closed := make(chan struct{}, 1)
	var historical atomic.Int32
	requestID := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			historical.Add(1)
			http.Error(w, "unexpected history read", 500)
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
		opened <- open
		_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: json.RawMessage(`{"type":"snapshot","cursor":9,"hasMore":true,"records":[{"type":"event","event":{"type":"user/message","seq":9,"data":{"source":{"kind":"user","rpcId":"older"}}}}]}`)})
		<-release
		event, _ := json.Marshal(map[string]any{"type": "event", "event": map[string]any{"type": "user/message", "seq": 10, "data": map[string]any{"source": map[string]string{"kind": "user", "rpcId": requestID}}}})
		_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: event})
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			if frame.Type == dsh.FrameCancel {
				closed <- struct{}{}
				return
			}
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := newRetirementRegressionStore()
	s.client.Endpoint.Origin = srv.URL
	owner, id := beginAcceptedRetirement(t, s, "a")
	requestID = id
	s.mu.Lock()
	s.selectedID = "b"
	s.mu.Unlock()
	s.AttachMux(mux)
	persisted := make(chan SubmissionRecord, 2)
	if err := s.BindSubmissionRecovery(ctx, "profile", nil, func(record SubmissionRecord) error { persisted <- record; return nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case open := <-opened:
		var payload struct {
			Args struct {
				Request struct {
					AssistantStream json.RawMessage `json:"assistantStream"`
					Address         json.RawMessage `json:"address"`
				} `json:"request"`
			} `json:"args"`
		}
		if json.Unmarshal(open.Payload, &payload) != nil || payload.Args.Request.AssistantStream != nil || string(payload.Args.Request.Address) != `{"kind":"session","sessionId":"a"}` {
			t.Fatalf("wrong lightweight owner: %s", open.Payload)
		}
	case <-ctx.Done():
		t.Fatal("confirmation observer not started")
	}
	close(release)
	select {
	case record := <-persisted:
		if record.Phase != "confirmed" || record.RequestID != id {
			t.Fatalf("wrong result: %+v", record)
		}
	case <-ctx.Done():
		t.Fatal("confirmation was lost after selection changed")
	}
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("confirmed observer retained stream")
	}
	if historical.Load() != 0 {
		t.Fatal("live submission scanned older history before waiting for current receipt")
	}
	if err := awaitRetirementRegression(t, owner, id); err != nil {
		t.Fatal(err)
	}
	if s.SelectedSessionID() != "b" {
		t.Fatal("confirmation changed selection")
	}
}

func TestSubmissionWatchResumesOnlyForCurrentEventsReadyAfterHandshakeFailure(t *testing.T) {
	var online atomic.Bool
	var opens atomic.Int32
	requestID := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			http.Error(w, "not a mux request", 404)
			return
		}
		if !online.Load() {
			http.Error(w, "reconnecting", 503)
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
		opens.Add(1)
		raw, _ := json.Marshal(map[string]any{"type": "snapshot", "cursor": 10, "hasMore": false, "records": []any{map[string]any{"type": "event", "event": map[string]any{"type": "user/message", "seq": 10, "data": map[string]any{"source": map[string]string{"kind": "user", "rpcId": requestID}}}}}})
		_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: raw})
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil || frame.Type == dsh.FrameCancel {
				return
			}
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := newRetirementRegressionStore()
	_, requestID = beginAcceptedRetirement(t, s, "a")
	s.mu.Lock()
	s.selectedID = "b"
	s.mu.Unlock()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s.AttachMux(mux)
	confirmed := make(chan SubmissionRecord, 1)
	if err := s.BindSubmissionRecovery(ctx, "profile", nil, func(r SubmissionRecord) error { confirmed <- r; return nil }); err != nil {
		t.Fatal(err)
	}
	for {
		s.mu.Lock()
		gone := len(s.submissionWatches) == 0
		s.mu.Unlock()
		if gone {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("failed watcher retained")
		case <-time.After(time.Millisecond):
		}
	}
	stale := s.BeginEvents(ctx, s.client)
	current := s.BeginEvents(ctx, s.client)
	online.Store(true)
	s.OnEventsReady(stale)
	if opens.Load() != 0 {
		t.Fatal("stale event owner revived confirmation")
	}
	s.OnEventsReady(current)
	select {
	case record := <-confirmed:
		if record.RequestID != requestID || record.Phase != "confirmed" {
			t.Fatalf("wrong outcome: %+v", record)
		}
	case <-ctx.Done():
		t.Fatal("current ready did not resume pending receipt")
	}
	if s.SelectedSessionID() != "b" || opens.Load() != 1 {
		t.Fatalf("recovery changed selection or duplicated reads: selected=%q opens=%d", s.SelectedSessionID(), opens.Load())
	}
}
