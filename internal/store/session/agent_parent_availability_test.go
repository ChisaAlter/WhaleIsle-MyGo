package session

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestAgentParentAvailability(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	s.client = client
	s.selectedID = "root"
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	s.agentNavigation = agentNavigationState{client: client, root: "root", binding: s.planHistoryBinding, view: AgentNavigationView{Token: 7, Address: address}}
	if known, available := s.agentParentAvailabilityLocked(address); known || available {
		t.Fatal("missing parent not unknown")
	}
	s.items = []Item{{SessionID: "parent", AgentAvailable: true}}
	if known, available := s.agentParentAvailabilityLocked(address); !known || !available {
		t.Fatal("known live parent not available")
	}
	s.items[0].AgentAvailable = false
	if known, available := s.agentParentAvailabilityLocked(address); !known || available {
		t.Fatal("offline parent not distinguished")
	}
	if !s.agentNavigationActionAllowedLocked(7, address) {
		t.Fatal("offline parent wrongly revoked interrupt/history authority")
	}
}

func TestAgentUnavailableParentKeepsInterrupt(t *testing.T) {
	s := New()
	var calls atomic.Int32
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: agentCatalogNoTransport{calls: &calls}}}
	s.client = client
	s.selectedID = "root"
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	a := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	s.items = []Item{{SessionID: "root", AgentAvailable: false}, {SessionID: "child", Running: true}}
	s.agentNavigation = agentNavigationState{client: client, root: "root", binding: s.planHistoryBinding, view: AgentNavigationView{Token: 7, Address: a}}
	s.PromptAgentNavigation(7, a, "must not send")
	if s.AgentNavigationSnapshot().ActionBusy || calls.Load() != 0 {
		t.Fatal("offline prompt admitted")
	}
	s.InterruptAgentNavigation(7, []AgentSessionAddress{a})
	deadline := time.Now().Add(time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() != 1 {
		t.Fatal("offline parent blocked child Stop")
	}
}
func TestAgentParentUsesCurrentOwnerSummary(t *testing.T) {
	owner := New()
	owner.items = []Item{{SessionID: "parent", AgentAvailable: true}}
	pane := owner.AgentNavigationPane("test")
	pane.SyncAgentNavigationPane()
	owner.mu.Lock()
	owner.items[0].AgentAvailable = false
	owner.mu.Unlock()
	known, available := pane.agentParentAvailabilityLocked(AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"})
	if !known || available {
		t.Fatal("stale pane summary enabled continuation")
	}
}
