package frame

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/projects"
)

// Only routing facts survive restoration; a live view never follows a new Host.
type WorkspaceOrdinaryTeamState struct {
	SessionID string `json:"sessionId"`
	Binding   string `json:"-"`
}

func ordinaryTeamResourceEntry(m Model) workspaceResourceEntry {
	return workspaceResourceEntry{"team", paneLabel("团队", "Team"), paneLabel("查看成员及共享任务", "View members and shared tasks"), conversationHeaderIcons["UsersOutline"], m.Store == nil || m.Store.SelectedSessionID() == "", func() {
		if m.Projects.ProjectTeam != nil {
			for _, p := range m.Projects.Listing.Projects {
				if p.CoordinatorSessionID == m.Store.SelectedSessionID() {
					_ = openProjectTeamSidebar(m, m.Projects.ProjectTeam(p.ID))
					return
				}
			}
		}
		if err := openOrdinaryTeamSidebar(m); err != nil && m.Sizes != nil && m.Sizes.Workspace != nil {
			m.Sizes.Workspace.message = err.Error()
		}
	}}
}

func openOrdinaryTeamSidebar(m Model) error {
	if m.Store == nil || m.OrdinaryTeam == nil || m.Sizes == nil || m.Sizes.Workspace == nil || m.Store.SelectedSessionID() == "" {
		return errors.New(paneLabel("暂时无法打开团队侧栏，请重试。", "The Team Sidebar cannot open right now. Try again."))
	}
	w := m.Sizes.Workspace
	sid, binding := m.Store.SelectedSessionID(), m.Store.PlanHistoryBinding()
	var target *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Kind == "team" && n.OrdinaryTeam != nil && n.OrdinaryTeam.SessionID == sid && (n.OrdinaryTeam.Binding == "" || n.OrdinaryTeam.Binding == binding) {
			target = n
		}
	})
	if target == nil {
		target = w.leaf("team")
		target.OrdinaryTeam = &WorkspaceOrdinaryTeamState{SessionID: sid, Binding: binding}
		source := w.Root.find(w.Active)
		if w.Root == nil {
			w.Root = target
		} else {
			w.placeDockResource(target, source, false)
		}
		if source != nil && source.Kind == "guide" {
			w.disposeWorkspaceResource(source.ID)
		}
	}
	w.activateDockTab(target.ID)
	w.focusPending = target.ID
	if target.Float != nil {
		w.MinimizeWorkspaceFloat(target.ID, false)
	}
	if m.RightPaneTab != nil {
		*m.RightPaneTab = "team"
	}
	if !m.RightPaneOpen && m.OnToggleTools != nil {
		m.OnToggleTools()
	}
	return nil
}

func workspaceOrdinaryTeam(c *ui.Context, m Model) {
	n := m.workspacePane
	if n == nil || m.Store == nil || m.OrdinaryTeam == nil {
		return
	}
	if n.OrdinaryTeam == nil {
		n.OrdinaryTeam = &WorkspaceOrdinaryTeamState{SessionID: m.Store.SelectedSessionID()}
	}
	s := n.OrdinaryTeam
	snap := m.Store.Snapshot()
	if s.Binding == "" {
		s.Binding = snap.PlanHistoryBinding
	}
	if s.SessionID != snap.SelectedID || s.Binding != snap.PlanHistoryBinding {
		ui.Text(c, paneLabel("请从来源会话重新打开团队。", "Reopen the Team from its source conversation.")).Padding(12)
		return
	}
	model := m.OrdinaryTeam()
	model.ThemeID, model.SystemDark = m.ThemeID, m.SystemDark
	model.Columns = 1
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Padding(8, 0, 0, 0).Children(func() {
		if model.State != nil && model.State.NavigationError != "" {
			model.Error = model.State.NavigationError
		}
		model.OnOpenMember = func(id string) {
			fail := func(message string) {
				if model.State != nil {
					model.State.NavigationError = message
				}
			}
			if err := openTeamMember(m, snap, id, true, fail); err != nil {
				fail(err.Error())
			}
		}
		projects.OrdinaryTeamView(c, model)
	})
}
