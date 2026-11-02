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

func TestSidebarCatalogAuthorityWatermarksAndUnknown(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.eventsGeneration = 7
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := s.beginSidebarCatalogWatchLocked(ctx, s.client, nil)
	s.items = []Item{{SessionID: "a", Running: true}, {SessionID: "b"}, {SessionID: "lineage-only", ParentID: "p", Running: true}}
	apply := func(raw string) { s.applySidebarCatalogControlLocked(owner, json.RawMessage(raw)) }
	apply(`{"type":"baseline","value":{"projections":{"p":{"asOfSeq":3,"values":{"subagentCatalog":[{"id":"a","label":"metadata not retained"},{"id":"b"}]}},"empty":{"asOfSeq":2,"values":{"subagentCatalog":[]}},"absent":{"asOfSeq":2,"values":{}}}}}`)
	status := s.sidebarSubagentActivityLocked()
	if status["p"] != (SidebarSubagentActivity{Count: 1, Known: true}) || status["empty"] != (SidebarSubagentActivity{Known: true}) || status["absent"].Known {
		t.Fatal(status)
	}
	if len(s.sidebarCatalog.ByParent["p"].IDs) != 2 {
		t.Fatal("lineage inferred a child")
	}
	apply(`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":4,"value":[{"id":"a"},{"id":"missing"}]}`)
	if got := s.sidebarSubagentActivityLocked()["p"]; got.Known || got.Count != 1 {
		t.Fatal("unknown child fabricated exact count", got)
	}
	for _, raw := range []string{
		`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":3,"value":[]}`,
		`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":4,"value":[]}`,
		`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":5,"value":null}`,
		`{"type":"projection","sessionId":"p","key":"subagentCatalog","value":[]}`,
		`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":5.5,"value":[]}`,
		`{"type":"baseline","value":{"projections":null}}`,
		`{"type":"baseline","value":{"projections":{"p":{"asOfSeq":6,"values":{"subagentCatalog":null}}}}}`,
	} {
		apply(raw)
		if s.sidebarCatalog.ByParent["p"].Seq != 4 || len(s.sidebarCatalog.ByParent["p"].IDs) != 2 {
			t.Fatal("old/malformed frame mutated good cut", raw)
		}
	}
	apply(`{"type":"baseline","value":{"projections":{"p":{"asOfSeq":4,"values":{}}}}}`)
	if len(s.sidebarCatalog.ByParent) != 1 || s.sidebarSubagentActivityLocked()["p"].Known {
		t.Fatal("baseline omission retained stale authority")
	}
	apply(`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":5,"value":[]}`)
	if !s.sidebarSubagentActivityLocked()["p"].Known {
		t.Fatal("explicit empty was not known zero")
	}
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/deleted", sessionID: "p"}, true)
	if _, ok := s.sidebarCatalog.ByParent["p"]; ok {
		t.Fatal("deleted parent retained status cache")
	}
	cancel()
	if s.sidebarSubagentActivityLocked() != nil {
		t.Fatal("cancelled control owner retained authority")
	}
}

func TestSidebarCatalogStreamEpochHostAndPaneGuards(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.eventsGeneration = 3
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := s.beginSidebarCatalogWatchLocked(ctx, s.client, nil)
	raw := json.RawMessage(`{"type":"baseline","value":{"projections":{"p":{"asOfSeq":1,"values":{"subagentCatalog":[]}}}}}`)
	s.applySidebarCatalogControlLocked(first, raw)
	second := s.beginSidebarCatalogWatchLocked(ctx, s.client, nil)
	s.applySidebarCatalogControlLocked(first, raw)
	if s.sidebarCatalog.Baseline {
		t.Fatal("old stream established new baseline")
	}
	s.applySidebarCatalogControlLocked(second, raw)
	s.finishSidebarCatalogWatch(first)
	if !s.sidebarSubagentActivityLocked()["p"].Known {
		t.Fatal("old cleanup cleared newer stream")
	}
	s.agentNavigationOwner = New()
	if s.sidebarSubagentActivityLocked() != nil {
		t.Fatal("child pane acquired main status authority")
	}
	s.agentNavigationOwner = nil
	s.BeginEvents(context.Background(), s.client)
	if s.sidebarCatalog.ByParent != nil || s.Snapshot().SidebarSubagentActivity != nil {
		t.Fatal("events generation retained old authority")
	}
	s.applySidebarCatalogControlLocked(second, raw)
	if s.sidebarCatalog.Baseline {
		t.Fatal("pre-reconnect control owner restored authority")
	}
	s.Attach(&dsh.Client{}, nil)
	if s.sidebarCatalog.ByParent != nil {
		t.Fatal("replacement Host retained child IDs")
	}
}

