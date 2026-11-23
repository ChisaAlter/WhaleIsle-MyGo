package session

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"whaleisle/internal/dsh"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func resetConn(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		t.Fatal(err)
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetLinger(0)
	}
	_ = conn.Close()
}

// TestSessionListSelfHealsAfterResetFirstPost is the sidebar's raw
// "wsarecv: connection forcibly closed": the first session/list lands on a
// socket that resets. The list retries and loads; nothing reaches the view.
func TestSessionListSelfHealsAfterResetFirstPost(t *testing.T) {
	var lists atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/list") && lists.Add(1) == 1 {
			resetConn(t, w)
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "A", "updatedAt": 1}}})
	}))
	defer srv.Close()
	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	s.OnEventsReady(s.BeginEvents(context.Background(), client))
	waitFor(t, "list", func() bool { return !s.Snapshot().ListLoading })
	v := s.Snapshot()
	if v.ListErr != nil || len(v.Items) != 1 || v.Items[0].SessionID != "A" {
		t.Fatalf("list did not heal: err=%v items=%d", v.ListErr, len(v.Items))
	}
	if lists.Load() != 2 {
		t.Fatalf("list requests=%d, want the reset one and its retry", lists.Load())
	}
}

func TestSessionListFailureIsAFaultWithoutRawText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret-host-detail at C:\\Users\\someone", http.StatusBadRequest)
	}))
	defer srv.Close()
	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	s.RefreshList(context.Background())
	waitFor(t, "list failure", func() bool { return s.Snapshot().ListErr != nil })
	a, b := s.Snapshot(), s.Snapshot()
	if a.ListErr != b.ListErr || s.SelectionSummary().ListErr != a.ListErr {
		t.Fatal("every view of one failure must share one Fault, or per-frame comparisons never settle")
	}
	sentence := a.ListErr.SentenceText()
	if sentence == "" || strings.Contains(sentence, "secret-host-detail") || strings.Contains(sentence, "HTTP 400") {
		t.Fatalf("list fault sentence %q carries raw text", sentence)
	}
	if a.ListErr.Code != "HTTP_400" {
		t.Fatalf("fault code %q, want HTTP_400", a.ListErr.Code)
	}
}

// muxFake answers each open of endpoint with fail first, then good.
func muxFake(t *testing.T, endpoint string, failures int, good json.RawMessage, opens *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx := r.Context()
		for {
			var f dsh.ClientFrame
			if wsjson.Read(ctx, c, &f) != nil {
				return
			}
			if f.Type != dsh.FrameOpen || f.Endpoint != endpoint {
				continue
			}
			if int(opens.Add(1)) <= failures {
				_ = wsjson.Write(ctx, c, dsh.ServerFrame{Type: dsh.FrameError, StreamID: f.StreamID,
					Error: &dsh.StreamFailure{Code: "gateway/invocation-unavailable", Message: "service is starting"}})
				continue
			}
			_ = wsjson.Write(ctx, c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: f.StreamID, Value: good})
		}
	}))
}

func TestSpeechFollowResubscribesOnInvocationUnavailable(t *testing.T) {
	var opens atomic.Int32
	srv := muxFake(t, "speech/follow", 2, json.RawMessage(`{"providers":[],"selection":{"providerId":"local"}}`), &opens)
	defer srv.Close()
	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	defer mux.Close()
	s.AttachMux(mux)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.WatchSpeechCatalog(ctx)
	waitFor(t, "speech catalog after the service started", func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.speechReadiness.view.Connected && s.speechSelection == "local"
	})
	if opens.Load() != 3 {
		t.Fatalf("speech/follow opened %d times, want two unavailable answers and one live stream", opens.Load())
	}
	s.mu.Lock()
	connErr := s.speechReadiness.view.ConnectionError
	s.mu.Unlock()
	if connErr != "" {
		t.Fatalf("a transient endpoint answer surfaced as %q", connErr)
	}
}

func TestWorkspaceFollowResubscribesOnInvocationUnavailable(t *testing.T) {
	var opens atomic.Int32
	baseline := json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"","items":[],"pinnedSessionIds":[],"archivedSessionIds":[]}}`)
	srv := muxFake(t, "workspace/follow", 1, baseline, &opens)
	defer srv.Close()
	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	defer mux.Close()
	s.AttachMux(mux)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.FollowWorkspace(ctx)
	waitFor(t, "workspace baseline", func() bool { return s.Snapshot().WorkspaceReady })
	if v := s.Snapshot(); v.WorkspaceError != nil {
		t.Fatalf("transient endpoint answer left a workspace fault: %+v", v.WorkspaceError)
	}
	if opens.Load() != 2 {
		t.Fatalf("workspace/follow opened %d times, want 2", opens.Load())
	}
}
