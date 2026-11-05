package session

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/dsh"
)

func retainedOwnerFixture() (*Store, *Store, AgentSessionAddress) {
	root := New()
	root.client = &dsh.Client{}
	root.mux = &dsh.Mux{}
	root.selectedID = "root"
	root.planHistoryBinding = fmt.Sprintf("%p\x00root", root.client)
	root.items = []Item{{SessionID: "root", AgentAvailable: true}}
	pane := root.NewRetainedAgentNavigationPane("same-pane")
	pane.SyncAgentNavigationPane()
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	pane.agentNavigation = agentNavigationState{client: pane.client, mux: pane.mux, root: pane.selectedID, binding: pane.planHistoryBinding, generation: pane.followGeneration, view: AgentNavigationView{Token: 7, SessionID: "child", Address: address, Depth: 1}}
	pane.agentNavigation.view.Process.Live = true
	pane.mu.Lock()
	pane.retainAgentReadLocked(address)
	pane.mu.Unlock()
	return root, pane, address
}
func TestRetainedAgentReadSurvivesSelectionWithoutActions(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	lease := pane.agentRetainedRead
	root.mu.Lock()
	root.selectSessionLocked("other")
	root.planHistoryBinding = fmt.Sprintf("%p\x00other", root.client)
	root.mu.Unlock()
	pane.SyncAgentNavigationPane()
	if !pane.RetainedAgentReadValid() || pane.selectedID != "root" || !pane.agentNavigationLiveCurrentLocked(lease.ctx, 7, pane.client, pane.mux, address) {
		t.Fatal("captured read lost or rebound")
	}
	snap := pane.AgentNavigationSnapshot()
	if !snap.RetainedReadOnly || snap.Address != address {
		t.Fatalf("read-only projection %+v", snap)
	}
	if pane.agentNavigationActionAllowedLocked(7, address) {
		t.Fatal("prompt/interrupt authority crossed root")
	}
	if _, _, _, ok := pane.AgentInteractionTarget(7, address); ok {
		t.Fatal("interaction authority crossed root")
	}
	pane.PromptAgentNavigation(7, address, "blocked")
	pane.InterruptAgentNavigation(7, []AgentSessionAddress{address})
	if pane.agentNavigation.view.ActionBusy {
		t.Fatal("action started")
	}
	if pane.prepareAgentReadReopen("root") {
		t.Fatal("reopen stole current parent")
	}
	root.mu.Lock()
	root.selectSessionLocked("root")
	root.planHistoryBinding = fmt.Sprintf("%p\x00root", root.client)
	root.mu.Unlock()
	if pane.agentNavigationActionAllowedLocked(7, address) {
		t.Fatal("return silently renewed action generation")
	}
	if !pane.prepareAgentReadReopen("root") || lease.ctx.Err() == nil {
		t.Fatal("explicit owner renewal did not retire old lease")
	}
}
func TestRetainedAgentOccurrenceExactRelease(t *testing.T) {
	root, old, _ := retainedOwnerFixture()
	fresh := root.NewRetainedAgentNavigationPane("same-pane")
	if fresh == old || fresh.AgentNavigationOccurrenceID() == old.AgentNavigationOccurrenceID() {
		t.Fatal("pane ID reused ownership")
	}
	root.ReleaseAgentNavigationOccurrence(old)
	root.ReleaseAgentNavigationOccurrence(old)
	if root.AgentNavigationPaneByID(fresh.AgentNavigationOccurrenceID()) != fresh || old.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("old disposal affected replacement or leaked lease")
	}
}
func TestRetainedAgentRevocationEventsAndHost(t *testing.T) {
	for _, event := range []string{"api-session/deleted", "api-session/removed"} {
		root, pane, _ := retainedOwnerFixture()
		emit := dsh.EventsEmit{Type: dsh.EventEmit, Event: event, Args: []json.RawMessage{json.RawMessage(`"root"`)}}
		stale := testEventsOwner(root, context.Background())
		stale.generation++
		root.OnEmit(stale, emit)
		if !pane.RetainedAgentReadValid() {
			t.Fatal("stale event revoked read")
		}
		root.OnEmit(testEventsOwner(root, context.Background()), emit)
		if pane.agentRetainedRead.ctx.Err() == nil || pane.RetainedAgentReadValid() {
			t.Fatal("accepted root event did not cancel")
		}
	}
	root, pane, _ := retainedOwnerFixture()
	root.Detach()
	if pane.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("detach leak")
	}
	root, pane, _ = retainedOwnerFixture()
	root.Attach(&dsh.Client{}, nil)
	if pane.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("host switch leak")
	}
	root, pane, _ = retainedOwnerFixture()
	root.BeginEvents(context.Background(), root.client)
	if pane.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("reconnect assumed missed revocation absent")
	}
}
func TestRetainedAgentReadContextCancellation(t *testing.T) {
	root, pane, _ := retainedOwnerFixture()
	ctx, cancel := context.WithCancel(pane.agentReadContextLocked())
	defer cancel()
	root.mu.Lock()
	root.cancelRetainedAgentReadsLocked("child")
	root.mu.Unlock()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("read transport context outlived revoke")
	}
	if !pane.HasRetainedAgentRead() {
		t.Fatal("revocation tombstone lost; render could auto reopen")
	}
	// A cancelled lease cannot be revived by render sync even at the same root.
	pane.SyncAgentNavigationPane()
	if pane.agentPaneCurrentLocked() || pane.agentNavigationOwnedLocked() {
		t.Fatal("revoked owner revived")
	}
}

func TestRetainedAgentIdleChildDisposalKeepsConversationUntilDeleted(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	defer root.ReleaseAgentNavigationOccurrence(pane)
	root.items = append(root.items, Item{SessionID: address.ChildSessionID, ParentID: address.ParentSessionID, Origin: "subagent", Running: true, AgentAvailable: true})
	pane.agentNavigation.view.Messages = []Message{{Role: "assistant", Body: "durable child final", Seq: 23}}
	emit := dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/removed", Args: []json.RawMessage{json.RawMessage(`"child"`)}}
	root.OnEmit(testEventsOwner(root, context.Background()), emit)
	v := pane.AgentNavigationSnapshot()
	if !pane.RetainedAgentReadValid() || v.Address != address || len(v.Messages) != 1 || !v.RetainedPromptAllowed {
		t.Fatalf("idle child disposal blanked or disabled its retained conversation: %+v", v)
	}
	if root.items[1].Running || root.items[1].AgentAvailable {
		t.Fatal("idle child retained live-registry availability")
	}
	emit.Event = "api-session/deleted"
	root.OnEmit(testEventsOwner(root, context.Background()), emit)
	if pane.RetainedAgentReadValid() || pane.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("explicit child deletion failed to revoke the same read")
	}
}
