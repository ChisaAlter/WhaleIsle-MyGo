package main

import (
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/projects"
)

func TestProjectObserverControllersRetainIndependentStateViaShell(t *testing.T) {
	parent := &mainState{dshClient: &dsh.Client{}}
	parentFrame := frame.Model{ThemeID: "frame-theme", SystemDark: true}
	parent.bindProjectModel(nil, &parentFrame)
	if parentFrame.Projects.ObserverModel == nil {
		t.Fatal("shell did not bind ObserverModel factory")
	}
	a := &projects.ObserverState{ProjectID: "a", CoordinatorSessionID: "lead-a", Section: "tasks"}
	b := &projects.ObserverState{ProjectID: "b", CoordinatorSessionID: "lead-b", Section: "materials"}
	mgr := parent.projectManager()
	ca := mgr.Observer(a)
	cb := mgr.Observer(b)
	if ca == cb || ca == mgr.Primary() || cb == mgr.Primary() {
		t.Fatal("observer controllers share mutable primary state")
	}
	if mgr.Observer(a) != ca || mgr.Observer(b) != cb {
		t.Fatal("observer controller lookup was not stable")
	}
	ma := parentFrame.Projects.ObserverModel(a)
	mb := parentFrame.Projects.ObserverModel(b)
	if ma.ThemeID != "frame-theme" || !ma.SystemDark || mb.ThemeID != "frame-theme" || !mb.SystemDark {
		t.Fatal("observer models did not retain parent frame theme")
	}
	if ma.OnBackRequests == nil || mb.OnBackRequests == nil || ma.VisualPage == nil || mb.VisualPage == nil {
		t.Fatal("observer models missing expected callbacks")
	}
	ma.OnBackRequests()
	ca.Dispose()
	mgr.Suspend()
	mgr.Close()
}
