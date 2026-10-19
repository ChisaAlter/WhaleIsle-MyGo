package pet

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

func quickChatName(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

// This is native MyGo content, not a page with general IPC/model credentials.
// Enter sends; Shift+Enter and IME composition remain with the editor.
func (q *QuickChat) view(c *ui.Context) {
	fonttheme.ApplyInterface(c)
	direct := q.options.Direct != nil
	active := q.win != nil && q.check()
	theme := c.Theme()
	ui.Column(c).Fill().Padding(14).Gap(10).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, quickChatName(q.state.Name, "鲸鱼娘")+" · 聊聊").Bold().FontSize(fonttheme.InterfaceSize(17)).Grow(1)
			if !direct && ui.Button(c, "↗ 主会话 / Open").Disabled(!active || q.state.SessionID == "" || q.jumping.busy()).Clicked() {
				q.openMain()
			}
			if ui.Button(c, "×").Label("关闭聊天卡片 / Close quick chat").Tooltip("关闭只取消本地等待，服务端可能继续 / Closing only cancels the local wait").Clicked() {
				q.owner.CloseQuickChat()
			}
		})
		if direct {
			ui.Text(c, "联网临时聊天 · deepseek-chat · 使用桌面 API 设置；仅保留本窗口最近 6 轮，不写入共同会话 / Temporary online API chat").FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.Warning)
		} else {
			ui.Text(c, "与主窗口共享模型、上下文和持久记录 / One shared conversation").FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.TextMuted)
		}
		if q.state.SessionID != "" {
			ui.Text(c, q.state.SessionID).Label("宠物会话 ID / Pet session ID").SingleLine().FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted).Selectable()
		}
		if !direct {
			q.modelPicker(c, active)
		}
		if len(q.state.History) == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				line := "正在读取共同记录… / Loading shared history…"
				if direct {
					line = "输入后点联网发送才会请求 API / Click Send to API to send"
				}
				if q.state.SessionID != "" {
					line = "还没有文字记录 / No messages yet"
				}
				if q.stale {
					line = "连接不可用；草稿仍保留 / Connection unavailable"
				}
				ui.Text(c, line).TextColor(theme.TextMuted).FontSize(fonttheme.InterfaceSize(12))
			})
		} else {
			historyLabel := "共同会话记录 / Shared history"
			if direct {
				historyLabel = "临时 API 聊天记录 / Temporary API history"
			}
			ui.List(c, &q.list, len(q.state.History)).Label(historyLabel).Grow(1).FillWidth().MinHeight(80).Rows(func(row ui.ListRow) {
				message := q.state.History[row.Index]
				label, color := q.state.Name, theme.Text
				switch message.Role {
				case "user":
					label = "你 / You"
				case "her":
					label = quickChatName(label, "鲸鱼娘")
				case "err":
					label, color = "状态 / Status", theme.Danger
				default:
					label = "会话 / Session"
				}
				ui.Column(row.Context).Padding(8).Gap(3).FillWidth().Children(func() {
					ui.Text(row.Context, label).FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted)
					ui.Text(row.Context, message.Text).Font(fonttheme.CurrentPreferences().Chat.Family).FontSize(fonttheme.CurrentPreferences().Chat.Size).TextColor(color).Selectable()
				})
			})
		}
		if q.readNote != "" {
			ui.Text(c, q.readNote).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.Warning)
		}
		ui.Text(c, q.status).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.TextMuted)
		input := ui.TextArea(c, &q.draft).Font(fonttheme.CurrentPreferences().Composer.Family).FontSize(fonttheme.CurrentPreferences().Composer.Size).Label("聊天草稿 / Quick chat draft").FillWidth().Lines(2, 4).Placeholder("说点什么… Enter 发送，Shift+Enter 换行 / Enter sends; Shift+Enter newline")
		// Read the IME composition state during the build: input callbacks run
		// outside the build pass, where an Element query panics ("element from
		// pass N used in pass N"). The next frame refreshes it.
		composing := input.Composing()
		input.HandleInput(func(ev ui.InputEvent) bool {
			if !quickChatSendKey(ev, composing) {
				return false
			}
			if !ev.Repeat && active && !q.sending.busy() && !q.picking.busy() && !q.stopping.busy() && strings.TrimSpace(q.draft) != "" {
				q.send()
			}
			return true
		})
		if !c.Root().FocusWithin() {
			input.AutoFocus()
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			sendLabel := "发送 / Send"
			if direct {
				sendLabel = "联网发送 / Send to API"
			}
			if ui.Button(c, sendLabel).Disabled(!active || (!direct && q.state.SessionID == "") || q.sending.busy() || q.picking.busy() || q.stopping.busy() || strings.TrimSpace(q.draft) == "").Clicked() {
				q.send()
			}
			if q.sending.busy() && ui.Button(c, "取消等待 / Cancel wait").Disabled(!active).Clicked() {
				q.cancelWait()
			}
			if !direct && ui.Button(c, "停止当前轮 / Stop turn").Disabled(!active || q.state.SessionID == "" || q.stopping.busy() || q.picking.busy()).Clicked() {
				q.stopTurn()
			}
			if !direct && ui.Button(c, "刷新 / Refresh").Disabled(!active || q.history.busy()).Clicked() {
				q.refreshHistory()
				q.refreshCatalog()
			}
		})
		if direct {
			ui.Text(c, "取消不保证撤回 API 请求。关闭清除临时记录；Host 恢复后重新点聊聊进入共同会话。").FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted)
		} else {
			ui.Text(c, "取消等待不会撤回消息。停止仅作用于此宠物会话的当前轮；排队消息请到主会话管理。").FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted)
		}
	})
	if c.Shortcut(0, ui.KeyEscape) {
		q.owner.CloseQuickChat()
	}
}

