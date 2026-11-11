package frame

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

// The header and Sidebar share one owner-scoped view, but member navigation
// chooses its real destination at the entry that the user clicked.
func conversationTeamAction(c *ui.Context, m Model, snap session.ViewState, compact bool, ink ui.Color) {
	if projectConversationTeamAction(c, m, snap, compact, ink) {
		return
	}
	if m.OrdinaryTeam == nil && m.OnOpenAgents != nil {
		conversationHeaderButton(c, "conversation-agents", i18n.T(i18n.KeyFrameCaptionAgents), "UsersOutline", compact, ink, m.OnOpenAgents, 0)
		return
	}
	if snap.SelectedID == "" || m.OrdinaryTeam == nil {
		return
	}
	model := m.OrdinaryTeam()
	owner := ui.Row(c.Key("conversation-team:" + snap.SelectedID)).Shrink(0)
	state := model.State
	if state == nil {
		state = ui.Local(owner, "team-state", func() projects.OrdinaryTeamViewState { return projects.OrdinaryTeamViewState{} })
		model.State = state
	}
	owner.Children(func() {
		anchor := conversationHeaderButton(c, "conversation-agents", paneLabel("团队", "Team"), "UsersOutline", compact, ink, func() {
			if state.FloatingOpen && state.Pinned {
				state.FloatingOpen, state.Pinned, state.HoverDismissed = false, false, true
			} else {
				state.FloatingOpen, state.Pinned, state.HoverDismissed = true, true, false
			}
			state.HoverAt, state.LeaveAt = time.Time{}, time.Time{}
		}, 0)
		if anchor.Hovered() {
			state.LeaveAt = time.Time{}
			if !compact && !state.FloatingOpen && !state.HoverDismissed {
				if state.HoverAt.IsZero() {
					state.HoverAt = c.Now()
				}
				if c.Now().Sub(state.HoverAt) >= 150*time.Millisecond {
					state.FloatingOpen = true
				} else {
					c.After(150 * time.Millisecond)
				}
			}
		} else {
			state.HoverAt = time.Time{}
			state.HoverDismissed = false
		}
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		w, h := viewportDIP(c)
		width := min(float32(500), max(float32(0), w-32))
		visibleTasks := 0
		if model.Team != nil {
			for _, task := range model.Team.Tasks {
				if task.Status != "deleted" {
					visibleTasks++
				}
			}
		}
		if model.Team != nil && len(model.Team.Members) == 1 && visibleTasks == 0 {
			width = min(float32(320), max(float32(0), w-32))
		}
		open := state.FloatingOpen
		ui.PopoverBase(c.Key("conversation-team-popover"), anchor, &open, func(panel ui.Element) {
			headerPopupInset(c, panel, anchor, width, w, 16)
			panel = chrome.PopoverReveal(c, panel.Label(paneLabel("智能体团队", "Agent Team")).Width(width).MaxHeight(min(float32(680), max(float32(0), h-32))).MinHeight(0).Margin(5, 0, 0, 0).Padding(8, 2, 0, 2).Radius(12).Border(.5, chrome.Color(colors.Border)).Background(chrome.Color(colors.BG)).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .18)))
			panel.Children(func() {
				model.ThemeID, model.SystemDark = m.ThemeID, m.SystemDark
				model.Columns = 1
				if w > 560 && width > 320 {
					model.Columns = 2
				}
				model.MaxHeight = min(float32(664), max(float32(0), h-48))
				if state.NavigationError != "" {
					model.Error = state.NavigationError
				}
				if m.Sizes != nil && m.Sizes.Workspace != nil {
					if child := m.Sizes.Workspace.currentMainChild(m); child != nil {
						model.SessionID = child.SubagentChat.Address.ChildSessionID
					}
				}
				model.OnOpenMember = func(id string) {
					if err := openTeamMember(m, snap, id, false, func(message string) { state.NavigationError = message }); err != nil {
						state.NavigationError = err.Error()
						return
					}
					state.NavigationError = ""
					open = false
					state.FloatingOpen = false
				}
				projects.OrdinaryTeamView(c, model)
			})
			if panel.Hovered() || panel.FocusWithin() || anchor.Hovered() {
				state.LeaveAt = time.Time{}
			} else if !state.Pinned {
				if state.LeaveAt.IsZero() {
					state.LeaveAt = c.Now()
				}
				if c.Now().Sub(state.LeaveAt) >= 120*time.Millisecond {
					open = false
				} else {
					c.After(120 * time.Millisecond)
				}
			}
			if panel.OverlayShortcut(0, ui.KeyEscape) {
				open = false
				state.HoverDismissed = true
				anchor.Focus()
			}
		})
		state.FloatingOpen = open
		if !open {
			state.Pinned = false
		}
	})
}
