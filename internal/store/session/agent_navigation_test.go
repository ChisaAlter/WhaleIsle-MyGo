package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// One synthetic fixture: no real Host, model, prompt, tool or subagent launch.
func TestNativeAgentNavigationFixture(t *testing.T) {
	var follows, pages, cancels atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					cancels.Add(1)
					continue
				}
				if f.Type != dsh.FrameOpen || f.Endpoint != "session/follow" {
					t.Errorf("unexpected stream %+v", f)
					return
				}
				follows.Add(1)
				var p struct {
					Args struct {
						Request struct {
							Address     AgentSessionAddress `json:"address"`
							MaxMessages int                 `json:"maxMessages"`
						} `json:"request"`
					} `json:"args"`
				}
				if json.Unmarshal(f.Payload, &p) != nil {
					t.Error("invalid follow")
					return
				}
				a := p.Args.Request.Address
				if a.Kind != "subagent" || a.SessionID != "" || a.ParentSessionID == "" || a.ChildSessionID == "" || (a.Mode != "continuable" && a.Mode != "one-shot" && a.Mode != "unknown") {
					t.Errorf("lost addressed identity: %+v", a)
				}
				if a.ChildSessionID == "child/one" && a != (AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child/one", Mode: "continuable"}) {
					t.Errorf("wrong parent/mode: %+v", a)
				}
				if a.ChildSessionID == "grandchild" && a.ParentSessionID != "child/one" {
					t.Errorf("nested child aliased root: %+v", a)
				}
				if a.ChildSessionID == "held" {
					continue
				}
				raw, _ := json.Marshal(map[string]any{"type": "snapshot", "header": map[string]any{"id": a.ChildSessionID}, "cursor": 50, "hasMore": true, "records": []any{map[string]any{"type": "event", "event": map[string]any{"seq": 30, "type": "message/user", "data": map[string]any{"content": "child history"}}}}})
				if wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: f.StreamID, Value: raw}) != nil {
					return
				}
			}
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		payload, _ := body["payload"].(map[string]any)
		args, _ := payload["args"].(map[string]any)
		req, _ := args["request"].(map[string]any)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/projections"):
			id, _ := req["sessionId"].(string)
			var catalog []any
			switch id {
			case "root":
				catalog = []any{map[string]any{"id": "child/one", "label": "Prepared child", "mode": "continuable"}, map[string]any{"id": "held", "label": "Held child", "mode": "one-shot"}}
			case "child/one":
				catalog = []any{map[string]any{"id": "grandchild", "label": "Nested child", "mode": "unknown"}}
			default:
				catalog = []any{}
			}
			rpcOK(w, body["rpcId"], map[string]any{"values": map[string]any{"subagentCatalog": catalog}})
		case strings.HasSuffix(r.URL.Path, "/session/page"):
			pages.Add(1)
			expected := map[string]any{"kind": "subagent", "parentSessionId": "root", "childSessionId": "child/one", "mode": "continuable"}
			if !reflect.DeepEqual(req["address"], expected) || req["throughSeq"] != float64(50) || req["beforeSeq"] != float64(30) {
				t.Errorf("page lost cut/address: %#v", req)
			}
			rpcOK(w, body["rpcId"], map[string]any{"hasMore": false, "records": []any{map[string]any{"type": "event", "event": map[string]any{"seq": 10, "type": "message/user", "data": map[string]any{"content": "earlier child"}}}}})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.Error(w, "unexpected", 500)
		}
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	binding := fmt.Sprintf("%p\x00root", client)
	s.mu.Lock()
	s.selectedID = "root"
	s.planHistoryBinding = binding
	s.messages = []Message{{ID: "parent", Body: "untouched"}}
	s.historyThroughSeq = 99
	s.items = []Item{{SessionID: "root", Title: "Root"}, {SessionID: "stale-child", ParentID: "root", Title: "Not in catalog"}}
	s.mu.Unlock()
	before := s.Snapshot()
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
		t.Fatal("navigation did not settle")
		return AgentNavigationView{}
	}
	s.SyncAgentNavigation(binding)
	root := wait(func(v AgentNavigationView) bool { return v.SessionID == "root" && !v.Busy })
	if len(root.Rows) != 2 || root.Rows[0].Label != "Prepared child" || follows.Load() != 0 {
		t.Fatalf("catalog must include unmaterialized child without following: %+v follows=%d", root, follows.Load())
	}
	wrong := root.Rows[0].Address
	wrong.Mode = "one-shot"
	s.OpenAgentNavigation(root.Token, wrong)
	if follows.Load() != 0 {
		t.Fatal("unlisted identity was opened")
	}
	s.OpenAgentNavigation(root.Token, root.Rows[0].Address)
	child := wait(func(v AgentNavigationView) bool { return v.SessionID == "child/one" && !v.Busy })
	if child.Error != "" || child.Depth != 1 || len(child.Rows) != 1 || child.Rows[0].Address.ParentSessionID != "child/one" || child.Process.Cursor != 50 {
		t.Fatalf("child navigation: %+v", child)
	}
	s.EarlierAgentNavigation(child.Token)
	earlier := wait(func(v AgentNavigationView) bool { return !v.Busy && v.Token != child.Token })
	if earlier.Error != "" || earlier.Process.Records[0].Event.Seq != 10 || earlier.Process.HasMore || pages.Load() != 1 {
		t.Fatalf("earlier: %+v", earlier)
	}
	s.OpenAgentNavigation(earlier.Token, earlier.Rows[0].Address)
	nested := wait(func(v AgentNavigationView) bool { return v.SessionID == "grandchild" && !v.Busy })
	if nested.Error != "" || nested.Depth != 2 || len(nested.Rows) != 0 {
		t.Fatalf("nested: %+v", nested)
	}
	n := follows.Load()
	s.BackAgentNavigation(nested.Token)
	restored := s.AgentNavigationSnapshot()
	if restored.SessionID != "child/one" || restored.Process.Records[0].Event.Seq != 10 || follows.Load() != n {
		t.Fatal("Back must restore cached parent page without follow")
	}
	s.BackAgentNavigation(restored.Token)
	root = s.AgentNavigationSnapshot()
	s.OpenAgentNavigation(root.Token, root.Rows[1].Address)
	deadline := time.Now().Add(3 * time.Second)
	for follows.Load() != n+1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if follows.Load() != n+1 {
		t.Fatal("held child did not open")
	}
	s.CloseAgentNavigation()
	if got := s.AgentNavigationSnapshot(); got.SessionID != "" || got.Body != "" {
		t.Fatal("closed navigation leaked state")
	}
	if !reflect.DeepEqual(before, s.Snapshot()) || s.historyThroughSeq != 99 {
		t.Fatal("navigation mutated main selection/composer/history")
	}
	s.SyncAgentNavigation(binding)
	root = wait(func(v AgentNavigationView) bool { return v.SessionID == "root" && !v.Busy })
	if follows.Load() != n+1 {
		t.Fatal("reopening pane prefetched history")
	}
	s.OpenAgentNavigation(root.Token, root.Rows[0].Address)
	child = wait(func(v AgentNavigationView) bool { return v.SessionID == "child/one" && !v.Busy })
	replacement := &dsh.Client{Endpoint: client.Endpoint}
	s.Attach(replacement, nil)
	if got := s.AgentNavigationSnapshot(); got.SessionID != "" || got.Body != "" {
		t.Fatal("Host replacement retained child history")
	}
	if cancels.Load() < 2 {
		t.Fatalf("snapshot follow not canceled: %d", cancels.Load())
	}
	// Distinguish absent catalog (lineage fallback) from authoritative empty.
	rows, err := agentNavigationRows("root", json.RawMessage(`{}`), []Item{{SessionID: "x", ParentID: "root"}})
	if err != nil || len(rows) != 1 || rows[0].Address.Mode != "unknown" {
		t.Fatal("lineage fallback lost unknown mode")
	}
	rows, err = agentNavigationRows("root", json.RawMessage(`{"subagentCatalog":[]}`), []Item{{SessionID: "x", ParentID: "root"}})
	if err != nil || len(rows) != 0 {
		t.Fatal("empty catalog resurrected stale child")
	}
}
