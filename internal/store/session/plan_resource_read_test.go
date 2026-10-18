package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

// One synthetic carrier fixture: the real mux/RPC envelopes, fixed-cut child
// paging, full identity matching, canceled reads and Host replacement. No Host.
func TestAddressedPlanResourceFixture(t *testing.T) {
	target := PlanResourceTarget{ParentSessionID: "parent/one", ChildSessionID: "child/two", Mode: "continuable", CallID: "call/three"}
	expected := map[string]any{"kind": "subagent", "parentSessionId": target.ParentSessionID, "childSessionId": target.ChildSessionID, "mode": target.Mode}
	canceled := make(chan struct{}, 8)
	pageStarted := make(chan struct{}, 8)
	var mu sync.Mutex
	hold := false
	release := make(chan struct{})
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
					canceled <- struct{}{}
					continue
				}
				if f.Type != dsh.FrameOpen || f.Endpoint != "session/follow" {
					t.Errorf("unexpected stream %+v", f)
					return
				}
				var p struct {
					Args struct {
						Request map[string]any `json:"request"`
					} `json:"args"`
				}
				_ = json.Unmarshal(f.Payload, &p)
				if !reflect.DeepEqual(p.Args.Request["address"], expected) || len(p.Args.Request) != 1 {
					t.Errorf("follow addressed envelope: %s", f.Payload)
				}
				snap := map[string]any{"type": "snapshot", "header": map[string]any{"id": target.ChildSessionID}, "cursor": 50, "hasMore": true, "records": []any{map[string]any{"type": "event", "event": map[string]any{"seq": 30, "type": "turn/start", "data": map[string]any{}}}}}
				raw, _ := json.Marshal(snap)
				if wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: f.StreamID, Value: raw}) != nil {
					return
				}
			}
		}
		if !strings.HasSuffix(r.URL.Path, "/session/page") {
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.Error(w, "unexpected", 500)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		payload, _ := body["payload"].(map[string]any)
		args, _ := payload["args"].(map[string]any)
		req, _ := args["request"].(map[string]any)
		if !reflect.DeepEqual(req["address"], expected) || req["throughSeq"] != float64(50) || req["beforeSeq"] != float64(30) {
			t.Errorf("page addressed envelope: %#v", req)
		}
		select {
		case <-canceled:
		case <-time.After(2 * time.Second):
			t.Error("snapshot follow was not closed before page")
		}
		pageStarted <- struct{}{}
		mu.Lock()
		blocked := hold
		mu.Unlock()
		if blocked {
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		arguments, _ := json.Marshal(map[string]any{"plan": "# Exact child plan\n\nComplete body\n"})
		rpcOK(w, body["rpcId"], map[string]any{"hasMore": false, "records": []any{map[string]any{"type": "event", "event": map[string]any{"type": "tool/call", "seq": 10, "data": map[string]any{"name": "exit_plan_mode", "callId": target.CallID, "arguments": string(arguments)}}}}})
	}))
	defer srv.Close()
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	s.mu.Lock()
	s.selectedID = target.ParentSessionID
	s.messages = []Message{{ID: "parent", Body: "unchanged"}}
	s.historyThroughSeq = 99
	s.historyHasMore = true
	s.mu.Unlock()
	binding := fmt.Sprintf("%p\x00%s", client, target.ParentSessionID)
	s.mu.Lock()
	s.planHistoryBinding = binding
	s.mu.Unlock()
	before := s.Snapshot()
	read := s.ReadSubagentPlan(ctx, binding, target, nil)
	defer read.Cancel()
	select {
	case <-read.done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	result := read.Snapshot()
	if result.Loading || result.Error != "" || result.Plan.CallID != target.CallID || result.Plan.Markdown != "# Exact child plan\n\nComplete body\n" {
		t.Fatalf("read result: %+v", result)
	}
	if !reflect.DeepEqual(before, s.Snapshot()) || s.historyThroughSeq != 99 {
		t.Fatal("resource read mutated active parent history")
	}
	if !read.Matches(binding, target) {
		t.Fatal("exact read identity missing")
	}
	for _, changed := range []PlanResourceTarget{{ParentSessionID: "other", ChildSessionID: target.ChildSessionID, Mode: target.Mode, CallID: target.CallID}, {ParentSessionID: target.ParentSessionID, ChildSessionID: "other", Mode: target.Mode, CallID: target.CallID}, {ParentSessionID: target.ParentSessionID, ChildSessionID: target.ChildSessionID, Mode: "one-shot", CallID: target.CallID}, {ParentSessionID: target.ParentSessionID, ChildSessionID: target.ChildSessionID, Mode: target.Mode, CallID: "other"}} {
		if read.Matches(binding, changed) {
			t.Fatal("distinct full address aliased")
		}
	}
	if read.Matches("other-host\x00"+target.ParentSessionID, target) {
		t.Fatal("different Host aliased")
	}
	<-pageStarted
	mu.Lock()
	hold = true
	mu.Unlock()
	late := s.ReadSubagentPlan(ctx, binding, target, nil)
	select {
	case <-pageStarted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	late.Cancel()
	select {
	case <-late.done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if got := late.Snapshot(); got.Plan.Markdown != "" || got.Loading {
		t.Fatalf("closed read retained body: %+v", got)
	}
	canceledContext, stop := context.WithCancel(ctx)
	stop()
	stopped := s.ReadSubagentPlan(canceledContext, binding, target, nil)
	select {
	case <-stopped.done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if got := stopped.Snapshot(); got.Loading || got.Error == "" || got.Plan.Markdown != "" {
		t.Fatalf("canceled caller left a loading reader: %+v", got)
	}
	replacement := &dsh.Client{Endpoint: client.Endpoint}
	s.Attach(replacement, nil)
	if got := read.Snapshot(); got.Plan.Markdown != "" {
		t.Fatal("same parent/child/call exposed previous Host body")
	}
	pendingBinding := fmt.Sprintf("%p\x00%s", replacement, target.ParentSessionID)
	pending := s.ReadSubagentPlan(ctx, pendingBinding, target, nil)
	if got := pending.Snapshot(); got.Loading || got.Error == "" || got.Plan.Markdown != "" {
		t.Fatalf("new client admitted a read before current follow/mux history: %+v", got)
	}
	invalid := target
	invalid.Mode = "unsupported"
	rejected := s.ReadSubagentPlan(ctx, fmt.Sprintf("%p\x00%s", replacement, target.ParentSessionID), invalid, nil)
	if got := rejected.Snapshot(); got.Error == "" || got.Loading {
		t.Fatalf("unsupported address was not explicit: %+v", got)
	}
}
