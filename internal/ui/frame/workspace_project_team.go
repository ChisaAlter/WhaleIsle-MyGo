package frame

import (
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

// Recovery records routing facts only. The Project owner supplies fresh state
// and authority each time the Team Sidebar is rendered.
type WorkspaceProjectTeamState struct {
	ProjectID, CoordinatorSessionID, Host string
}

func projectConversationTeamAction(c *ui.Context, m Model, snap session.ViewState, compact bool, ink ui.Color) bool {
	if m.Projects.ProjectTeam == nil || snap.SelectedID == "" {
		return false
	}
	for _, project := range m.Projects.Listing.Projects {
		if project.CoordinatorSessionID != snap.SelectedID {
			continue
		}
		model := m.Projects.ProjectTeam(project.ID)
		model.ThemeID, model.SystemDark = m.ThemeID, m.SystemDark
		if m.Sizes != nil && m.Sizes.Workspace != nil {
			if child := m.Sizes.Workspace.currentMainChild(m); child != nil {
				model.SessionID = child.SubagentChat.Address.ChildSessionID
			}
		}
		projectTeamHeader(c, m, snap, model, compact, ink)
		c.After(time.Second)
		return true
	}
	return false
}

func projectTeamHeader(c *ui.Context, m Model, snap session.ViewState, model projects.ProjectTeamModel, compact bool, ink ui.Color) {
	state := model.State
	if state == nil {
		return
	}
	owner := ui.Row(c.Key("project-team-action:" + model.Project.ID)).Shrink(0)
	owner.Children(func() {
		count := "?"
		if model.CountKnown {
			count = fmt.Sprint(model.RunningCount)
			if model.RunningCount > 99 {
				count = "99+"
			}
		}
		label := paneLabel("团队", "Team")
		hint := paneLabel("团队活动暂未确认。点击查看保留的记录。", "Team activity is unconfirmed. Click to inspect retained records.")
		if model.CountKnown {
			hint = fmt.Sprintf(paneLabel("%d 名成员正在运行，含后台命令。点击查看。", "%d members running, including background commands. Click to inspect."), model.RunningCount)
		}
		anchor := ui.Button(c.Key("conversation-project-team"), "").Label(paneLabel("智能体团队", "Agent Team")+" · "+count).Tooltip(hint).Height(28).MinWidth(28).Padding(0, 8).Radius(6).BorderWidth(0).Background(ui.Transparent)
		anchor.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				ui.Icon(c, conversationHeaderIcons["UsersOutline"]).Size(14, 14).TextColor(ink)
				if !compact {
					ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(ink)
				}
				ui.Text(c, count).FontSize(theme.InterfaceSize(12)).TextColor(ink)
			})
		})
		if anchor.Clicked() {
			state.SetFloating(!state.FloatingOpen)
		}
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		w, h := viewportDIP(c)
		width := min(float32(500), max(float32(0), w-32))
		if model.Overview != nil && model.Overview.Team != nil && len(model.Overview.Team.Members) == 1 && len(model.Overview.Team.Tasks) == 0 {
			width = min(float32(320), max(float32(0), w-32))
		}
		model.Columns = 1
		if w > 560 && width > 320 {
			model.Columns = 2
		}
		open := state.FloatingOpen
		ui.PopoverBase(c.Key("project-team-popover"), anchor, &open, func(panel ui.Element) {
			headerPopupInset(c, panel, anchor, width, w, 16)
			panel = panel.Label(paneLabel("智能体团队", "Agent Team")).Width(width).MaxHeight(min(float32(680), max(float32(0), h-32))).MinHeight(0).Margin(5, 0, 0, 0).Padding(8, 2, 0, 2).Radius(12).Border(.5, chrome.Color(colors.Border)).Background(chrome.Color(colors.BG)).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .18))
			panel = chrome.PopoverReveal(c, panel)
			panel.Children(func() {
				ui.Row(c).FillWidth().Justify(ui.End).AlignItems(ui.Center).Gap(8).Padding(4, 14, 8, 14).Children(func() {
					if ui.Button(c, paneLabel("持续观察", "Keep observing")).Clicked() {
						if err := openProjectTeamSidebar(m, model); err != nil {
							state.NavigationError = err.Error()
						} else {
							state.SetFloating(false)
							open = false
						}
					}
					if ui.Button(c, paneLabel("关闭", "Close")).Clicked() {
						state.SetFloating(false)
						open = false
						anchor.Focus()
					}
				})
				model.OnOpenMember = func(id string) error { return openProjectTeamMember(m, snap, id, false, state) }
				model.MaxHeight = min(float32(630), max(float32(0), h-82))
				projects.ProjectTeamBody(c, model)
			})
			if panel.OverlayShortcut(0, ui.KeyEscape) {
				state.SetFloating(false)
				open = false
				anchor.Focus()
			}
		})
		if open != state.FloatingOpen {
			state.SetFloating(open)
		}
	})
}

func openProjectTeamSidebar(m Model, model projects.ProjectTeamModel) error {
	if m.Store == nil || m.Sizes == nil || m.Sizes.Workspace == nil || m.Projects.ObserverHost == "" || m.Store.SelectedSessionID() != model.Project.CoordinatorSessionID {
		return errors.New(paneLabel("暂时无法打开团队侧栏，请重试。", "The Team Sidebar cannot open right now. Try again."))
	}
	w := m.Sizes.Workspace
	var target *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Kind == "project-team" && n.ProjectTeam != nil && n.ProjectTeam.ProjectID == model.Project.ID && n.ProjectTeam.CoordinatorSessionID == model.Project.CoordinatorSessionID && n.ProjectTeam.Host == m.Projects.ObserverHost {
			target = n
		}
	})
	if target == nil {
		target = w.leaf("project-team")
		target.ProjectTeam = &WorkspaceProjectTeamState{model.Project.ID, model.Project.CoordinatorSessionID, m.Projects.ObserverHost}
		source := w.Root.find(w.Active)
		w.placeDockResource(target, source, false)
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
		*m.RightPaneTab = "project-team"
	}
	if !m.RightPaneOpen && m.OnToggleTools != nil {
		m.OnToggleTools()
	}
	return nil
}

