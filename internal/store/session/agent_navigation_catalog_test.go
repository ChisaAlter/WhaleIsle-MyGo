package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestAgentLiveCatalogFixture(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	mux := &dsh.Mux{}
	s.client, s.mux, s.selectedID = client, mux, "root"
	s.items = []Item{{SessionID: "root", AgentAvailable: true}}
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, "root")
	s.agentNavigation = agentNavigationState{binding: s.planHistoryBinding, root: "root", client: client, mux: mux, generation: s.followGeneration, view: AgentNavigationView{Token: 7, SessionID: "root", Address: AgentSessionAddress{Kind: "session", SessionID: "root"}}}
	s.items = []Item{{SessionID: "child", ParentID: "root", Title: "Child title", Running: false}, {SessionID: "leaf", ParentID: "child", Title: "Leaf", Running: false}}
	apply := func(raw string) {
		s.mu.Lock()
		s.applyAgentNavigationControlLocked(json.RawMessage(raw))
		s.mu.Unlock()
	}
	apply(`{"type":"baseline","value":{"projections":{"root":{"asOfSeq":10,"values":{"subagentCatalog":[]}}}}}`)
	if rows := s.AgentNavigationSnapshot().Rows; len(rows) != 0 {
		t.Fatal("authoritative empty lost")
	}
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":11,"value":[{"id":"child","mode":"unknown","createdAt":1}]}`)
	rows := s.AgentNavigationSnapshot().Rows
	if len(rows) != 1 || rows[0].Address != (AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "unknown"}) || rows[0].Label != "Child title" || rows[0].Running {
		t.Fatalf("new child: %+v", rows)
	}
	for _, raw := range []string{
		`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":10,"value":[]}`,
		`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":11,"value":[]}`,
		`{"type":"baseline","value":{"projections":{"root":{"asOfSeq":10,"values":{}}}}}`,
		`{"type":"projection","sessionId":"root","key":"subagentCatalog","value":[]}`,
		`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":99,"value":null}`,
		`{"type":"projection","sessionId":"foreign","key":"subagentCatalog","seq":99,"value":[]}`,
	} {
		apply(raw)
		if len(s.AgentNavigationSnapshot().Rows) != 1 {
			t.Fatalf("stale/malformed/foreign changed rows: %s", raw)
		}
	}
	old := float64(9)
	if err := seedAgentNavigationCatalog(&s.agentNavigation.view, json.RawMessage(`{"subagentCatalog":[]}`), &old, s.items); err != nil {
		t.Fatal(err)
	}
	if len(s.AgentNavigationSnapshot().Rows) != 1 {
		t.Fatal("late RPC/follow seed regressed push")
	}
	owner := s.BeginEvents(context.Background(), client)
	s.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/status", Args: []json.RawMessage{json.RawMessage(`"child"`), json.RawMessage(`true`)}})
	if !s.AgentNavigationSnapshot().Rows[0].Running {
		t.Fatal("running state not refreshed from live Items")
	}
	s.items[0].Title = "New live title"
	if s.SyncAgentNavigation(s.planHistoryBinding).Rows[0].Label != "New live title" {
		t.Fatal("catalog fallback title stale")
	}
	// Whole replacements keep Host order, including distinct facts for the same id.
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":12,"value":[{"id":"child","mode":"unknown","createdAt":1},{"id":"child","mode":"continuable","label":"Explicit label","createdAt":2}]}`)
	rows = s.AgentNavigationSnapshot().Rows
	if len(rows) != 2 || rows[0].Address.Mode != "unknown" || rows[1].Address.Mode != "continuable" || rows[1].Label != "Explicit label" {
		t.Fatalf("Host fact order/identity changed: %+v", rows)
	}
	// Existing path parents receive metadata, never new follows or recursive reads.
	s.agentNavigation.parents = []AgentNavigationView{cloneAgentNavigationView(s.agentNavigation.view)}
	childAddress := rows[1].Address
	s.agentNavigation.view = AgentNavigationView{Token: 8, SessionID: "child", Address: childAddress, Depth: 1}
	apply(`{"type":"projection","sessionId":"child","key":"subagentCatalog","seq":3,"value":[{"id":"leaf","mode":"one-shot","label":"Worker","createdAt":1}]}`)
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":13,"value":[]}`)
	child := s.AgentNavigationSnapshot()
	if len(child.Rows) != 1 || child.Rows[0].Address.ParentSessionID != "child" || child.Address != childAddress || child.Token != 8 {
		t.Fatalf("nested binding changed: %+v", child)
	}
	s.BackAgentNavigation(8)
	if len(s.AgentNavigationSnapshot().Rows) != 0 {
		t.Fatal("Back restored stale catalog")
	}
	apply(`{"type":"baseline","value":{"projections":{"root":{"asOfSeq":13,"values":{}}}}}`)
	if len(s.AgentNavigationSnapshot().Rows) != 1 {
		t.Fatal("same-cut omitted capability did not restore lineage fallback")
	}
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":13,"value":[]}`)
	if len(s.AgentNavigationSnapshot().Rows) != 0 {
		t.Fatal("absent capability incorrectly retained a watermark")
	}
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":14,"value":[]}`)
	if len(s.AgentNavigationSnapshot().Rows) != 0 {
		t.Fatal("empty projection merged instead of replaced")
	}
	token := s.agentNavigation.view.Token
	// A late catalog frame cannot cross the owning Host or selected parent.
	s.client = &dsh.Client{}
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":15,"value":[{"id":"bad","mode":"one-shot"}]}`)
	if len(s.agentNavigation.view.Rows) != 0 {
		t.Fatal("old Host mutated catalog")
	}
	s.client = client
	s.selectedID = "other"
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":16,"value":[{"id":"bad","mode":"one-shot"}]}`)
	if len(s.agentNavigation.view.Rows) != 0 || s.agentNavigation.view.Token != token {
		t.Fatal("parent scope escaped")
	}
	s.selectedID = "root"
	s.CloseAgentNavigation()
	apply(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":17,"value":[{"id":"bad","mode":"one-shot"}]}`)
	if s.agentNavigation.binding != "" || len(s.agentNavigation.view.Rows) != 0 {
		t.Fatal("closed pane reactivated")
	}
	if s.selectedID != "root" || len(s.messages) != 0 {
		t.Fatal("catalog changed main chat")
	}
}