func (q *QuickChat) modelPicker(c *ui.Context, active bool) {
	blocked := !active || q.sending.busy() || q.picking.busy() || q.stopping.busy()
	selected := q.state.Model
	modelLabel := "模型 / Model"
	if selected != nil {
		modelLabel = selected.Provider + " / " + selected.Model
	}
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		ui.MenuButton(c, modelLabel, func(menu *ui.Menu) {
			for _, group := range q.state.Groups {
				menu.Submenu(quickChatName(group.Name, group.ID)+" ["+group.ID+"]", func(sub *ui.Menu) {
					for _, model := range group.Models {
						choice := quickChatModel{Provider: group.ID, Model: model.ID, ReasoningEffort: model.DefaultEffort}
						if !q.state.allows(choice) {
							choice.ReasoningEffort = ""
						}
						picked := selected != nil && selected.Provider == group.ID && selected.Model == model.ID
						if sub.Item(quickChatName(model.Name, model.ID)+" ["+model.ID+"]").Checked(picked).Disabled(blocked).Chosen() && !picked {
							q.choose(choice)
						}
					}
				})
			}
		}).Label("选择共同会话模型 / Select shared model").MaxWidthPercent(100).MinWidth(0).Disabled(blocked || len(q.state.Groups) == 0)
		if selected != nil {
			for _, group := range q.state.Groups {
				if group.ID != selected.Provider {
					continue
				}
				for _, model := range group.Models {
					if model.ID != selected.Model || len(model.Efforts) == 0 {
						continue
					}
					label := "思考 / Reasoning: " + quickChatName(selected.ReasoningEffort, "默认 / Default")
					ui.MenuButton(c, label, func(menu *ui.Menu) {
						if menu.Item("默认 / Default").Checked(selected.ReasoningEffort == "").Disabled(blocked).Chosen() {
							q.choose(quickChatModel{Provider: selected.Provider, Model: selected.Model})
						}
						for _, effort := range model.Efforts {
							if menu.Item(fmt.Sprintf("%s [%s]", quickChatName(effort.Name, effort.ID), effort.ID)).Checked(selected.ReasoningEffort == effort.ID).Disabled(blocked).Chosen() {
								q.choose(quickChatModel{Provider: selected.Provider, Model: selected.Model, ReasoningEffort: effort.ID})
							}
						}
					}).Label("选择思考强度 / Select reasoning effort").Disabled(blocked)
				}
			}
		}
	})
	if q.catalogNote != "" {
		ui.Text(c, q.catalogNote).FontSize(fonttheme.InterfaceSize(10)).TextColor(c.Theme().TextMuted)
	}
}