// Run through the existing mux receiver, not a direct helper call. The fixture
// opens exactly session/control and never issues an RPC or additional stream.
func TestSidebarCatalogExistingControlRouteSnapshotAndCleanup(t *testing.T) {
	type carrier struct {
		conn *websocket.Conn
		id   string
	}
	opened := make(chan carrier, 2)
	var opens, unexpected atomic.Int32
	baseline := json.RawMessage(`{"type":"baseline","value":{"projections":{"p":{"asOfSeq":3,"values":{"subagentCatalog":[{"id":"a"}]}},"empty":{"asOfSeq":2,"values":{"subagentCatalog":[]}}}}}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			unexpected.Add(1)
			http.Error(w, "unexpected RPC", 500)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			if frame.Type == dsh.FrameCancel {
				continue
			}
			if frame.Type != dsh.FrameOpen || frame.Endpoint != "session/control" {
				unexpected.Add(1)
				return
			}
			opens.Add(1)
			if err := wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: baseline}); err != nil {
				return
			}
			opened <- carrier{conn, frame.StreamID}
		}
	}))
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	notifications := make(chan struct{}, 32)
	s.Attach(client, func() {
		select {
		case notifications <- struct{}{}:
		default:
		}
	})
	s.AttachMux(mux)
	s.mu.Lock()
	s.selectedID = "main"
	s.items = []Item{{SessionID: "main"}, {SessionID: "a", Running: true}, {SessionID: "fake-lineage", ParentID: "p", Running: true}}
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.WatchControl(ctx)
	wait := func(test func(ViewState) bool) ViewState {
		t.Helper()
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		for {
			v := s.Snapshot()
			if test(v) {
				return v
			}
			select {
			case <-notifications:
			case <-timer.C:
				t.Fatalf("control condition not reached: %+v", v.SidebarSubagentActivity)
			}
		}
	}
	v := wait(func(v ViewState) bool {
		return v.SidebarSubagentActivity["p"] == (SidebarSubagentActivity{Count: 1, Known: true})
	})
	if v.SelectedID != "main" || s.AgentNavigationSnapshot().Token != 0 {
		t.Fatal("status read activated navigation")
	}
	v.SidebarSubagentActivity["p"] = SidebarSubagentActivity{Count: 99, Known: true}
	if s.Snapshot().SidebarSubagentActivity["p"].Count != 1 {
		t.Fatal("Snapshot map aliases cached state")
	}
	s.mu.Lock()
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/status", sessionID: "a", running: false}, true)
	s.mu.Unlock()
	s.fire()
	wait(func(v ViewState) bool {
		return v.SidebarSubagentActivity["p"] == (SidebarSubagentActivity{Known: true})
	})
	var stream carrier
	select {
	case stream = <-opened:
	case <-time.After(3 * time.Second):
		t.Fatal("existing control stream never opened")
	}
	if err := wsjson.Write(ctx, stream.conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: stream.id, Value: json.RawMessage(`{"type":"projection","sessionId":"p","key":"subagentCatalog","seq":4,"value":[{"id":"missing"}]}`)}); err != nil {
		t.Fatal(err)
	}
	wait(func(v ViewState) bool { status, ok := v.SidebarSubagentActivity["p"]; return ok && !status.Known })
	cancel()
	wait(func(v ViewState) bool { return v.SidebarSubagentActivity == nil })
	if opens.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("status wiring made extra requests: streams=%d unexpected=%d", opens.Load(), unexpected.Load())
	}
}
