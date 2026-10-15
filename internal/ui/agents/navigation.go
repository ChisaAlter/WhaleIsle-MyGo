package agents

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
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
	ui.Column(c).Fill().MinHeight(0).Background(chrome.Color(colors.Field)).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAgentsTitle)).FontSize(14).Bold().TextColor(fg).Grow(1)
			var running []session.AgentSessionAddress
			for _, row := range v.Rows {
				if row.Running {
					running = append(running, row.Address)
				}
			}
			if len(running) > 0 && ui.Button(c, i18n.T(i18n.KeyAgentsInterruptAll)).FontSize(11).Disabled(v.ActionBusy).Clicked() {
				m.Store.InterruptAgentNavigation(v.Token, running)
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyAgentsClose)).FontSize(11).Clicked() {
				m.Store.CloseAgentNavigation()
				m.OnClose()
			}
		})
		if m.Manager != nil {
			presetManagerView(c, m.Manager, fg, muted)
		}
		if v.SessionID == "" {
			ui.Text(c, agentText("等待当前会话连接…", "Waiting for the current session…")).FontSize(12).TextColor(muted)
			return
		}
		ui.Row(c).FillWidth().Padding(8, 12).Gap(6).Children(func() {
			if v.Depth > 0 && ui.Button(c, agentText("返回上一级", "Back to parent")).FontSize(11).Clicked() {
				m.Store.BackAgentNavigation(v.Token)
			}
			ui.Text(c, v.Title).FontSize(13).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, agentText("刷新", "Refresh")).FontSize(11).Disabled(v.Busy).Clicked() {
				m.Store.RefreshAgentNavigation(v.Token)
			}
		})
		ui.Text(c, v.SessionID).FontSize(10).TextColor(muted).Selectable()
		if v.Depth > 0 {
			ui.Text(c, v.Address.ParentSessionID+" → "+v.Address.ChildSessionID+" · "+v.Address.Mode).FontSize(10).TextColor(muted).Selectable()
			ui.Text(c, agentText("子会话历史；主聊天仍保留原会话。", "Child history; the main chat retains its original session.")).FontSize(11).TextColor(muted)
		}
		if v.Error != "" {
			ui.Text(c, v.Error).FontSize(12).TextColor(fg).Selectable()
		}
		if v.ActionStatus != "" {
			ui.Text(c, v.ActionStatus).FontSize(11).TextColor(muted).Selectable()
		}
		if v.Busy {
			ui.Text(c, agentText("正在读取…", "Loading…")).FontSize(11).TextColor(muted)
		}
		ui.Scroll(c.Key(fmt.Sprintf("agents-navigation:%s:%d", v.SessionID, v.Token))).FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Padding(10, 12).Gap(8).Children(func() {
				ui.Text(c, agentText("当前会话的子代理", "Children of this session")).FontSize(11).TextColor(muted)
				ui.Text(c, agentText("点击打开历史；Host 可能激活已准备的会话。", "Click to open history; the Host may activate a prepared Session.")).FontSize(10).TextColor(muted)
				if len(v.Rows) == 0 && !v.Busy {
					ui.Text(c, i18n.T(i18n.KeyAgentsEmpty)).FontSize(12).TextColor(muted)
				}
				for _, row := range v.Rows {
					ui.Row(c).FillWidth().Gap(6).Children(func() {
						title := row.Label + " · " + row.Address.Mode
						if row.Running {
							title = "● " + title
						}
						if ui.Button(c, title).Grow(1).FontSize(12).Disabled(v.Busy).Clicked() {
							m.Store.OpenAgentNavigation(v.Token, row.Address)
						}
						if row.Running && ui.Button(c, i18n.T(i18n.KeyAgentsInterrupt)).FontSize(11).Disabled(v.ActionBusy).Clicked() {
							m.Store.InterruptAgentNavigation(v.Token, []session.AgentSessionAddress{row.Address})
						}
					})
				}
				if v.Depth > 0 {
					ui.Row(c).FillWidth().Gap(6).Children(func() {
						ui.Text(c, agentText("过程历史", "Process history")).FontSize(12).Bold().TextColor(fg).Grow(1)
						if ui.Button(c, agentText("较早记录", "Earlier records")).FontSize(11).Disabled(v.Busy || !v.Process.HasMore).Clicked() {
							m.Store.EarlierAgentNavigation(v.Token)
						}
					})
					if v.Process.Truncated {
						ui.Text(c, agentText("当前窗口最多400条；可继续读取较早记录。", "Up to 400 records per window; earlier history remains pageable.")).FontSize(10).TextColor(muted)
					}
					body := v.Body
					if body == "" && !v.Busy {
						body = agentText("尚无可显示记录。", "No visible records yet.")
					}
					ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
					// Preserve the existing dedicated continuation route. Never send through
					// the main composer or pretend one-shot history is an ordinary Session.
					if v.Address.Mode != "one-shot" && m.PromptDraft != nil {
						ui.Text(c, agentText("向此子代理发送（专用入口）", "Send to this child (dedicated route)")).FontSize(11).TextColor(muted)
						ui.TextInput(c, m.PromptDraft).FillWidth().Placeholder(i18n.T(i18n.KeyAgentsPromptPlaceholder))
						if ui.Button(c, i18n.T(i18n.KeyAgentsPromptSend)).FontSize(11).Disabled(v.Busy || v.ActionBusy || strings.TrimSpace(*m.PromptDraft) == "").Clicked() {
							m.Store.PromptAgentNavigation(v.Token, v.Address, strings.TrimSpace(*m.PromptDraft))
						}
					}
				}
			})
		})
	})
}
