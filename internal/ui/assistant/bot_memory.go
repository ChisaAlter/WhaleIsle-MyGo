package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

type BotMemoryModel struct {
	State                                                dshbotstore.MemorySnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnTrack, OnChange                                    func(string)
	OnEdit, OnDelete                                     func(int)
}

func botMemoryBlock(c *ui.Context, m BotMemoryModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-memory")).Gap(7).Children(func() {
		ui.Text(c, "Bot 长期记忆 / Bot long-term memory").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "当前 Host、所选 Bot 的 bot/user 两轨。读取和保存不会启动 Bot；刷新保留同身份草稿，不会自动覆盖或重试。").FontSize(11).TextColor(muted)
		if ui.Button(c, "读取 / 刷新 Bot 记忆 · Refresh memory").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted)
		}
		if !s.Loaded {
			ui.Text(c, "请读取 Host 当前记忆；下方保留内容不是新的权威回执 / Refresh before editing or retrying").FontSize(11).TextColor(muted)
		}
		if s.Server.BotID == "" {
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s", m.HostLabel, s.BotName, s.Scope.BotID, s.SessionID)).FontSize(11).TextColor(muted).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 写权限未验证或目录只读；可查看，不开放编辑 / Read-only or write authority unavailable").FontSize(11).TextColor(accent)
		}
		ui.Row(c).Gap(8).Children(func() {
			for _, track := range []string{"bot", "user"} {
				label := "Bot 经验 / bot"
				if track == "user" {
					label = "此 Bot 的用户信息 / user"
				}
				if ui.Button(c, label).Disabled(blocked || s.Draft != nil || s.Track == track).Clicked() && m.OnTrack != nil {
					m.OnTrack(track)
				}
			}
		})
		active := s.Server.Track(s.Track)
		disabled := blocked || !s.Loaded || !s.Writable || s.Draft != nil
		ui.Text(c, fmt.Sprintf("Track: %s · %d / %d UTF-16 · Memory revision: %s", s.Track, pluginrpc.WhaleTextLength(active.Text), s.Server.Limit(s.Track), active.Revision)).FontSize(11).TextColor(muted).Selectable()
		ui.Text(c, "Host 原始内容 / Host raw content:\n"+active.Text).FontSize(12).TextColor(fg).Selectable()
		if len(active.Entries) == 0 {
			ui.Text(c, "暂无可编辑条目 / No parsed entries").FontSize(11).TextColor(muted)
		}
		for index, entry := range active.Entries {
			ui.Column(c.Key(fmt.Sprintf("bot-memory-entry-%s-%d", s.Track, index))).Gap(4).Children(func() {
				ui.Text(c, entry).FontSize(12).TextColor(fg).Selectable()
				ui.Row(c).Gap(7).Children(func() {
					if m.OnEdit != nil && ui.Button(c, fmt.Sprintf("编辑记忆条目 %d / Edit entry %d", index+1, index+1)).Disabled(disabled).Clicked() {
						m.OnEdit(index)
					}
					if m.OnDelete != nil && ui.Button(c, fmt.Sprintf("删除记忆条目 %d… / Delete entry %d…", index+1, index+1)).Disabled(disabled).Clicked() {
						m.OnDelete(index)
					}
				})
			})
		}
		if m.OnEdit != nil && ui.Button(c, "添加记忆条目 / Add memory entry").Disabled(disabled).Clicked() {
			m.OnEdit(-1)
		}
		if s.Draft != nil {
			d := s.Draft
			ui.Text(c, "保留草稿 / Draft · "+d.Track+" · "+d.BaseRevision).FontSize(11).TextColor(muted)
			if !d.Remove {
				input := d.Input
				ui.TextArea(c.Key("bot-memory-entry-editor"), &input).Label("Bot memory entry").FillWidth().Lines(3, 9).Disabled(blocked).OnChange(func() {
					if m.OnChange != nil {
						m.OnChange(input)
					}
				})
			} else {
				ui.Text(c, "拟删除条目 / Entry to remove: "+d.Input).FontSize(12).TextColor(accent).Selectable()
			}
			ui.Row(c).Gap(8).Children(func() {
				if m.OnReview != nil && ui.Button(c, "审阅保存记忆 / Review memory save").Disabled(blocked || !s.Loaded || !s.Writable).Clicked() {
					m.OnReview()
				}
				if m.OnDiscard != nil && ui.Button(c, "放弃草稿并采用已读取内容 / Discard draft").Disabled(blocked).Clicked() {
					m.OnDiscard()
				}
			})
		}
		if s.Review != nil {
			r := s.Review
			ui.Column(c.Key("bot-memory-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, "确认替换记忆 / Review memory replacement").FontSize(13).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s\nTrack: %s · Memory revision: %s", m.HostLabel, r.Command.BotID, r.Command.SessionID, r.Command.Track, r.Command.MemoryRevision)).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, "这会替换当前轨的整轨文件，影响此 Bot 后续使用的记忆；不会修改另一轨或启动 Bot，也不会撤销当前会话已读取的快照。空内容会删除该轨文件，移除内容不能自动恢复。").FontSize(12).TextColor(accent)
				ui.Text(c, "沿用原版条目序列化：去掉每项首尾空白，使用 - 前缀重新组成全文；原始非条目行和格式也将被以下内容替换。请核对完整前后内容。").FontSize(11).TextColor(muted)
				ui.Text(c, "替换前 / Before:\n"+r.Before+"\n替换后 / After:\n"+r.Command.Text).FontSize(12).TextColor(fg).Selectable()
				ui.Row(c).Gap(8).Children(func() {
					if m.OnDismiss != nil && ui.Button(c, "返回记忆编辑 / Back to memory edit").Disabled(blocked).Clicked() {
						m.OnDismiss()
					}
					if m.OnConfirm != nil && ui.Button(c, "确认替换此轨记忆 / Confirm track replacement").Disabled(blocked || !s.Loaded || !s.Writable).Clicked() {
						m.OnConfirm()
					}
				})
			})
		}
	})
}
