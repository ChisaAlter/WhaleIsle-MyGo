package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// parsedUserLinks memoizes the markdown link scan for one user message body.
type parsedUserLinks struct {
	source string
	links  []markdown.Link
}

// userMessageParts recognizes the legacy attachment lines outside code fences.
// User-authored whitespace and fenced image-like text remain literal content.
func userMessageParts(body string) (refs []string, text string) {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	seen := map[string]bool{}
	var fence userTextFence
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence.consumeLine(line) {
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
	if m.RenderSlot != nil && !m.nativeUserEditor {
		native := m
		native.nativeUserEditor = true
		m.RenderSlot(c, m.Snapshot, msg, "conversation.chat.user-editor", func() { userMessageView(c, native, msg, colors, refs, body, onOpenImage) })
		return
	}
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	border, field := chrome.Color(colors.Border), chrome.Color(colors.Field)
	referenceLabels, skillNames := userMessageContextFacts(m.Snapshot, msg)
	axis := m.ContentWidth
	if axis <= 0 {
		axis = 748
	}
	capW := max(float32(748), axis) * .702
	if m.ContentWidth > 0 && m.ContentWidth < 748 {
		capW = 536
	}
	owner := ui.Column(c).FillWidth().Gap(6).AlignItems(ui.End)
	content := ui.Local(owner, "user-content", func() userContentMemo { return userContentMemo{} })
	content.resolve(msg)
	if content.value.Structured {
		body = content.value.Text
		msg.Body = body // User actions copy text, never attachment placeholders.
	}
	owner.Children(func() {
		paintContent := func() {
			ui.Column(c).FillWidth().MaxWidthPercent(82).MinWidth(0).AlignItems(ui.End).Children(func() {
				ui.Column(c).FillWidth().MaxWidth(capW).MinWidth(0).Gap(8).AlignItems(ui.End).Children(func() {
					var open func(string)
					if m.OnOpenImageLightbox != nil || m.ImageLightboxPath != nil {
						open = onOpenImage
					}
					if content.value.Structured {
						userContentAttachments(c, content.value.Attachments, fg, muted, border, field, open)
					} else if len(refs) > 1 {
						ui.Row(c).Wrap().Gap(8).Justify(ui.End).MaxWidthPercent(100).Children(func() {
							for _, ref := range refs {
								paintImageTile(c, ref, muted, open)
							}
						})
					} else {
						for _, ref := range refs {
							ImageInlineView(c, ref, fg, muted, open)
						}
					}
					if body != "" || len(content.value.ExtraBlocks) > 0 {
						radii := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark).BubbleRadii
						ui.Column(c).Label("User message bubble").MinWidth(0).MaxWidthPercent(100).
							Background(chrome.Color(colors.Bubble)).Radius(radii[:]...).Padding(10, 16, 10, 16).Children(func() {
							ink := fg
							if m.EditingSeq > 0 && msg.Seq == m.EditingSeq {
								ink = muted
							}
							var openReference func(string)
							var openSkill func(string)
							if m.OnOpenDeliverable != nil && m.HistoryStore != nil {
								scope := m.Snapshot.TurnFiles.Scope
								openReference = func(path string) {
									if scope.Binding != "" && scope == m.HistoryStore.TurnFilesScope() {
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
							// Root history references require their captured turn-file scope.
							// Addressed child readers instead validate their own token/address
							// in OnOpenHistoryReference; no root Store is attached to them.
							var currentReference func() bool
							if m.HistoryStore != nil {
								currentReference = func() bool {
									return m.Snapshot.TurnFiles.Scope.Binding != "" && m.Snapshot.TurnFiles.Scope == m.HistoryStore.TurnFilesScope()
								}
							}
							userReferenceText(c, body, theme.CurrentPreferences().Chat.Size, ink, muted, openReference, userReferenceFacts{Interactive: true, Occurrence: m.Snapshot.SelectedID + ":" + msg.ID + fmt.Sprint(msg.Seq), Labels: referenceLabels, Skills: skillNames, OpenSkill: openSkill, OpenFile: m.OnOpenHistoryReference, Current: currentReference})
							for index, block := range content.value.ExtraBlocks {
								userExtraBlockView(c.Key(index), block, fg, muted)
							}
						})
					}
					if len(referenceLabels) > 0 {
						prefix, separator := "Referenced session · ", ", "
						if i18n.Locale() == i18n.LocaleZH {
							prefix, separator = "引用会话 · ", "、"
						}
						ui.Text(c, prefix+strings.Join(referenceLabels, separator)).FontSize(11).TextColor(muted)
					}
					// Preserve the existing URL-opening action alongside literal user text.
					if m.OnOpenCitationURL != nil {
						// Links is a second goldmark pass over the body; memoize it on
						// the row so scrolling does not reparse it every frame.
						holder := ui.Local(owner, "user-links", func() parsedUserLinks { return parsedUserLinks{} })
						if holder.source != body {
							*holder = parsedUserLinks{source: body, links: markdown.Links(body)}
						}
						links := holder.links
						if len(links) > 0 {
							ui.Row(c).FillWidth().Wrap().Gap(6).Justify(ui.End).Children(func() {
								for _, link := range links {
									label := "↗ " + maskedUserText(link.Label)
									button := ui.Button(c, "").Label(label).Tooltip(maskedUserText(link.URL)).MaxWidthPercent(100).MinWidth(0).
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
		}
		if m.RenderSlot != nil {
			m.RenderSlot(c, m.Snapshot, msg, "user.content", paintContent)
		} else {
			paintContent()
		}
		actions := ui.Row(c).Label("User message actions").Gap(8).Height(28 + theme.CurrentPreferences().Chat.Size - 14).AlignItems(ui.Center)
		messageActionMenu(c, actions, m, msg)
		if !latestUserMessage(m.Snapshot.Messages, msg) && !owner.Hovered() && !owner.FocusWithin() {
			actions.Opacity(0)
		}
		actions.Children(func() {
			if msg.EventTime > 0 {
				ui.Text(c, formatMessageClock(msg.EventTime, c.Now())).FontSize(13).FixedLineHeight(24).Padding(0, 12, 0, 0).TextColor(muted)
			}
			action := func(icon *ui.SVG, label string, run func()) {
				if messageAction(c, icon, label, false, muted).Clicked() {
					run()
				}
			}
			if showMessageCopy(msg) {
				messageCopyAction(c, msg, false, muted)
			}
			paintEdit := func() {
				if m.EditingSeq > 0 && msg.Seq == m.EditingSeq && m.OnCancelEdit != nil {
					ui.Text(c, i18n.T(i18n.KeyChatEditHint)).FontSize(11).TextColor(muted)
					action(messageIconCloseOutline, i18n.T(i18n.KeyQuitConfirmCancel), m.OnCancelEdit)
				} else if editSeq, editTextOnly := projectedLatestEditable(m.Snapshot); msg.Seq > 0 && msg.Seq == editSeq && m.OnEditMessage != nil {
					tip := i18n.T(i18n.KeyChatEdit)
					editAllowed := editTextOnly && m.Snapshot.SelectedID != "" && !m.Snapshot.PromptBusy && !m.Snapshot.MessageEditBusy
					if !editAllowed && m.Snapshot.SelectedID == "" && !m.Snapshot.PromptBusy && !m.Snapshot.MessageEditBusy && editTextOnly {
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
			}
			if m.RenderSlot != nil {
				m.RenderSlot(c, m.Snapshot, msg, "conversation.chat.user-actions", paintEdit)
			} else {
				paintEdit()
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
