package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
)

func TestParityChildQueueProjectionAndConsumedEditor(t *testing.T) {
	v := AgentNavigationView{QueueEditor: QueueEditorState{ItemID: "q", Token: 9, Ready: true}}
	seedChildQueue(&v, json.RawMessage(`{"inbox":{"next-turn":[{"id":"q","content":[{"type":"text","text":"full authored content"}],"source":{"kind":"user"}}]}}`))
	if len(v.Board.Queue) != 1 || !v.Board.Queue[0].TextOnly || v.Board.Queue[0].Preview != "full authored content" || v.QueueEditor.ItemID != "q" {
		t.Fatalf("%+v", v)
	}
	childQueueFrame(&v, json.RawMessage(`{"type":"projection","key":"inbox","value":{"next-turn":[]}}`))
	if len(v.Board.Queue) != 0 || v.QueueEditor.ItemID != "" || v.QueueEditor.Token != 10 {
		t.Fatal("consumed occurrence retained editor")
	}
	var start projectflow.ProcessRecord
	start.Event.Type = "turn/start"
	start.Event.Seq = 10
	v.Process.Records = []projectflow.ProcessRecord{start}
	v.Messages = []Message{{EventType: "turn/end", Seq: 9}}
	childQueueFrame(&v, json.RawMessage(`{"type":"event"}`))
	if !v.Running {
		t.Fatal("older presented turn/end hid the active non-transcript turn/start")
	}
}
func TestParityChildGestureReceiptAndQueueWire(t *testing.T) {
	calls := make(chan map[string]any, 8)
	wake := make(chan struct{}, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		args := b["payload"].(map[string]any)["args"].(map[string]any)
		request := args["request"].(map[string]any)
		request["method"] = r.URL.Path
		calls <- request
		if strings.HasSuffix(r.URL.Path, "/subagents/prompt") {
			rpcOK(w, b["rpcId"], map[string]any{"messageId": "accepted"})
		} else {
			rpcOK(w, b["rpcId"], map[string]any{"accepted": true})
		}
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	s := New()
	s.Attach(client, func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	})
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	s.mu.Lock()
	s.selectedID = "root"
	s.items = []Item{{SessionID: "root", AgentAvailable: true}}
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	s.agentNavigation = agentNavigationState{client: client, mux: s.mux, generation: s.followGeneration, binding: s.planHistoryBinding, root: "root", view: AgentNavigationView{Token: 7, SessionID: "child", Address: address, Running: true, Process: projectflow.ProcessSnapshot{Live: true}, Board: PlanBoard{Queue: []PlanQueueItem{{ID: "q1"}, {ID: "q2"}}}}}
	s.mu.Unlock()
	wait := func(predicate func(AgentNavigationView) bool) {
		t.Helper()
		timeout := time.After(2 * time.Second)
		for !predicate(s.AgentNavigationSnapshot()) {
			select {
			case <-wake:
			case <-timeout:
				t.Fatal("child action did not settle")
			}
		}
	}
	s.PromptAgentNavigationGesture(7, address, "authored", true, "steer")
	req := <-calls
	if req["delivery"] != "queue" || req["parentSessionId"] != "root" || req["childSessionId"] != "child" {
		t.Fatalf("gesture inversion/identity lost: %+v", req)
	}
	wait(func(v AgentNavigationView) bool { return !v.ActionBusy && v.PromptReceipt == 1 })
	s.PromptAgentNavigationGesture(7, address, "", true, "queue")
	first, second := <-calls, <-calls
	if !strings.HasSuffix(first["method"].(string), "/session/updateQueue") || first["sessionId"] != "child" || first["action"].(map[string]any)["kind"] != "steer" || second["itemId"] == first["itemId"] {
		t.Fatalf("empty Ctrl Enter did not steer captured inbox: %+v %+v", first, second)
	}
	wait(func(v AgentNavigationView) bool { return !v.ActionBusy && len(v.Board.Queue) == 0 })
	s.UpdateAgentQueue(6, address, "q1", QueueActionRemove)
	select {
	case req := <-calls:
		t.Fatalf("stale child token sent %+v", req)
	default:
	}
}

func TestParityChildQueueConsumesSharedHostControlProjection(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	defer root.ReleaseAgentNavigationOccurrence(pane)
	root.mu.Lock()
	root.applyQuestionFaceLocked(address.ChildSessionID, "inbox", json.RawMessage(`{"next-turn":[{"id":"queued","content":[{"type":"text","text":"actual child pending text"}],"source":{"kind":"user"}}],"next-step":[]}`), 31)
	root.mu.Unlock()
	v := pane.AgentNavigationSnapshot()
	if len(v.Board.Queue) != 1 || v.Board.Queue[0].ID != "queued" || v.Board.Queue[0].Preview != "actual child pending text" || len(root.Snapshot().PlanBoard.Queue) != 0 {
		t.Fatalf("child control inbox was absent or leaked into parent: %+v", v.Board.Queue)
	}
	pane.mu.Lock()
	pane.agentNavigation.view.QueueEditor = QueueEditorState{ItemID: "queued", Ready: true, Token: 7}
	stale := float64(30)
	seedChildQueue(&pane.agentNavigation.view, json.RawMessage(`{"inbox":{"next-turn":[]}}`), &stale)
	pane.mu.Unlock()
	if len(pane.AgentNavigationSnapshot().Board.Queue) != 1 {
		t.Fatal("older follow baseline overwrote current child inbox")
	}
	root.mu.Lock()
	root.applyQuestionFaceLocked(address.ChildSessionID, "inbox", json.RawMessage(`{"next-turn":[],"next-step":[]}`), 32)
	root.mu.Unlock()
	v = pane.AgentNavigationSnapshot()
	if len(v.Board.Queue) != 0 || v.QueueEditor.ItemID != "" || v.QueueEditor.Token != 8 {
		t.Fatal("consumed authoritative child inbox retained row/editor")
	}
}

func TestParityNestedConversationRoutesWithoutEntering(t *testing.T) {
	s, a, leaf := nestedCatalogRoutingFixture()
	sibling := leaf
	sibling.ChildSessionID = "sibling"
	branch := s.agentNavigation.branches[agentCatalogKey(s.agentNavigation.view.Address, []AgentSessionAddress{a})]
	branch.view.catalog.values = json.RawMessage(`{"subagentCatalog":[{"id":"leaf","mode":"continuable","label":"Leaf"},{"id":"sibling","mode":"continuable","label":"Sibling"}]}`)
	ancestors, siblings := s.AgentConversationRoutes(7, leaf)
	if len(ancestors) != 1 || ancestors[0].Address != a || len(siblings) != 2 {
		t.Fatalf("nested header routing absent: ancestors=%+v siblings=%+v", ancestors, siblings)
	}
	path := s.AgentChatRestoreLineage(7, sibling)
	if len(path) != 2 || path[0] != a || path[1] != sibling {
		t.Fatalf("sibling did not retain verified full path: %+v", path)
	}
	branch.expanded = false
	ancestors, siblings = s.AgentConversationRoutes(7, leaf)
	if len(ancestors) != 0 || len(siblings) != 0 {
		t.Fatal("collapsed parent retained nested controls")
	}
}
