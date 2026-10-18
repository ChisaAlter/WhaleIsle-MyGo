package main

import (
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

func TestProjectObserverControllersRetainIndependentState(t *testing.T) {
	parent := &mainState{dshClient: &dsh.Client{}}
	parent.projectFlow = projectFlowState{client: parent.dshClient}
	a := &projects.ObserverState{ProjectID: "a", CoordinatorSessionID: "lead-a", Section: "tasks"}
	b := &projects.ObserverState{ProjectID: "b", CoordinatorSessionID: "lead-b", Section: "materials"}
	ca := parent.observerControllerLocked(a)
	cb := parent.observerControllerLocked(b)
	if ca == cb || ca.projectFlow == cb.projectFlow || ca.projectFlow == &parent.projectFlow {
		t.Fatal("observers share the mutable global controller")
	}
	ca.projectFlow.requestProjectID = "a"
	ca.projectFlow.requestKind = "completed"
	ca.projectFlow.requestCursors = []string{"", "a-page-2"}
	ca.projectFlow.requestCard = &projectflow.RequestCard{ID: "a-card", ProjectID: "a"}
	cb.projectFlow.requestProjectID = "b"
	cb.projectFlow.requestKind = "archived"
	cb.projectFlow.requestCursors = []string{"", "b-page-2", "b-page-3"}
	cb.projectFlow.requestCard = &projectflow.RequestCard{ID: "b-card", ProjectID: "b"}
	ca.projectFlow.materials = map[string]*projectflow.MaterialDraft{"a": {ProjectID: "a", Draft: "unsaved a", Saved: "a", Loaded: true}}
	cb.projectFlow.materials = map[string]*projectflow.MaterialDraft{"b": {ProjectID: "b", Draft: "unsaved b", Saved: "b", Loaded: true}}
	// This is the same callback the mounted native observer invokes. Its receiver
	// must remain attached to A even after B has been selected/rendered.
	ma := ca.projectModel(nil)
	_ = cb.projectModel(nil)
	ma.OnBackRequests()
	if ca.projectFlow.requestCard != nil || cb.projectFlow.requestCard == nil || cb.projectFlow.requestCard.ID != "b-card" {
		t.Fatal("A callback modified B's selected card")
	}
	if len(ca.projectFlow.requestCursors) != 2 || len(cb.projectFlow.requestCursors) != 3 || cb.projectFlow.requestKind != "archived" {
		t.Fatal("request pagination did not remain independently retained")
	}
	if parent.observerControllerLocked(a) != ca || parent.observerControllerLocked(b) != cb {
		t.Fatal("switching coordinator replaced its observer controller")
	}
	if !ca.projectFlow.materials["a"].Dirty() || !cb.projectFlow.materials["b"].Dirty() {
		t.Fatal("switching observer lost material drafts")
	}
	if parent.projectFlow.requestCard != nil || parent.projectFlow.requestProjectID != "" {
		t.Fatal("observer mutated sidebar controller")
	}
	ca.disposeLocked()
	if cb.projectFlow.client != parent.dshClient || cb.projectFlow.requestCard == nil {
		t.Fatal("disposing A invalidated B")
	}
}
