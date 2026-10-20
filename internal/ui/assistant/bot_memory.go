package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
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
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantBotLongTermMemory)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "当前 Host、所选 Bot 的 bot/user 两轨。读取和保存不会启动 Bot；刷新保留同身份草稿，不会自动覆盖或重试。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "读取 / 刷新 Bot 记忆 · Refresh memory").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeEditingOrRetrying)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Server.BotID == "" {
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s", m.HostLabel, s.BotName, s.Scope.BotID, s.SessionID)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadOnlyOrWriteAuthorityUnavailable)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
		}
		ui.Row(c).Gap(8).Children(func() {
			for _, track := range []string{"bot", "user"} {
				label := i18n.T(i18n.KeyMiscAssistantBot)
				if track == "user" {
					label = i18n.T(i18n.KeyMiscAssistantUser)
				}
				if ui.Button(c, label).Disabled(blocked || s.Draft != nil || s.Track == track).Clicked() && m.OnTrack != nil {
					m.OnTrack(track)
				}
			}
		})
		active := s.Server.Track(s.Track)
		disabled := blocked || !s.Loaded || !s.Writable || s.Draft != nil
		ui.Text(c, fmt.Sprintf("Track: %s · %d / %d UTF-16 · Memory revision: %s", s.Track, pluginrpc.WhaleTextLength(active.Text), s.Server.Limit(s.Track), active.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantHostRawContent)+active.Text).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if len(active.Entries) == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoParsedEntries)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for index, entry := range active.Entries {
			ui.Column(c.Key(fmt.Sprintf("bot-memory-entry-%s-%d", s.Track, index))).Gap(4).Children(func() {
				ui.Text(c, entry).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Row(c).Gap(7).Children(func() {
					if m.OnEdit != nil && ui.Button(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantEditEntry), index+1)).Disabled(disabled).Clicked() {
						m.OnEdit(index)
					}
					if m.OnDelete != nil && ui.Button(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantDeleteEntry), index+1)).Disabled(disabled).Clicked() {
						m.OnDelete(index)
					}
				})
			})
		}
		if m.OnEdit != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAddMemoryEntry)).Disabled(disabled).Clicked() {
			m.OnEdit(-1)
		}
		if s.Draft != nil {
			d := s.Draft
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantDraft2)+d.Track+" · "+d.BaseRevision).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if !d.Remove {
				input := d.Input
				ui.TextArea(c.Key("bot-memory-entry-editor"), &input).Label("Bot memory entry").FillWidth().Lines(3, 9).Disabled(blocked).OnChange(func() {
					if m.OnChange != nil {
						m.OnChange(input)
					}
				})
			} else {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantEntryToRemove)+d.Input).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
			}
			ui.Row(c).Gap(8).Children(func() {
				if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewMemorySave)).Disabled(blocked || !s.Loaded || !s.Writable).Clicked() {
					m.OnReview()
				}
				if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardDraft)).Disabled(blocked).Clicked() {
					m.OnDiscard()
				}
			})
		}
		if s.Review != nil {
			r := s.Review
			ui.Column(c.Key("bot-memory-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewMemoryReplacement)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s\nTrack: %s · Memory revision: %s", m.HostLabel, r.Command.BotID, r.Command.SessionID, r.Command.Track, r.Command.MemoryRevision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "这会替换当前轨的整轨文件，影响此 Bot 后续使用的记忆；不会修改另一轨或启动 Bot，也不会撤销当前会话已读取的快照。空内容会删除该轨文件，移除内容不能自动恢复。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "沿用原版条目序列化：去掉每项首尾空白，使用 - 前缀重新组成全文；原始非条目行和格式也将被以下内容替换。请核对完整前后内容。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantBeforeContent)+r.Before+i18n.T(i18n.KeyMiscAssistantAfterContent)+r.Command.Text).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Row(c).Gap(8).Children(func() {
					if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToMemoryEdit)).Disabled(blocked).Clicked() {
						m.OnDismiss()
					}
					if m.OnConfirm != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmTrackReplacement)).Disabled(blocked || !s.Loaded || !s.Writable).Clicked() {
						m.OnConfirm()
					}
				})
			})
		}
	})
}
