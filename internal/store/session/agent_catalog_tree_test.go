package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestAgentCatalogTreeExplicitBranches(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if r.URL.Path != "/rpc/session/projections" && !strings.HasSuffix(r.URL.Path, "/session/projections") {
			t.Errorf("unexpected activating RPC %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": b["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"asOfSeq": 1, "values": map[string]any{"subagentCatalog": []any{map[string]any{"id": "leaf", "mode": "continuable"}}}}}})
	}))
	defer srv.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{}
	s.client, s.mux, s.selectedID = client, mux, "root"
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	a := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "a", Mode: "continuable"}
	b := a
	b.ChildSessionID = "b"
	s.agentNavigation = agentNavigationState{binding: s.planHistoryBinding, root: "root", client: client, mux: mux, view: AgentNavigationView{Token: 7, SessionID: "root", Address: AgentSessionAddress{Kind: "session", SessionID: "root"}, Rows: []AgentNavigationRow{{Address: a}, {Address: b}}}}
	if len(s.AgentCatalogTree(7)) != 2 || calls.Load() != 0 {
		t.Fatal("tree prefetched")
	}
	s.ToggleAgentCatalogBranch(7, []AgentSessionAddress{a})
	s.ToggleAgentCatalogBranch(7, []AgentSessionAddress{b})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows := s.AgentCatalogTree(7)
		if len(rows) == 4 && !rows[0].Busy && !rows[2].Busy {
			break
		}
		time.Sleep(time.Millisecond)
	}
	rows := s.AgentCatalogTree(7)
	if len(rows) != 4 || rows[1].Depth != 1 || rows[3].Depth != 1 || calls.Load() != 2 {
		t.Fatalf("multi-branch %+v calls %d", rows, calls.Load())
	}
	s.ToggleAgentCatalogBranch(7, []AgentSessionAddress{a})
	if len(s.AgentCatalogTree(7)) != 3 {
		t.Fatal("collapse hid sibling")
	}
	s.ToggleAgentCatalogBranch(6, []AgentSessionAddress{b})
	if len(s.AgentCatalogTree(7)) != 3 {
		t.Fatal("stale token changed tree")
	}
	s.mu.Lock()
	s.agentNavigation.view.Rows = s.agentNavigation.view.Rows[:1]
	s.mu.Unlock()
	if len(s.AgentCatalogTree(7)) != 1 {
		t.Fatal("removed branch survived")
	}
	s.CloseAgentNavigation()
	if len(s.AgentCatalogTree(7)) != 0 {
		t.Fatal("close retained authority")
	}
}
