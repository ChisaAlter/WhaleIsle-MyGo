package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
)

type WhaleSettingsModel struct {
	State                                        assiststore.WhaleSettingsSnapshot
	OnRefresh                                    func()
	OnEdit                                       func(field, value string)
	OnRestore                                    func(field string)
	OnSave                                       func(action string)
	OnReviewClear, OnCancelClear, OnConfirmClear func()
}

func whaleSettingsBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeWhale || m.WhaleSettings.OnRefresh == nil {
		return
	}
	w := m.WhaleSettings
	s := w.State
	blocked := s.Busy || m.PluginRPCBusy || m.Busy || !m.HarnessReady || !m.WhaleEnabled || s.HostKey == ""
	ui.Column(c.Key("whale-settings")).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "鲸鱼娘 · 人设与长期记忆 / Whale settings").FontSize(13).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, "读取 / 刷新 Host 内容 · Refresh").Disabled(blocked).Clicked() {
				w.OnRefresh()
			}
		})
		ui.Text(c, "读取、保存只作用于当前 Host 的鲸鱼娘；刷新保留未保存草稿，不会调用模型。").FontSize(11).TextColor(muted)
		if s.Busy {
			ui.Text(c, "正在等待 Host… / Waiting for Host…").FontSize(12).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "请先读取当前 Host 内容。失败或结果不确定时，重新读取后才可提交。 / Refresh before editing or retrying.").FontSize(11).TextColor(muted)
		}
		if s.Server.HomeDir == "" {
			return
		}
		ui.Text(c, s.Server.HomeDir+" · "+s.Server.SessionID).FontSize(11).TextColor(muted).Selectable()
		disabled := blocked || !s.Loaded
		ui.Column(c.Key("whale-memory")).Background(field).Border(1, border).Radius(8).Padding(12).Gap(7).Children(func() {
			ui.Text(c, "长期记忆 / Long-term memory").FontSize(13).Bold().TextColor(fg)
			ui.Text(c, "MEMORY.md 随鲸鱼娘会话注入。清空会写入 Host 的默认记忆模板。 / MEMORY.md is injected into Whale's session.").FontSize(11).TextColor(muted)
			memory := s.Memory
			ui.TextArea(c.Key("whale-memory-editor"), &memory).Label("Whale long-term memory").FillWidth().Lines(6, 14).Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("memory", memory)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 40000 UTF-16 · %s", pluginrpc.WhaleTextLength(memory), whaleDraftLabel(memory != s.Server.Memory))).FontSize(11).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && ui.Button(c, "保存记忆 / Save memory").Disabled(disabled || pluginrpc.WhaleTextLength(memory) > 40000).Clicked() {
					w.OnSave("memory")
				}
				if w.OnRestore != nil && ui.Button(c, "还原记忆草稿 / Restore memory draft").Disabled(disabled || memory == s.Server.Memory).Clicked() {
					w.OnRestore("memory")
				}
				if w.OnReviewClear != nil && ui.Button(c, "清空记忆… / Clear memory…").Disabled(disabled).Clicked() {
					w.OnReviewClear()
				}
			})
			if s.ConfirmClear {
				ui.Column(c.Key("whale-memory-clear-review")).Border(1, accent).Radius(8).Padding(10).Gap(7).Children(func() {
					ui.Text(c, "确定清空她的长期记忆？此操作不可撤销。 / Clear long-term memory? This cannot be undone.").FontSize(12).TextColor(accent)
					ui.Text(c, "确认后会替换当前 Host 的 MEMORY.md；取消保留当前草稿。 / Cancel keeps this draft.").FontSize(11).TextColor(muted)
					ui.Row(c).Gap(8).Children(func() {
						if w.OnCancelClear != nil && ui.Button(c, "取消清空 / Cancel clear").Disabled(blocked).Clicked() {
							w.OnCancelClear()
						}
						if w.OnConfirmClear != nil && ui.Button(c, "确认清空长期记忆 / Confirm clear memory").Disabled(disabled).Clicked() {
							w.OnConfirmClear()
						}
					})
				})
			}
		})
		ui.Column(c.Key("whale-persona")).Background(field).Border(1, border).Radius(8).Padding(12).Gap(7).Children(func() {
			ui.Text(c, "额外人设 / Extra persona").FontSize(13).Bold().TextColor(fg)
			ui.Text(c, "追加到人格末尾；留空即不用。 / Appended to her persona; leave blank to skip.").FontSize(11).TextColor(muted)
			persona := s.PersonaText
			ui.TextArea(c.Key("whale-persona-editor"), &persona).Label("Whale extra persona").FillWidth().Lines(3, 8).Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("persona", persona)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 4000 UTF-16 · %s", pluginrpc.WhaleTextLength(persona), whaleDraftLabel(persona != s.Server.PersonaText))).FontSize(11).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && ui.Button(c, "保存人设 / Save persona").Disabled(disabled || pluginrpc.WhaleTextLength(persona) > 4000).Clicked() {
					w.OnSave("persona")
				}
				if w.OnRestore != nil && ui.Button(c, "还原人设草稿 / Restore persona draft").Disabled(disabled || persona == s.Server.PersonaText).Clicked() {
					w.OnRestore("persona")
				}
			})
		})
		ui.Column(c.Key("whale-identity")).Gap(7).Children(func() {
			ui.Text(c, "名字与称呼 / Name and title").FontSize(13).Bold().TextColor(fg)
			name, title := s.Name, s.UserTitle
			ui.Text(c, "名字 / Name").FontSize(11).TextColor(muted)
			ui.TextInput(c.Key("whale-name"), &name).Label("Whale name").FillWidth().Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("name", name)
				}
			})
			ui.Text(c, "对你的称呼 / How she calls you").FontSize(11).TextColor(muted)
			ui.TextInput(c.Key("whale-user-title"), &title).Label("Whale user title").FillWidth().Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("title", title)
				}
			})
			ui.Text(c, "名字和称呼各限 64 UTF-16 字符；空名字由 Host 恢复为「鲸鱼娘」。").FontSize(11).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && ui.Button(c, "保存名字与称呼 / Save name and title").Disabled(disabled || pluginrpc.WhaleTextLength(name) > 64 || pluginrpc.WhaleTextLength(title) > 64).Clicked() {
					w.OnSave("identity")
				}
				if w.OnRestore != nil && ui.Button(c, "还原名字草稿 / Restore name draft").Disabled(disabled).Clicked() {
					w.OnRestore("identity")
				}
			})
		})
	})
}
func whaleDraftLabel(dirty bool) string {
	if dirty {
		return "未保存草稿 / Unsaved draft"
	}
	return "与 Host 一致 / Matches Host"
}
