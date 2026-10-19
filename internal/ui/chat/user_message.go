package chat

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// userMessageParts recognizes the legacy attachment lines outside code fences.
// User-authored whitespace and fenced image-like text remain literal content.
func userMessageParts(body string) (refs []string, text string) {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	seen := map[string]bool{}
	var fence byte
	fenceLength := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		marker, count := byte(0), 0
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			marker = trimmed[0]
			for count < len(trimmed) && trimmed[count] == marker {
				count++
			}
		}
		if fence != 0 {
			kept = append(kept, line)
			if marker == fence && count >= fenceLength && strings.TrimSpace(trimmed[count:]) == "" {
				fence = 0
			}
			continue
		}
		if count >= 3 {
			fence, fenceLength = marker, count
			kept = append(kept, line)
			continue
		}
		for _, ref := range session.ImageRefs(line) {
			if !seen[ref] {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
		if !strings.HasPrefix(trimmed, "[image] ") {
			kept = append(kept, line)
		}
	}
	return refs, strings.Join(kept, "\n")
}

// userMessageView follows MessageItem's userRow/userStack geometry. The two
// nested caps express min(chat-width * .702, 82% of the available row).
func userMessageView(c *ui.Context, m Model, msg session.Message, colors theme.Colors, refs []string, body string, onOpenImage func(string)) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	referenceLabels := userMessageReferenceLabels(m.Snapshot.Messages, msg.Seq)
	axis := m.ContentWidth
	if axis <= 0 {
		axis = 748
	}
	owner := ui.Column(c).FillWidth().Gap(6).AlignItems(ui.End)
	owner.Children(func() {
		ui.Column(c).FillWidth().MaxWidthPercent(82).MinWidth(0).AlignItems(ui.End).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(axis * .702).MinWidth(0).Gap(8).AlignItems(ui.End).Children(func() {
				for _, ref := range refs {
					var open func(string)
					if m.OnOpenImageLightbox != nil || m.ImageLightboxPath != nil {
						open = onOpenImage
					}
					ImageInlineView(c, ref, fg, muted, open)
				}
				if body != "" {
					ui.Column(c).Label("User message bubble").MinWidth(0).MaxWidthPercent(100).
						Background(chrome.Color(colors.Bubble)).Radius(20).Padding(10, 16, 10, 16).Children(func() {
						ink := fg
						if m.EditingSeq > 0 && msg.Seq == m.EditingSeq {
							ink = muted
						}
						var openReference func(string)
						var openSkill func(string)
						if m.OnOpenDeliverable != nil && m.HistoryStore != nil {
							scope := m.Snapshot.TurnFiles.Scope
							openReference = func(path string) {
								if scope.Binding != "" && scope == m.HistoryStore.Snapshot().TurnFiles.Scope {
									m.OnOpenDeliverable(path)
								}
							}
							openSkill = func(name string) {
								ref, ok := m.HistoryStore.CaptureHistorySkillReference(scope, name)
								if !ok {
									return
								}
								go func() {
									resolved, ok := m.HistoryStore.ResolveHistorySkillReference(context.Background(), ref)
									if !ok {
										return
									}
									mygo.RunOnMain(func() {
										if path, ok := m.HistoryStore.HistorySkillReferencePath(resolved); ok {
											m.OnOpenDeliverable(path)
										}
									})
								}()
							}
						}
						userReferenceText(c, body, theme.CurrentPreferences().Chat.Size, ink, muted, openReference, userReferenceFacts{Labels: referenceLabels, Skills: userMessageSkillNames(m.Snapshot.Messages, msg), OpenSkill: openSkill, OpenFile: m.OnOpenHistoryReference, Current: func() bool {
							return m.HistoryStore != nil && m.Snapshot.TurnFiles.Scope.Binding != "" && m.Snapshot.TurnFiles.Scope == m.HistoryStore.Snapshot().TurnFiles.Scope
						}})
					})
				}
				if len(referenceLabels) > 0 {
					ui.Text(c, "Referenced session · "+strings.Join(referenceLabels, ", ")).FontSize(11).TextColor(muted)
				}
				// Preserve the existing URL-opening action alongside literal user text.
				if m.OnOpenCitationURL != nil {
					links := markdown.Links(body)
					if len(links) > 0 {
						ui.Row(c).FillWidth().Wrap().Gap(6).Justify(ui.End).Children(func() {
							for _, link := range links {
								label := "↗ " + link.Label
								button := ui.Button(c, "").Label(label).Tooltip(link.URL).MaxWidthPercent(100).MinWidth(0).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).TextColor(chrome.Color(colors.Accent))
								button.Children(func() { ui.Text(c, label).Grow(1).MinWidth(0).SingleLine() })
								if button.Clicked() {
									m.OnOpenCitationURL(link.URL)
								}
							}
						})
					}
				}
			})
		})
		actions := ui.Row(c).Label("User message actions").Gap(8).Height(28 + theme.CurrentPreferences().Chat.Size - 14).AlignItems(ui.Center)
		messageQuoteMenu(actions, m, msg)
		if !latestUserMessage(m.Snapshot.Messages, msg) && !owner.Hovered() && !owner.FocusWithin() {
			actions.Opacity(0)
		}
		actions.Children(func() {
			if msg.EventTime > 0 {
				ui.Text(c, formatEventTime(msg.EventTime)).FontSize(13).FixedLineHeight(24).Padding(0, 12, 0, 0).TextColor(muted)
			}
			action := func(icon *ui.SVG, label string, run func()) {
				if messageAction(c, icon, label, false, muted).Clicked() {
					run()
				}
			}
			if showMessageCopy(msg) {
				messageCopyAction(c, msg, false, muted)
			}
			if m.EditingSeq > 0 && msg.Seq == m.EditingSeq && m.OnCancelEdit != nil {
				ui.Text(c, i18n.T(i18n.KeyChatEditHint)).FontSize(11).TextColor(muted)
				action(messageIconCloseOutline, i18n.T(i18n.KeyQuitConfirmCancel), m.OnCancelEdit)
			} else if msg.Seq == session.LatestEditableMessage(m.Snapshot.Messages) && m.OnEditMessage != nil {
				tip := i18n.T(i18n.KeyChatEdit)
				editAllowed := session.CanEditMessage(m.Snapshot, msg.Seq)
				if !editAllowed && m.Snapshot.SelectedID == "" && !m.Snapshot.PromptBusy && !m.Snapshot.MessageEditBusy {
					editAllowed = true
				}
				if m.Snapshot.PromptBusy || m.Snapshot.MessageEditBusy {
					tip = i18n.T(i18n.KeyChatEditRunning)
				} else if !editAllowed {
					tip = i18n.T(i18n.KeyChatEditUnsupported)
				}
				button := messageAction(c, messageIconEditOutline, i18n.T(i18n.KeyChatEdit), false, muted).Tooltip(tip)
				if !editAllowed {
					button.Opacity(.4)
				}
				if m.EditReturnSeq != nil && *m.EditReturnSeq > 0 && *m.EditReturnSeq == msg.Seq {
					button.Focus()
					*m.EditReturnSeq = 0
				}
				if editAllowed && button.Clicked() {
					m.OnEditMessage(msg.Seq, msg.Body)
				}
			}
		})
	})
}

func latestUserMessage(messages []session.Message, msg session.Message) bool {
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == "user" {
			return messages[index].ID == msg.ID && messages[index].Seq == msg.Seq
		}
	}
	return true
}
