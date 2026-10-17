package agents

import (
	"fmt"
	"github.com/egoist/mygo"
	"path/filepath"
	"strings"

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
	ui.Column(c).Fill().MinHeight(0).Background(chrome.Color(colors.Field)).Children(func() {
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
			if len(running) > 0 && ui.Button(c, interruptLabel).FontSize(theme.InterfaceSize(11)).Disabled(v.ActionBusy).Clicked() {
				m.Store.InterruptAgentNavigation(v.Token, running)
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
					chat.ReadOnlyHistoryView(c, chat.ReadOnlyHistoryScope(snap.PlanHistoryBinding, v.Address, v.Token), v.Messages, body,
						chat.DefaultToolCardStyle(fg, muted, chrome.Color(colors.Border), chrome.Color(colors.Field), chrome.Color(colors.Accent)), resources)
					childDeliveriesView(c, m, v, chat.DefaultToolCardStyle(fg, muted, chrome.Color(colors.Border), chrome.Color(colors.Field), chrome.Color(colors.Accent)))
					childInteractionsView(c, m, v)
					// Preserve the existing dedicated continuation route. Never send through
					// the main composer or pretend one-shot history is an ordinary Session.
					if v.Address.Mode == "continuable" && m.PromptDraft != nil {
						ui.Text(c, agentText("向此子代理发送（专用入口）", "Send to this child (dedicated route)")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if ui.Button(c, agentText("添加图片", "Add images")).Disabled(v.Busy || v.ActionBusy).Clicked() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: agentText("向子代理添加图片", "Add images for child"), Multiple: true, Filters: []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"}}}})
							m.Store.AddAgentNavigationImages(v.Token, v.Address, paths, err)
						}
						for _, path := range v.PromptImages {
							ui.Row(c).Gap(6).Children(func() {
								ui.Text(c, filepath.Base(path)).TextColor(muted)
								if ui.Button(c, agentText("移除", "Remove")).Disabled(v.ActionBusy).Clicked() {
									m.Store.RemoveAgentNavigationImage(v.Token, v.Address, path)
								}
							})
						}
						ui.TextInput(c, m.PromptDraft).FillWidth().Placeholder(i18n.T(i18n.KeyAgentsPromptPlaceholder))
						if ui.Button(c, i18n.T(i18n.KeyAgentsPromptSend)).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy || v.ActionBusy || (strings.TrimSpace(*m.PromptDraft) == "" && len(v.PromptImages) == 0)).Clicked() {
							m.Store.PromptAgentNavigation(v.Token, v.Address, strings.TrimSpace(*m.PromptDraft))
						}
					}
				}
			})
		})
	})
}
