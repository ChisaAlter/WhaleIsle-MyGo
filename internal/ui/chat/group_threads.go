package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

func groupThreadsView(c *ui.Context, m Model) {
	t := design.For(c)
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	colors := palette
	fg, muted, border := t.Foreground, t.MutedForeground, t.Border
	threads := session.ProjectGroupThreads(m.Snapshot.Messages, m.GroupMembers)
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		ConnBannerView(c, palette, m.ConnectionBanner, muted, t.Muted, border, m.OnReconnect)
		ui.Row(c).Padding(design.Sp(2), design.Sp(3)).Gap(design.Sp(2)).Wrap().Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatGroupTitle)).Bold().TextColor(fg)
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
		if ShowLoadEarlier(m.Snapshot) && m.OnLoadEarlier != nil && stdButton(c, palette, i18n.T(i18n.KeyChatGroupEarlier), buttonGhost, m.Snapshot.HistoryPageBusy).Clicked() {
			m.OnLoadEarlier()
		}
		if m.Snapshot.HistoryPageErr != nil {
			ui.Text(c, m.Snapshot.HistoryPageErr.SentenceText()).TextColor(muted)
		}
		coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
			ui.Column(c).FillWidth().Padding(design.Sp(3)).Gap(design.Sp(3)).Children(func() {
				if len(threads) == 0 {
					ui.Text(c, i18n.T(i18n.KeyChatGroupEmpty)).TextColor(muted)
				}
				for i, thread := range threads {
					key := "group-thread:" + m.Snapshot.SelectedID + ":" + thread.ID
					expanded := i == len(threads)-1
					if m.ThinkingExpanded != nil && *m.ThinkingExpanded != nil {
						if value, ok := (*m.ThinkingExpanded)[key]; ok {
							expanded = value
						}
					}
					chatCardSurface(c, ui.Column(c.Key(key)).FillWidth().Padding(design.Sp(3)).Gap(design.Sp(2)), design.RadiusMD).Children(func() {
						label := fmt.Sprintf(i18n.T(i18n.KeyChatGroupCount), thread.ID, len(thread.Messages))
						if !expanded && len(thread.Messages) > 0 {
							label = thread.Messages[0].Text + " · " + label
						}
						if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM}).Clicked() && m.ThinkingExpanded != nil {
							if *m.ThinkingExpanded == nil {
								*m.ThinkingExpanded = map[string]bool{}
							}
							(*m.ThinkingExpanded)[key] = !expanded
						}
						if !expanded {
							return
						}
						for _, message := range thread.Messages {
							ui.Column(c.Key(message.ID)).FillWidth().Gap(design.Sp(1)).Children(func() {
								label := message.Speaker
								if message.Late {
									label += " · " + i18n.T(i18n.KeyChatGroupLate)
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
									markdown.ViewInteractive(c, message.Text, transcriptMarkdownStyleFor(c, theme.CurrentPreferences().Chat.Size), m.OnOpenCitationURL)
								}
							})
						}
						if m.OnReplyGroupThread != nil && stdButton(c, palette, i18n.T(i18n.KeyChatGroupReply), buttonGhost, m.Snapshot.PromptBusy).Clicked() {
							m.OnReplyGroupThread(thread.ID)
						}
					})
				}
			})
		}).Grow(1).MinHeight(0)
	})
}
