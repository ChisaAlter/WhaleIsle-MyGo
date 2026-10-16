package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/projects"
)

// OpenProjectObserver only attaches a resource to an existing coordinator. No
// session creation, Host replacement, or independent copy of project data occurs.
func (w *WorkspacePanes) OpenProjectObserver(m projects.Model, id string) *WorkspacePane {
	if w == nil || m.ObserverHost == "" || !m.Listing.Available {
		return nil
	}
	for _, p := range m.Listing.Projects {
		if p.ID != id || p.CoordinatorSessionID == "" || p.CoordinatorSessionID != m.SelectedSession {
			continue
		}
		var found *WorkspacePane
		w.Root.eachResource(func(n *WorkspacePane) {
			if n.Kind == "project-observer" && n.ProjectObserver != nil && n.ProjectObserver.ProjectID == p.ID && n.ProjectObserver.CoordinatorSessionID == p.CoordinatorSessionID {
				found = n
			}
		})
		if found == nil {
			if w.Root == nil {
				w.Root = w.leaf("files")
				w.Active = w.Root.ID
			}
			found = w.leaf("project-observer")
			found.ProjectObserver = &projects.ObserverState{ProjectID: p.ID, CoordinatorSessionID: p.CoordinatorSessionID, Host: m.ObserverHost, Section: "tasks"}
			group := w.Root.dockGroup(w.Active)
			if group == nil {
				group = w.Root.firstLeaf()
			}
			setDockTabs(group, append(group.dockTabs(), found))
		}
		if found.ProjectObserver.Host != m.ObserverHost {
			found.ProjectObserver.Host = m.ObserverHost
			found.ProjectObserver.Initialized = false
		}
		w.FloatWorkspaceTab(found.ID)
		return found
	}
	return nil
}
func projectObserverVisible(m Model, n *WorkspacePane) bool {
	if n.Kind != "project-observer" {
		return true
	}
	_, ok := n.ProjectObserver.Project(m.Projects)
	return ok
}
func openProjectObserver(m Model, id string) {
	if m.Sizes == nil {
		return
	}
	if m.Sizes.Workspace == nil {
		m.Sizes.Workspace = &WorkspacePanes{}
	}
	if n := m.Sizes.Workspace.OpenProjectObserver(m.Projects, id); n != nil {
		if m.Projects.OnObserver != nil {
			m.Projects.OnObserver(id)
		}
		if m.Projects.OnObserverSection != nil {
			m.Projects.OnObserverSection(id, n.ProjectObserver.Section)
		}
	}
}
func projectObserverHeader(c *ui.Context, m Model) {
	if !m.Projects.Listing.Available {
		return
	}
	for _, p := range m.Projects.Listing.Projects {
		if p.CoordinatorSessionID == "" || p.CoordinatorSessionID != m.Projects.SelectedSession {
			continue
		}
		ui.Row(c).FillWidth().Gap(4).Children(func() {
			if ui.Button(c, paneLabel("项目概览", "Project overview")).Clicked() {
				openProjectObserver(m, p.ID)
			}
			ui.Text(c, p.Title+" · "+p.Lifecycle)
		})
		return
	}
}
