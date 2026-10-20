package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
)

// One new carrier fixture exercises the live navigator rather than replaying
// old navigation/Workflow suites. No real Host, model, child or tool is started.
func TestNativeAgentLiveNavigationFixture(t *testing.T) {
	type liveStream struct {
		conn *websocket.Conn
		id   string
	}
	var writes sync.Mutex
	send := func(stream liveStream, raw string) {
		t.Helper()
		writes.Lock()
		defer writes.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := wsjson.Write(ctx, stream.conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: stream.id, Value: json.RawMessage(raw)}); err != nil {
			t.Errorf("fixture send: %v", err)
		}
	}
	opened := make(chan liveStream, 8)
	canceled := make(chan string, 8)
	pageStarted := make(chan struct{}, 1)
	releasePage := make(chan struct{})
	var follows, notifies atomic.Int32
	var latestMu sync.Mutex
	var latest liveStream
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			for {
				var f dsh.ClientFrame
				if wsjson.Read(r.Context(), c, &f) != nil {
					return
				}
				if f.Type == dsh.FrameCancel {
					canceled <- f.StreamID
					continue
				}
				if f.Type != dsh.FrameOpen || f.Endpoint != "session/follow" {
					t.Errorf("unexpected stream %+v", f)
					return
				}
				var envelope struct {
					Args struct {
						Request struct {
							Address         AgentSessionAddress `json:"address"`
							AssistantStream bool                `json:"assistantStream"`
						} `json:"request"`
					} `json:"args"`
				}
				_ = json.Unmarshal(f.Payload, &envelope)
				address := envelope.Args.Request.Address
				if !envelope.Args.Request.AssistantStream || address != (AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}) {
					t.Errorf("live wire lost identity/opt-in: %s", f.Payload)
				}
				stream := liveStream{c, f.StreamID}
				n := follows.Add(1)
				latestMu.Lock()
				latest = stream
				latestMu.Unlock()
				if n == 1 {
					send(stream, `{"type":"snapshot","header":{"id":"child"},"cursor":20,"records":[{"type":"event","event":{"type":"user/message","seq":15,"data":{"content":[{"type":"text","text":"original"}]}}}],"hasMore":true,"assistantStream":{"revision":10,"activeAttempt":{"attemptId":"a","turn":2,"step":0,"startedAfterSeq":20,"nextIndex":1,"stream":[{"type":"text-chunks","time0":1,"index":0,"dt":[],"texts":["pre"]}]}}}`)
				} else {
					send(stream, `{"type":"snapshot","header":{"id":"child"},"cursor":30,"records":[{"type":"event","event":{"type":"user/message","seq":25,"data":{"content":[{"type":"text","text":"baseline"}]}}}],"hasMore":true,"projections":{"values":{"subagentCatalog":[]}},"assistantStream":{"revision":40,"activeAttempt":{"attemptId":"fresh","turn":3,"step":0,"startedAfterSeq":30,"nextIndex":1,"stream":[{"type":"text-chunks","time0":1,"index":0,"dt":[],"texts":["fresh"]}]}}}`)
				}
				opened <- stream
			}
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/projections"):
			req := args["request"].(map[string]any)
			rows := []any{}
			if req["sessionId"] == "root" {
				rows = []any{map[string]any{"id": "child", "mode": "continuable", "label": "Child"}}
			}
			rpcOK(w, body["rpcId"], map[string]any{"values": map[string]any{"subagentCatalog": rows}})
		case strings.HasSuffix(r.URL.Path, "/session/page"):
			req := args["request"].(map[string]any)
			address := req["address"].(map[string]any)
			if address["kind"] != "subagent" || address["parentSessionId"] != "root" || address["childSessionId"] != "child" || address["mode"] != "continuable" || req["throughSeq"] != float64(31) || req["beforeSeq"] != float64(25) {
				t.Errorf("page lost fixed live cut/address: %#v", req)
			}
			pageStarted <- struct{}{}
			select {
			case <-releasePage:
			case <-r.Context().Done():
				return
			}
			rpcOK(w, body["rpcId"], map[string]any{"records": []any{map[string]any{"type": "event", "event": map[string]any{"type": "user/message", "seq": 5, "data": map[string]any{"content": []any{map[string]any{"type": "text", "text": "older page"}}}}}}, "hasMore": false})
		case strings.HasSuffix(r.URL.Path, "/subagents/prompt"):
			req := args["request"].(map[string]any)
			if req["mode"] != "continuable" || req["delivery"] != "queue" {
				t.Errorf("invalid prompt: %#v", req)
			}
			latestMu.Lock()
			stream := latest
			latestMu.Unlock()
			send(stream, `{"type":"event","event":{"type":"user/message","seq":31,"data":{"content":[{"type":"text","text":"live submitted message"}]}}}`)
			rpcOK(w, body["rpcId"], map[string]any{"messageId": "accepted-live"})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.Error(w, "unexpected", 500)
		}
	}))
	defer srv.Close()
	defer close(releasePage)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, func() { notifies.Add(1) })
	s.AttachMux(mux)
	s.mu.Lock()
	s.selectedID = "root"
	s.items = []Item{{SessionID: "root", AgentAvailable: true}}
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	binding := s.planHistoryBinding
	s.messages = []Message{{ID: "parent", Body: "unchanged"}}
	s.historyThroughSeq = 99
	s.mu.Unlock()
	wait := func(check func(AgentNavigationView) bool) AgentNavigationView {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := s.AgentNavigationSnapshot()
			if check(v) {
				return v
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("live navigation did not settle: %+v", s.AgentNavigationSnapshot())
		return AgentNavigationView{}
	}
	nextOpen := func() liveStream {
		t.Helper()
		select {
		case stream := <-opened:
			return stream
		case <-time.After(3 * time.Second):
			t.Fatal("follow did not open")
			return liveStream{}
		}
	}
	s.SyncAgentNavigation(binding)
	root := wait(func(v AgentNavigationView) bool { return !v.Busy && len(v.Rows) == 1 })
	if follows.Load() != 0 {
		t.Fatal("catalog prefetched child follow")
	}
	s.OpenAgentNavigation(root.Token, root.Rows[0].Address)
	first := nextOpen()
	child := wait(func(v AgentNavigationView) bool { return !v.Busy && v.Process.Live && strings.Contains(v.Body, "pre") })
	send(first, `{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a","revision":11,"index":1,"time":2,"chunk":{"type":"text-delta","index":0,"text":"view"}}}`)
	wait(func(v AgentNavigationView) bool { return strings.Contains(v.Body, "preview") && v.Process.Cursor == 20 })
	send(first, `{"type":"event","event":{"type":"assistant/message","seq":21,"time":3,"surfaceOp":"append","data":{"turn":2,"step":0,"message":{"id":"settled","content":[{"type":"text","text":"canonical"}]}}}}`)
	wait(func(v AgentNavigationView) bool { return v.Process.Cursor == 21 && strings.Contains(v.Body, "preview") })
	send(first, `{"type":"assistant-stream","frame":{"type":"end","attemptId":"a","revision":12,"index":2,"outcome":{"kind":"committed","eventType":"assistant/message","seq":21}}}`)
	wait(func(v AgentNavigationView) bool {
		return strings.Contains(v.Body, "canonical") && !strings.Contains(v.Body, "preview")
	})
	send(first, `{"type":"assistant-stream","frame":{"type":"start","attemptId":"bad","revision":14,"startedAfterSeq":21,"turn":2,"step":1}}`)
	second := nextOpen()
	child = wait(func(v AgentNavigationView) bool {
		return v.Process.Live && !v.Busy && v.Process.Cursor == 30 && strings.Contains(v.Body, "fresh")
	})
	if follows.Load() != 2 || strings.Contains(child.Body, "canonical") {
		t.Fatal("rebaseline retained obsolete prefix")
	}
	beforeNotify := notifies.Load()
	for i := 0; i < 20; i++ {
		send(second, fmt.Sprintf(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"fresh","revision":%d,"index":%d,"time":2,"chunk":{"type":"text-delta","index":0,"text":"x"}}}`, 41+i, 1+i))
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		body := s.agentNavigation.view.Body
		s.mu.Unlock()
		if strings.Contains(body, "fresh"+strings.Repeat("x", 20)) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if delta := notifies.Load() - beforeNotify; delta > 2 {
		t.Fatalf("live frames scheduled %d undrained repaints", delta)
	}
	child = wait(func(v AgentNavigationView) bool { return strings.Contains(v.Body, "fresh"+strings.Repeat("x", 20)) })
	s.PromptAgentNavigation(child.Token, child.Address, "explicit user send")
	child = wait(func(v AgentNavigationView) bool {
		return !v.ActionBusy && strings.Contains(v.Body, "live submitted message") && v.Process.Cursor == 31
	})
	if follows.Load() != 2 {
		t.Fatal("sending reopened/duplicated follow")
	}
	s.EarlierAgentNavigation(child.Token)
	select {
	case <-pageStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("page did not start")
	}
	if got := s.AgentNavigationSnapshot(); got.Process.Live || strings.Contains(got.Body, storelocale.T(storelocale.KeyStoreSessionLivePreview)) {
		t.Fatal("older-page transition kept live preview")
	}
	send(second, `{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"fresh","revision":61,"index":21,"time":2,"chunk":{"type":"text-delta","index":0,"text":"LATE"}}}`)
	releasePage <- struct{}{}
	old := wait(func(v AgentNavigationView) bool { return !v.Busy && strings.Contains(v.Body, "older page") })
	if old.Process.Live || old.Process.Cursor != 31 || strings.Contains(old.Body, "LATE") {
		t.Fatal("late child output corrupted historical cut")
	}
	s.RefreshAgentNavigation(old.Token)
	third := nextOpen()
	_ = third
	live := wait(func(v AgentNavigationView) bool { return !v.Busy && v.Process.Live })
	s.BackAgentNavigation(live.Token)
	if v := s.AgentNavigationSnapshot(); v.SessionID != "root" || v.Process.Live {
		t.Fatal("Back retained active child follow")
	}
	if follows.Load() != 3 {
		t.Fatal("Back auto-followed cached parent")
	}
	root = s.AgentNavigationSnapshot()
	s.OpenAgentNavigation(root.Token, root.Rows[0].Address)
	_ = nextOpen()
	wait(func(v AgentNavigationView) bool { return v.Process.Live && !v.Busy })
	s.CloseAgentNavigation()
	s.SyncAgentNavigation(binding)
	wait(func(v AgentNavigationView) bool { return !v.Busy && len(v.Rows) == 1 })
	if follows.Load() != 4 {
		t.Fatal("pane recovery auto-followed child")
	}
	root = s.AgentNavigationSnapshot()
	s.OpenAgentNavigation(root.Token, root.Rows[0].Address)
	_ = nextOpen()
	wait(func(v AgentNavigationView) bool { return v.Process.Live && !v.Busy })
	s.Attach(&dsh.Client{Endpoint: client.Endpoint}, nil)
	if v := s.AgentNavigationSnapshot(); v.SessionID != "" || v.Body != "" {
		t.Fatal("new Host retained old live child")
	}
	if s.Snapshot().SelectedID != "root" || s.historyThroughSeq != 99 || s.Snapshot().Messages[0].Body != "unchanged" {
		t.Fatal("live navigator mutated main chat")
	}
	if len(canceled) < 3 {
		t.Fatal("rebaseline/page/back did not cancel logical streams")
	}
}
