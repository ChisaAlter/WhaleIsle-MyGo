package frame

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

func TestProjectObserverRetainsCoordinatorResource(t *testing.T) {
	w := &WorkspacePanes{}
	m := projects.Model{ObserverHost: "host-a", SelectedSession: "lead-a", Listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "project-a", CoordinatorSessionID: "lead-a"}, {ID: "project-b", CoordinatorSessionID: "lead-b"}}}}
	if w.OpenProjectObserver(m, "project-b") != nil {
		t.Fatal("observer attached to another coordinator")
	}
	n := w.OpenProjectObserver(m, "project-a")
	if n == nil || n.Float == nil || n.Kind != "project-observer" {
		t.Fatal("existing Project did not open a real floating observer")
	}
	n.ProjectObserver.Section = "materials"
	w.ResizeWorkspaceFloat(n.ID, 490, 540)
	w.MinimizeWorkspaceFloat(n.ID, true)
	again := w.OpenProjectObserver(m, "project-a")
	if again != n || again.ProjectObserver.Section != "materials" || again.Float.Minimized || again.Float.Height != 540 {
		t.Fatal("reopen replaced state or failed to restore full presentation")
	}
	model := Model{Projects: m}
	if !projectObserverVisible(model, n) {
		t.Fatal("matching coordinator unavailable")
	}
	model.Projects.SelectedSession = "lead-b"
	if projectObserverVisible(model, n) {
		t.Fatal("observer visible in wrong session")
	}
	model.Projects = m
	model.Projects.ObserverHost = "host-b"
	if projectObserverVisible(model, n) {
		t.Fatal("observer accepted unrelated Host")
	}
	raw, err := json.Marshal(n.ProjectObserver)
	if err != nil {
		t.Fatal(err)
	}
	var restored projects.ObserverState
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Host != "" || restored.Section != "materials" {
		t.Fatal("runtime authority persisted or retained route lost")
	}
	if _, ok := restored.Project(m); ok {
		t.Fatal("restored layout bypassed explicit Host rebinding")
	}
	count := 0
	w.Root.eachResource(func(p *WorkspacePane) {
		if p.Kind == "project-observer" {
			count++
		}
	})
	if count != 1 {
		t.Fatalf("observer duplicated: %d", count)
	}
}
