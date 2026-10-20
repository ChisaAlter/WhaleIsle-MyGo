package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

func TestAgentNavigationImagesWireFixture(t *testing.T) {
	requests := make(chan api.SubagentPromptRequest, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request api.SubagentPromptRequest `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/subagents/prompt") {
			t.Errorf("unexpected RPC %s", r.URL.Path)
		}
		requests <- body.Payload.Args.Request
		if len(body.Payload.Args.Request.Content) == 0 {
			rpcOK(w, body.RPCID, map[string]any{})
			return
		}
		rpcOK(w, body.RPCID, map[string]any{"messageId": "accepted-message"})
	}))
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s := New()
	s.Attach(client, nil)
	target := AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent/exact", ChildSessionID: "child/exact", Mode: "continuable"}
	s.mu.Lock()
	s.selectedID = target.ParentSessionID
	s.items = []Item{{SessionID: target.ParentSessionID, AgentAvailable: true}}
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, s.selectedID)
	s.agentNavigation = agentNavigationState{client: client, mux: s.mux, generation: s.followGeneration, binding: s.planHistoryBinding, root: s.selectedID, view: AgentNavigationView{Token: 12, SessionID: target.ChildSessionID, Address: target}}
	s.mu.Unlock()
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, []byte("\x89PNG\r\n\x1a\nimage"), 0600); err != nil {
		t.Fatal(err)
	}
	previousID := ""
	for _, delivery := range []PromptMode{PromptModeQueue, PromptModeSteer} {
		s.SetPromptMode(delivery)
		s.AddAgentNavigationImages(11, target, []string{path}, nil)
		if len(s.AgentNavigationSnapshot().PromptImages) != 0 {
			t.Fatal("stale picker admitted")
		}
		s.AddAgentNavigationImages(12, target, []string{path}, nil)
		s.PromptAgentNavigation(12, target, "")
		select {
		case req := <-requests:
			if req.Mode != "continuable" || req.Delivery != string(delivery) || req.ParentSessionId != target.ParentSessionID || req.ChildSessionId != target.ChildSessionID || req.RequestId == "" || req.RequestId == previousID || len(req.Content) != 1 || !strings.Contains(string(req.Content[0]), `"type":"image"`) || !strings.Contains(string(req.Content[0]), `"data":`) {
				t.Fatalf("incorrect child request: %+v", req)
			}
			previousID = req.RequestId
		case <-time.After(2 * time.Second):
			t.Fatal("prompt not dispatched")
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && s.AgentNavigationSnapshot().ActionBusy {
			time.Sleep(time.Millisecond)
		}
		if v := s.AgentNavigationSnapshot(); v.ActionBusy || len(v.PromptImages) != 0 || !strings.Contains(v.ActionStatus, storelocale.T(storelocale.KeyStoreSessionChildPromptSubmitted)) {
			t.Fatalf("ack failed: %+v", v)
		}
	}
	for _, mode := range []string{"one-shot", "unknown"} {
		unsupported := target
		unsupported.Mode = mode
		s.mu.Lock()
		s.agentNavigation.view.Address = unsupported
		s.mu.Unlock()
		s.PromptAgentNavigation(12, unsupported, "must not send")
	}
	select {
	case req := <-requests:
		t.Fatalf("unsupported mode dispatched: %+v", req)
	default:
	}
	if s.Snapshot().LastAction != "" {
		t.Fatal("child prompt changed main feedback")
	}
}