// A catalog push may invalidate a queued click between admission and dispatch.
// The synchronous first notify makes that interleaving deterministic, without
// a server, sleeps, real action or transport. Existing owner fences stay intact.
func TestAgentCatalogStaleActionFixture(t *testing.T) {
	for _, prompt := range []bool{false, true} {
		s := New()
		var requests atomic.Int32
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://catalog-fixture.invalid"}, HTTP: &http.Client{Transport: agentCatalogNoTransport{calls: &requests}}}
		mux := &dsh.Mux{}
		s.client, s.mux, s.selectedID = client, mux, "root"
		s.items = []Item{{SessionID: "root", AgentAvailable: true}}
		s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, "root")
		address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
		s.agentNavigation = agentNavigationState{binding: s.planHistoryBinding, root: "root", client: client, mux: mux, generation: s.followGeneration, view: AgentNavigationView{Token: 7, SessionID: "root", Address: AgentSessionAddress{Kind: "session", SessionID: "root"}, Rows: []AgentNavigationRow{{Address: address}}}}
		finished := make(chan struct{}, 1)
		var notifications atomic.Int32
		s.notify = func() {
			if notifications.Add(1) == 1 {
				s.mu.Lock()
				s.applyAgentNavigationControlLocked(json.RawMessage(`{"type":"projection","sessionId":"root","key":"subagentCatalog","seq":1,"value":[]}`))
				s.mu.Unlock()
				return
			}
			finished <- struct{}{}
		}
		if prompt {
			s.PromptAgentNavigation(7, address, "synthetic input")
		} else {
			s.InterruptAgentNavigation(7, []AgentSessionAddress{address})
		}
		select {
		case <-finished:
		case <-time.After(2 * time.Second):
			t.Fatal("stale action left busy state without completion")
		}
		v := s.AgentNavigationSnapshot()
		if v.ActionBusy || !strings.Contains(v.ActionStatus, "target is no longer") || s.agentNavigation.actionCancel != nil || requests.Load() != 0 {
			t.Fatalf("stale action failed to settle without RPC: prompt=%v busy=%v status=%q requests=%d", prompt, v.ActionBusy, v.ActionStatus, requests.Load())
		}
	}
}

type agentCatalogNoTransport struct{ calls *atomic.Int32 }

func (r agentCatalogNoTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls.Add(1)
	return nil, errors.New("synthetic fixture forbids all network transport")
}
