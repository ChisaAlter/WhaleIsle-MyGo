package session

import (
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/dsh"
)

func TestAgentNavigationExternalMetadataCannotOpen(t *testing.T) {
	rows, err := agentNavigationRows("root", json.RawMessage(`{"subagentCatalog":[{"id":"outside","label":"External analysis","mode":"external"},{"id":"ordinary","mode":"continuable"}]}`), []Item{{SessionID: "outside", ParentID: "root", Title: "Fallback", Running: true}})
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows: %+v %v", rows, err)
	}
	if rows[0].Address.Mode != "external" || rows[0].Label != "External analysis" || rows[0].Address.ChildSessionID != "outside" || rows[0].CanOpen() {
		t.Fatalf("external metadata/openness: %+v", rows[0])
	}
	if !rows[1].CanOpen() {
		t.Fatal("ordinary child lost navigation")
	}
	s := New()
	client := &dsh.Client{}
	mux := &dsh.Mux{}
	s.client, s.mux, s.selectedID = client, mux, "root"
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	s.agentNavigation = agentNavigationState{binding: s.planHistoryBinding, root: "root", client: client, mux: mux, generation: s.followGeneration, view: AgentNavigationView{Token: 7, SessionID: "root", Address: AgentSessionAddress{Kind: "session", SessionID: "root"}, Rows: rows}}
	external := rows[0].Address
	s.OpenAgentNavigation(7, external)
	forged := external
	forged.Mode = "unknown"
	s.OpenAgentNavigation(7, forged)
	v := s.AgentNavigationSnapshot()
	if v.SessionID != "root" || v.Depth != 0 || v.Busy || len(s.agentNavigation.parents) != 0 || s.agentNavigation.cancel != nil {
		t.Fatalf("external started navigation/follow: %+v", v)
	}
	s.mu.Lock()
	allowed := s.agentNavigationActionAllowedLocked(7, external) || s.agentNavigationActionAllowedLocked(7, forged)
	s.mu.Unlock()
	if allowed {
		t.Fatal("external admitted actions")
	}
}
