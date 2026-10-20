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
	"whaleisle/internal/store/storelocale"
)

// Separate new fence fixture; the earlier navigation fixture is not rerun.
func TestNativeAgentActionScopeFixture(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var oldCalls, newCalls atomic.Int32
	oldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldCalls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		req := args["request"].(map[string]any)
		if !strings.HasSuffix(r.URL.Path, "/subagents/prompt") || req["parentSessionId"] != "root" || req["childSessionId"] != "child" {
			t.Errorf("wrong old Host/action: %s %#v", r.URL.Path, req)
		}
		started <- struct{}{}
		<-release
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer oldServer.Close()
	defer close(release)
	newServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newCalls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		if !strings.HasSuffix(r.URL.Path, "/subagents/interruptByParent") || args["parentSessionId"] != "root" || args["childSessionId"] != "child" {
			t.Errorf("wrong new Host/action: %s %#v", r.URL.Path, args)
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer newServer.Close()
	oldClient := &dsh.Client{Endpoint: dsh.Endpoint{Origin: oldServer.URL}}
	newClient := &dsh.Client{Endpoint: dsh.Endpoint{Origin: newServer.URL}}
	s := New()
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	bind := func(client *dsh.Client, token uint64) {
		s.Attach(client, nil)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.selectedID = "root"
		s.items = []Item{{SessionID: "root", AgentAvailable: true}}
		s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
		s.agentNavigation = agentNavigationState{client: client, mux: s.mux, generation: s.followGeneration, binding: s.planHistoryBinding, root: "root", view: AgentNavigationView{Token: token, SessionID: "child", Address: address, ActionStatus: "current marker"}}
	}
	bind(oldClient, 10)
	wrong := address
	wrong.Mode = "unknown"
	s.PromptAgentNavigation(10, wrong, "wrong mode")
	s.PromptAgentNavigation(9, address, "stale token")
	if oldCalls.Load() != 0 {
		t.Fatal("stale/mismatched action admitted")
	}
	s.PromptAgentNavigation(10, address, "approved child text")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("captured old Host prompt did not dispatch")
	}
	bind(newClient, 20)
	s.PromptAgentNavigation(10, address, "stale old view")
	s.InterruptAgentNavigation(10, []AgentSessionAddress{address})
	if newCalls.Load() != 0 {
		t.Fatal("stale old view reached new Host")
	}
	s.InterruptAgentNavigation(20, []AgentSessionAddress{address})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		v := s.AgentNavigationSnapshot()
		if !v.ActionBusy && strings.Contains(v.ActionStatus, storelocale.T(storelocale.KeyStoreSessionInterruptRequested)) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	v := s.AgentNavigationSnapshot()
	if newCalls.Load() != 1 || v.ActionBusy || !strings.Contains(v.ActionStatus, storelocale.T(storelocale.KeyStoreSessionInterruptRequested)) {
		t.Fatalf("current action did not settle: %+v calls=%d", v, newCalls.Load())
	}
	if s.Snapshot().LastAction != "" {
		t.Fatal("navigation action wrote main chat feedback")
	}
	s.CloseAgentNavigation()
	s.InterruptAgentNavigation(20, []AgentSessionAddress{address})
	if newCalls.Load() != 1 {
		t.Fatal("closed navigation dispatched")
	}
}
