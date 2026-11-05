package frame

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

// A dedicated requirements occurrence moves between the header and Sidebar.
func (w *WorkspacePanes) OpenProjectObserver(m projects.Model, id string) *WorkspacePane {
	return w.openProjectObserverResource(m, id, true)
}

func (w *WorkspacePanes) openProjectObserverResource(m projects.Model, id string, floating bool) *WorkspacePane {
	if w == nil || m.ObserverHost == "" || !m.Listing.Available {
		return nil
	}
	for _, p := range m.Listing.Projects {
		if p.ID != id || p.CoordinatorSessionID == "" || p.CoordinatorSessionID != m.SelectedSession {
			continue
		}
		var target *WorkspacePane
		w.Root.eachResource(func(n *WorkspacePane) {
			if n.Kind == "project-observer" && n.ProjectObserver != nil && n.ProjectObserver.ProjectID == id && n.ProjectObserver.CoordinatorSessionID == p.CoordinatorSessionID {
				target = n
			}
		})
		if target == nil {
			target = w.leaf("project-observer")
			target.ProjectObserver = &projects.ObserverState{ProjectID: id, CoordinatorSessionID: p.CoordinatorSessionID, Host: m.ObserverHost, Section: "tasks"}
			source := w.Root.find(w.Active)
			if w.Root == nil {
				w.Root = target
			} else {
				w.placeDockResource(target, source, false)
			}
		}
		w.activateDockTab(target.ID)
		if floating {
			w.FloatWorkspaceTab(target.ID)
		}
		w.focusPending = target.ID
		return target
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
	if n := m.Sizes.Workspace.openProjectObserverResource(m.Projects, id, false); n != nil {
		n.Float = nil
		n.ProjectObserver.Pinned = true
		n.ProjectObserver.FloatingOpen = false
		if m.RightPaneTab != nil {
			*m.RightPaneTab = "project-observer"
		}
		if !m.RightPaneOpen && m.OnToggleTools != nil {
			m.OnToggleTools()
		}
	}
}
func pinProjectRequirements(m Model, o *projects.ObserverState) {
	if m.Sizes == nil {
		return
	}
	if m.Sizes.Workspace == nil {
		m.Sizes.Workspace = &WorkspacePanes{}
	}
	n := m.Sizes.Workspace.openProjectObserverResource(m.Projects, o.ProjectID, false)
	if n == nil {
		return
	}
	n.ProjectObserver = o
	n.Float = nil
	o.Pinned = true
	o.FloatingOpen = false
	if m.RightPaneTab != nil {
		*m.RightPaneTab = "project-observer"
	}
	if !m.RightPaneOpen && m.OnToggleTools != nil {
		m.OnToggleTools()
	}
}
func returnProjectRequirements(m Model, n *WorkspacePane, o *projects.ObserverState) {
	w := m.Sizes.Workspace
	docked := n.Float == nil
	w.pending = func() {
		if !w.close(n.ID) {
			return
		}
		o.Pinned = false
		o.FloatingOpen = true
		onlyGuide := true
		w.Root.eachResource(func(p *WorkspacePane) { onlyGuide = onlyGuide && p.Kind == "guide" })
		if docked && onlyGuide && m.RightPaneOpen && m.OnToggleTools != nil {
			m.OnToggleTools()
		}
	}
}
func requestNavigationModel(m Model, model projects.Model) projects.Model {
	original := model.RequestCardActions
	if original != nil {
		model.RequestCardActions = func(id string, revision int) projects.RequestCardActions {
			actions := original(id, revision)
			actions.OpenCurrent = func(work projectflow.RequestWork) error {
				return openTeamMember(m, m.Store.Snapshot(), work.WorkerSessionID, false, func(message string) {
					if m.Store != nil {
						m.Store.SetLastAction(message)
					}
				})
			}
			return actions
		}
	}
	return model
}
func workspaceProjectRequirements(c *ui.Context, m Model) {
	n := m.workspacePane
	if n == nil || n.ProjectObserver == nil {
		return
	}
	o := n.ProjectObserver
	if o.Host == "" {
		o.Host = m.Projects.ObserverHost
	}
	o.Pinned = true
	o.OnReturn = func() { returnProjectRequirements(m, n, o) }
	o.OnPin = nil
	model := m.Projects
	if model.ObserverModel != nil {
		model = model.ObserverModel(o)
	}
	model = requestNavigationModel(m, model)
	projects.ObserverView(c, model, o)
}
func projectObserverHeader(c *ui.Context, m Model) {
	if !m.Projects.Listing.Available {
		return
	}
	for _, p := range m.Projects.Listing.Projects {
		if p.CoordinatorSessionID == "" || p.CoordinatorSessionID != m.Projects.SelectedSession {
			continue
		}
		owner := ui.Row(c.Key("project-requirements-header:" + p.ID)).Shrink(0)
		local := ui.Local(owner, "requirements", func() projects.ObserverState {
			return projects.ObserverState{ProjectID: p.ID, CoordinatorSessionID: p.CoordinatorSessionID, Host: m.Projects.ObserverHost, Section: "tasks"}
		})
		o := local
		if m.Sizes != nil && m.Sizes.Workspace != nil {
			m.Sizes.Workspace.Root.eachResource(func(n *WorkspacePane) {
				if n.ProjectObserver != nil && n.ProjectObserver.ProjectID == p.ID && n.ProjectObserver.CoordinatorSessionID == p.CoordinatorSessionID {
					o = n.ProjectObserver
				}
			})
		}
		if o.Host != m.Projects.ObserverHost {
			*o = projects.ObserverState{ProjectID: p.ID, CoordinatorSessionID: p.CoordinatorSessionID, Host: m.Projects.ObserverHost, Section: "tasks"}
		}
		owner.Children(func() {
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			ink := chrome.Color(colors.Muted)
			directory := projectHeaderDirectory(c, m, p, ink)
			directory.Tooltip(p.CanonicalWorkingDirectory)
			count := 0
			if p.Queue != nil {
				count = len(p.Queue.Queued) + p.Queue.DiscussingCount
				if p.Queue.Active != nil {
					count++
				}
			}
			badge := fmt.Sprint(count)
			if count > 99 {
				badge = "99+"
			}
			status := projectRequirementStatus(p)
			if m.Projects.Busy && !m.Projects.Loaded {
				badge = "?"
				status = paneLabel("正在加载", "Loading")
			}
			anchor := ui.Button(c.Key("project-requirements-trigger"), "").Label(paneLabel("需求", "Requirements")+" · "+status).Tooltip(status).Height(28).Padding(0, 8).BorderWidth(0).Radius(6).Background(ui.Transparent)
			anchor.Children(func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					projects.HeaderIcon(c, "checklist", 14, ink)
					w, _ := viewportDIP(c)
					if w > 560 {
						ui.Text(c, paneLabel("需求", "Requirements")).FontSize(theme.InterfaceSize(13)).TextColor(ink)
					}
					ui.Text(c, badge).FontSize(theme.InterfaceSize(12)).TextColor(ink)
					if w > 720 {
						ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(ink)
					}
				})
			})
			if anchor.Clicked() {
				if o.Pinned {
					if m.Sizes != nil && m.Sizes.Workspace != nil {
						m.Sizes.Workspace.Root.eachResource(func(n *WorkspacePane) {
							if n.ProjectObserver == o {
								m.Sizes.Workspace.activateDockTab(n.ID)
							}
						})
					}
					if !m.RightPaneOpen && m.OnToggleTools != nil {
						m.OnToggleTools()
					}
				} else {
					o.FloatingOpen = !o.FloatingOpen
				}
			}
			open := o.FloatingOpen && !o.Pinned
			w, h := viewportDIP(c)
			width := min(float32(460), max(float32(0), w-32))
			ui.PopoverBase(c.Key("project-requirements-popover"), anchor, &open, func(panel ui.Element) {
				headerPopupInset(c, panel, anchor, width, w, 16)
				panel = chrome.PopoverReveal(c, panel.Label(paneLabel("需求", "Requirements")).Width(width).MaxHeight(min(float32(620), max(float32(0), h-32))).MinHeight(0).Margin(5, 0, 0, 0).Radius(12).Background(chrome.Color(colors.BG)).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .18)))
				panel.Children(func() {
					o.OnPin = func() { pinProjectRequirements(m, o); open = false }
					o.OnReturn = nil
					model := m.Projects
					if model.ObserverModel != nil {
						model = model.ObserverModel(o)
					}
					model = requestNavigationModel(m, model)
					projects.ObserverView(c, model, o)
				})
				if panel.OverlayShortcut(0, ui.KeyEscape) {
					open = false
					anchor.Focus()
				}
			})
			o.FloatingOpen = open
		})
		return
	}
}
func projectRequirementStatus(p projectflow.Project) string {
	switch {
	case p.Archiving:
		return paneLabel("正在停止", "Stopping")
	case p.Lifecycle == "archived":
		return paneLabel("已归档", "Archived")
	case p.Paused:
		return paneLabel("已暂停", "Paused")
	case p.Activity.Running > 0:
		return paneLabel("执行中", "Active")
	case p.Activity.Queued > 0:
		return paneLabel("等待中", "Waiting")
	case p.Activity.State != "":
		switch p.Activity.State {
		case "blocked", "summaryFailed", "needsReview":
			return paneLabel("待处理", "Needs attention")
		case "pendingSummary":
			return paneLabel("正在汇总", "Summarizing")
		case "pendingDelivery":
			return paneLabel("正在交接", "Handing over")
		case "done":
			return paneLabel("已交付", "Delivered")
		}
	}
	return paneLabel("待命", "Idle")
}