func workspaceProjectTeam(c *ui.Context, m Model) {
	n := m.workspacePane
	if n == nil || n.ProjectTeam == nil || m.Store == nil || m.Projects.ProjectTeam == nil || n.ProjectTeam.Host != m.Projects.ObserverHost || n.ProjectTeam.CoordinatorSessionID != m.Store.SelectedSessionID() {
		ui.Text(c, paneLabel("请从来源项目的团队按钮重新打开。", "Reopen this Team from its source Project.")).Padding(12)
		return
	}
	model := m.Projects.ProjectTeam(n.ProjectTeam.ProjectID)
	if model.State == nil || model.Project.CoordinatorSessionID != n.ProjectTeam.CoordinatorSessionID {
		ui.Text(c, paneLabel("团队暂不可用，请从来源项目重试。", "Team is unavailable. Retry from its source Project.")).Padding(12)
		return
	}
	model.ThemeID, model.SystemDark = m.ThemeID, m.SystemDark
	snap := m.Store.Snapshot()
	if child := m.Sizes.Workspace.currentMainChild(m); child != nil {
		model.SessionID = child.SubagentChat.Address.ChildSessionID
	}
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Padding(12).Gap(10).Children(func() {
		if ui.Button(c, paneLabel("回到浮层", "Return to popover")).AlignSelf(ui.Start).Clicked() {
			state := model.State
			w := m.Sizes.Workspace
			docked := n.Float == nil
			w.pending = func() {
				if !w.close(n.ID) {
					return
				}
				state.SetFloating(true)
				onlyGuide := true
				w.Root.eachResource(func(tab *WorkspacePane) { onlyGuide = onlyGuide && tab.Kind == "guide" })
				if docked && onlyGuide && m.RightPaneOpen && m.OnToggleTools != nil {
					m.OnToggleTools()
				}
			}
		}
		model.OnOpenMember = func(id string) error { return openProjectTeamMember(m, snap, id, true, model.State) }
		projects.ProjectTeamBody(c, model)
	})
}

func openProjectTeamMember(m Model, snap session.ViewState, id string, sidebar bool, state *projects.TeamViewState) error {
	return openTeamMember(m, snap, id, sidebar, func(message string) { state.NavigationError, state.FailedMember = message, id })
}

func openTeamMember(m Model, snap session.ViewState, id string, sidebar bool, fail func(string)) error {
	if m.Store == nil || m.Store.SelectedSessionID() != snap.SelectedID || m.Store.PlanHistoryBinding() != snap.PlanHistoryBinding || m.Sizes == nil || m.Sizes.Workspace == nil {
		return errors.New(paneLabel("会话已切换，请重新打开团队。", "The conversation changed. Reopen the Team."))
	}
	w := m.Sizes.Workspace
	if id == snap.SelectedID {
		w.closeMainChild()
		return nil
	}
	navigation := m.Store.SyncAgentNavigation(snap.PlanHistoryBinding)
	for _, row := range navigation.Rows {
		if row.Address.ParentSessionID != snap.SelectedID || row.Address.ChildSessionID != id || !row.CanOpen() {
			continue
		}
		local := m
		local.AgentNavigationStore = m.Store
		if !sidebar {
			subagentConversationOpener(local)(navigation.Token, row.Address)
			if child := w.currentMainChild(m); child != nil && child.SubagentChat.Address == row.Address {
				return nil
			}
			break
		}
		root, title, current := m.Store.AgentChatTarget(navigation.Token, row.Address)
		if !current || m.workspacePane == nil {
			break
		}
		sourceID := m.workspacePane.ID
		w.pending = func() {
			if _, _, valid := m.Store.AgentChatTarget(navigation.Token, row.Address); !valid || w.Root.find(sourceID) == nil {
				fail(paneLabel("成员来源已更新，请刷新后重试。", "The member source changed. Refresh and try again."))
				return
			}
			var existing *WorkspacePane
			w.Root.eachResource(func(n *WorkspacePane) {
				if n.Kind == "subagentchat" && n.SubagentChat != nil && n.SubagentChat.Root == root && n.SubagentChat.Address == row.Address {
					existing = n
				}
			})
			if existing != nil {
				w.activateDockTab(existing.ID)
				w.focusPending = existing.ID
				if existing.Float != nil {
					w.MinimizeWorkspaceFloat(existing.ID, false)
				}
				return
			}
			target := w.leaf("subagentchat")
			target.SubagentChat = &SubagentChatPaneState{Root: root, Address: row.Address, Title: title, Lineage: m.Store.AgentChatRestoreLineage(navigation.Token, row.Address)}
			w.placeDockResource(target, w.Root.find(sourceID), false)
			local := w.agentPaneModel(m, target)
			if !local.AgentNavigationStore.OpenAgentChatFromCatalog(m.Store, navigation.Token, root, row.Address, title) {
				w.disposeWorkspaceResource(target.ID)
				fail(paneLabel("成员目录尚未就绪，请刷新后重试。", "The member catalog is not ready. Refresh and try again."))
				return
			}
			w.activateDockTab(target.ID)
			w.focusPending = target.ID
		}
		return nil
	}
	return errors.New(paneLabel("成员目录尚未就绪，请刷新后重试。", "The member catalog is not ready. Refresh and try again."))
}
