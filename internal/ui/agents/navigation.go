package agents

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func agentText(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}

// navigationView is deliberately a read-only history seat beside the main chat.
// Only user buttons open addressed follow; drawing/restoring it loads catalog.
func navigationView(c *ui.Context, m Model) {
	snap := m.Snapshot
	v := m.Store.SyncAgentNavigation(snap.PlanHistoryBinding)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	root := ui.Column(c.Key(fmt.Sprintf("agents-navigation-root:%p:%d:%s:%s:%s", m.Store, v.Token, v.Address.ParentSessionID, v.Address.ChildSessionID, v.Address.Mode))).Fill().MinHeight(0).Background(chrome.Color(colors.Field))
	childFileDropTarget(c, root, m.Store, v)
	root.Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			title := i18n.T(i18n.KeyAgentsTitle)
			if m.ChatOnly {
				title = v.Title
			}
			ui.Text(c, title).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			var running []session.AgentSessionAddress
			if m.ChatOnly {
				for _, item := range snap.Items {
					if item.SessionID == v.Address.ChildSessionID && item.Running {
						running = append(running, v.Address)
					}
				}
			}
			for _, row := range v.Rows {
				if !m.ChatOnly && row.CanOpen() && row.Running {
					running = append(running, row.Address)
				}
			}
			interruptLabel := i18n.T(i18n.KeyAgentsInterruptAll)
			if m.ChatOnly {
				interruptLabel = i18n.T(i18n.KeyAgentsInterrupt)
			}
			if len(running) > 0 && ui.Button(c, interruptLabel).FontSize(theme.InterfaceSize(11)).Disabled(v.ActionBusy || (v.RetainedReadOnly && !(m.ChatOnly && v.RetainedPromptAllowed))).Clicked() {
				if m.ChatOnly && v.RetainedReadOnly {
					m.Store.InterruptRetainedAgentExplicit(v.Token, v.Address)
				} else {
					m.Store.InterruptAgentNavigation(v.Token, running)
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyAgentsClose)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.Store.CloseAgentNavigation()
				m.OnClose()
			}
		})
		if m.Manager != nil {
			presetManagerView(c, m.Manager, fg, muted)
		}
		if v.SessionID == "" {
			ui.Text(c, agentText("等待当前会话连接…", "Waiting for the current session…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			return
		}
		if !m.ChatOnly {
			paintAgentAncestors(c, m.Store.AgentNavigationAncestors(v.Token), func(depth int) { m.Store.JumpAgentNavigation(v.Token, depth) })
		}
		ui.Row(c).FillWidth().Padding(8, 12).Gap(6).Children(func() {
			if !m.ChatOnly && v.Depth > 0 && ui.Button(c, agentText("返回上一级", "Back to parent")).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.Store.BackAgentNavigation(v.Token)
			}
			ui.Text(c, v.Title).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg).Grow(1)
			caption := agentText("刷新目录", "Refresh directory")
			if v.Depth > 0 {
				caption = agentText("回到最新 / 重连", "Latest / reconnect")
			}
			if ui.Button(c, caption).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy).Clicked() {
				m.Store.RefreshAgentNavigation(v.Token)
			}
		})
		ui.Text(c, v.SessionID).FontSize(theme.InterfaceSize(10)).TextColor(muted).Selectable()
		if v.Depth > 0 {
			ui.Text(c, v.Address.ParentSessionID+" → "+v.Address.ChildSessionID+" · "+v.Address.Mode).FontSize(theme.InterfaceSize(10)).TextColor(muted).Selectable()
			if !m.MainConversation {
				ui.Text(c, agentText("子会话历史；主聊天仍保留原会话。", "Child history; the main chat retains its original session.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
		}
		if v.Error != "" {
			ui.Text(c, v.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg).Selectable()
		}
		if v.ActionStatus != "" {
			ui.Text(c, v.ActionStatus).FontSize(theme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if v.Depth > 0 {
			status := agentText("实时输出已暂停；回到最新恢复。", "Live output paused; choose Latest to resume.")
			if v.Process.Live {
				status = agentText("实时过程（只读观察）", "Live process (read-only observation)")
			}
			ui.Text(c, status).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if v.Busy {
			ui.Text(c, agentText("正在读取…", "Loading…")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Scroll(c.Key(fmt.Sprintf("agents-navigation:%s:%d", v.SessionID, v.Token))).FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Padding(10, 12).Gap(8).Children(func() {
				if !m.ChatOnly {
					catalogRowsView(c, m, v, muted)
				}
				if v.Depth > 0 {
					ui.Row(c).FillWidth().Gap(6).Children(func() {
						ui.Text(c, agentText("过程历史", "Process history")).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg).Grow(1)
						if ui.Button(c, agentText("较早记录", "Earlier records")).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy || !v.Process.HasMore).Clicked() {
							m.Store.EarlierAgentNavigation(v.Token)
						}
					})
					if v.Process.Truncated {
						ui.Text(c, agentText("当前窗口最多400条；可继续读取较早记录。", "Up to 400 records per window; earlier history remains pageable.")).FontSize(theme.InterfaceSize(10)).TextColor(muted)
					}
					body := v.Body
					if body == "" && !v.Busy {
						body = agentText("尚无可显示记录。", "No visible records yet.")
					}
					resources := childResourceOptions(m, v)
					historyStyle := chat.DefaultToolCardStyle(fg, muted, chrome.Color(colors.Border), chrome.Color(colors.Field), chrome.Color(colors.Accent))
					historyStyle.Colors = colors
					chat.ReadOnlyHistoryView(c, chat.ReadOnlyHistoryScope(snap.PlanHistoryBinding, v.Address, v.Token), v.Messages, body, historyStyle, resources)
					childDeliveriesView(c, m, v, chat.DefaultToolCardStyle(fg, muted, chrome.Color(colors.Border), chrome.Color(colors.Field), chrome.Color(colors.Accent)))
					childInteractionsView(c, m, v)
					// Preserve the existing dedicated continuation route. Never send through
					// the main composer or pretend one-shot history is an ordinary Session.
					if reason := childReadOnlyReason(v); reason != "" {
						ui.Text(c, reason).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					}
					if v.Address.Mode == "continuable" && m.PromptDraft != nil {
						childEditor(c, m, v)
					}
				}
			})
		})
	})
}
