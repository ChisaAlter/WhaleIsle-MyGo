package session

import "testing"

func nestedCatalogRoutingFixture() (*Store, AgentSessionAddress, AgentSessionAddress) {
	s := New()
	s.selectedID, s.planHistoryBinding = "root", "binding"
	a := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "a", Mode: "continuable"}
	leaf := AgentSessionAddress{Kind: "subagent", ParentSessionID: "a", ChildSessionID: "leaf", Mode: "continuable"}
	s.agentNavigation = agentNavigationState{root: "root", binding: "binding", view: AgentNavigationView{Token: 7, SessionID: "root", Title: "Root", Address: AgentSessionAddress{Kind: "session", SessionID: "root"}, Rows: []AgentNavigationRow{{Address: a, Label: "A"}}}, branches: map[string]*agentCatalogBranch{}}
	b := &agentCatalogBranch{expanded: true, view: AgentNavigationView{SessionID: "a", Address: a, Title: "A", Rows: []AgentNavigationRow{{Address: leaf, Label: "Leaf"}}}}
	b.view.catalog.ready = true
	b.view.catalog.values = []byte(`{"subagentCatalog":[{"id":"leaf","mode":"continuable","label":"Leaf"}]}`)
	s.agentNavigation.branches[agentCatalogKey(s.agentNavigation.view.Address, []AgentSessionAddress{a})] = b
	return s, a, leaf
}
func TestAgentCatalogDirectRoutingWithoutEntering(t *testing.T) {
	s, _, leaf := nestedCatalogRoutingFixture()
	root, title, ok := s.AgentChatTarget(7, leaf)
	if !ok || root != "root" || title != "Leaf" {
		t.Fatalf("nested route absent: %s %s %v", root, title, ok)
	}
	if s.agentNavigation.view.SessionID != "root" || len(s.agentNavigation.parents) != 0 || s.agentNavigation.cancel != nil {
		t.Fatal("routing activated navigation")
	}
}
func TestAgentCatalogDirectRoutingRevocationAndNoActionGrant(t *testing.T) {
	s, a, leaf := nestedCatalogRoutingFixture()
	if s.agentNavigationActionAllowedLocked(7, leaf) {
		t.Fatal("nested prompt/interrupt authority widened")
	}
	altered := leaf
	altered.Mode = "one-shot"
	if _, _, ok := s.AgentChatTarget(7, altered); ok {
		t.Fatal("mode forgery")
	}
	if _, _, ok := s.AgentChatTarget(6, leaf); ok {
		t.Fatal("stale token")
	}
	b := s.agentNavigation.branches[agentCatalogKey(s.agentNavigation.view.Address, []AgentSessionAddress{a})]
	b.expanded = false
	if _, _, ok := s.AgentChatTarget(7, leaf); ok {
		t.Fatal("collapsed branch retained route")
	}
	b.expanded = true
	s.agentNavigation.view.Rows = nil
	if _, _, ok := s.AgentChatTarget(7, leaf); ok {
		t.Fatal("removed parent retained route")
	}
	s, a, leaf = nestedCatalogRoutingFixture()
	s.selectedID = "other"
	if _, _, ok := s.AgentChatTarget(7, leaf); ok {
		t.Fatal("root switch retained route")
	}
}
func TestAgentNavigationAncestorJumpOwnership(t *testing.T) {
	s, a, leaf := nestedCatalogRoutingFixture()
	root := s.agentNavigation.view
	parent := s.agentNavigation.branches[agentCatalogKey(root.Address, []AgentSessionAddress{a})].view
	s.agentNavigation.parents = []AgentNavigationView{root, parent}
	s.agentNavigation.view = AgentNavigationView{Token: 7, SessionID: "leaf", Address: leaf, Depth: 2}
	ancestors := s.AgentNavigationAncestors(7)
	if len(ancestors) != 2 || ancestors[0].SessionID != "root" || ancestors[1].Depth != 1 {
		t.Fatalf("lineage %+v", ancestors)
	}
	s.JumpAgentNavigation(6, 0)
	if s.agentNavigation.view.SessionID != "leaf" {
		t.Fatal("stale jump")
	}
	cancelled := 0
	s.agentNavigation.cancel = func() { cancelled++ }
	s.agentNavigation.actionCancel = func() { cancelled++ }
	s.JumpAgentNavigation(7, 0)
	if s.agentNavigation.view.SessionID != "root" || len(s.agentNavigation.parents) != 0 || cancelled != 2 || s.agentNavigation.view.Token != 8 {
		t.Fatalf("jump=%+v canceled=%d", s.agentNavigation.view, cancelled)
	}
	if len(s.AgentNavigationAncestors(7)) != 0 {
		t.Fatal("old lineage remains")
	}
}
func TestAgentNavigationAncestorRejectsRemovedParent(t *testing.T) {
	s, a, leaf := nestedCatalogRoutingFixture()
	root := s.agentNavigation.view
	parent := s.agentNavigation.branches[agentCatalogKey(root.Address, []AgentSessionAddress{a})].view
	root.Rows = nil
	s.agentNavigation.parents = []AgentNavigationView{root, parent}
	s.agentNavigation.view = AgentNavigationView{Token: 7, SessionID: "leaf", Address: leaf, Depth: 2}
	s.JumpAgentNavigation(7, 1)
	if s.agentNavigation.view.SessionID != "leaf" {
		t.Fatal("revoked ancestor accepted")
	}
	s.JumpAgentNavigation(7, 0)
	if s.agentNavigation.view.SessionID != "root" {
		t.Fatal("root escape lost")
	}
}
