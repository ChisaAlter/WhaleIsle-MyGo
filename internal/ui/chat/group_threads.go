package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

func groupThreadsView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	threads := session.ProjectGroupThreads(m.Snapshot.Messages, m.GroupMembers)
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		ConnBannerView(c, m.ConnectionBanner, muted, chrome.Color(colors.Field), border, m.OnReconnect)
		ui.Row(c).Padding(8, 12).Gap(8).Wrap().Children(func() {
			ui.Text(c, "群聊讨论 / Group discussion").Bold().TextColor(fg)
			for _, member := range m.GroupMembers {
				name := member.Title
				if name == "" {
					name = member.Name
				}
				if member.RuntimeStatus != "" {
					name += " · " + member.RuntimeStatus
				}
				if member.RuntimeError != "" {
					name += " · " + member.RuntimeError
				}
				ui.Text(c, name).TextColor(muted)
			}
		})
		if ShowLoadEarlier(m.Snapshot) && m.OnLoadEarlier != nil && ui.Button(c, "加载更早讨论 / Earlier discussions").Disabled(m.Snapshot.HistoryPageBusy).Clicked() {
			m.OnLoadEarlier()
		}
		if m.Snapshot.HistoryPageErr != "" {
			ui.Text(c, m.Snapshot.HistoryPageErr).TextColor(muted)
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Padding(12).Gap(12).Children(func() {
				if len(threads) == 0 {
					ui.Text(c, "尚无讨论 / No discussion yet").TextColor(muted)
				}
				for i, thread := range threads {
					key := "group-thread:" + m.Snapshot.SelectedID + ":" + thread.ID
					expanded := i == len(threads)-1
					if m.ThinkingExpanded != nil && *m.ThinkingExpanded != nil {
						if value, ok := (*m.ThinkingExpanded)[key]; ok {
							expanded = value
						}
					}
					ui.Column(c.Key(key)).FillWidth().Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
						label := fmt.Sprintf("%s · %d 条消息 / messages", thread.ID, len(thread.Messages))
						if !expanded && len(thread.Messages) > 0 {
							label = thread.Messages[0].Text + " · " + label
						}
						if ui.Button(c, label).Clicked() && m.ThinkingExpanded != nil {
							if *m.ThinkingExpanded == nil {
								*m.ThinkingExpanded = map[string]bool{}
							}
							(*m.ThinkingExpanded)[key] = !expanded
						}
						if !expanded {
							return
						}
						for _, message := range thread.Messages {
							ui.Column(c.Key(message.ID)).FillWidth().Gap(4).Children(func() {
								label := message.Speaker
								if message.Late {
									label += " · 延迟回复 / Late reply"
								}
								if message.State != "settled" {
									label += " · " + message.State
								}
								ui.Text(c, label).FontSize(12).Bold().TextColor(muted)
								if p := message.Original.Submission; p != nil {
									copy := *p
									copy.Text = message.Text
									paintPendingSubmission(c, copy, fg, muted)
									return
								}
								if message.User {
									refs, _ := userMessageParts(message.Original.Body)
									userMessageView(c, m, message.Original, colors, refs, message.Text, m.OnOpenImageLightbox)
									for _, name := range session.GroupAttachmentNames(message.Original) {
										ui.Text(c, "📎 "+name).TextColor(muted)
									}
								} else if message.Text != "" {
									markdown.ViewInteractive(c, message.Text, markdown.Style{FontSize: 14, FG: fg, Muted: muted, CodeBG: chrome.Color(colors.Field), Accent: chrome.Color(colors.Accent)}, m.OnOpenCitationURL)
								}
							})
						}
						if m.OnReplyGroupThread != nil && ui.Button(c, "回复此讨论 / Reply to thread").Disabled(m.Snapshot.PromptBusy).Clicked() {
							m.OnReplyGroupThread(thread.ID)
						}
					})
				}
			})
		})
	})
}
