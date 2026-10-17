// Package chat is the virtualized message list stub for M4.
package chat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
	"whaleisle/internal/ui/widgets/markdown"
)

// Model is chat state for one frame.
type Model struct {
	Snapshot         session.ViewState
	ThemeID          string
	SystemDark       bool
	List             *ui.ListState
	ThinkingExpanded *map[string]bool
	ConnectionBanner string
	OnReconnect      func()
	// UsageJobsToday nil omits the jobs segment; non-nil shows even when zero.
	UsageJobsToday *int
	UsageBalance   string
	OnOpenUsage    func()
	// OnMessageFeedback submits thumbs (rating "up"|"down") with optional note.
	OnMessageFeedback func(messageID, rating, note string)
	// OnClearMessageFeedback maps to messageFeedback/delete.
	OnClearMessageFeedback func(messageID string)
	// FeedbackCommentMsgID / FeedbackCommentDraft drive the optional dislike reason field.
	FeedbackCommentMsgID *string
	FeedbackCommentDraft *string
	OnEditMessage        func(seq float64, body string)
	// OnQuoteMessage inserts a markdown quote of the message into the composer draft.
	OnQuoteMessage       func(body string)
	OnLoadEarlier        func()
	ImageLightboxPath    *string
	OnOpenImageLightbox  func(displayPath string)
	OnCloseImageLightbox func()
	OnOpenCitationURL    func(url string)
	// OnOpenDeliverable opens a presented file path in the files dock.
	OnOpenDeliverable func(path string)
}

// View paints the message list with FollowEnd for future streaming.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	// Let the frame desk-wash show through (Electron chat pane is translucent mica).
	_ = colors.BG

	msgs := m.Snapshot.Messages
	list := m.List
	if list == nil {
		list = &ui.ListState{FollowEnd: true}
	}

	ui.Column(c).Fill().MinHeight(0).Children(func() {
		ConnBannerView(c, m.ConnectionBanner, muted, chrome.Color(colors.Field), border, m.OnReconnect)
		UsageLabelView(c, FormatUsageLabel(m.UsageJobsToday, m.UsageBalance), muted, chrome.Color(colors.Field), border, m.OnOpenUsage)
		if m.Snapshot.LastAction != "" {
			ui.Text(c, m.Snapshot.LastAction).FontSize(11).TextColor(muted).Padding(8, 12, 0, 12)
		}
		ui.Box(c).Fill().Height(1).Background(border)
		if ShowLoadEarlier(m.Snapshot) && m.OnLoadEarlier != nil {
			label := i18n.T(i18n.KeyChatLoadEarlier)
			if m.Snapshot.HistoryPageBusy {
				label = i18n.T(i18n.KeyChatLoadEarlierBusy)
			}
			ui.Row(c).Padding(8, 12, 0, 12).Children(func() {
				if ui.Button(c, label).Disabled(m.Snapshot.HistoryPageBusy).Clicked() {
					m.OnLoadEarlier()
				}
			})
		}
		if len(msgs) == 0 {
			ui.Text(c, i18n.T(i18n.KeyChatEmpty)).FontSize(13).TextColor(muted).Padding(16, 16, 16, 16)
			ui.Box(c).Grow(1) // keep composer pinned to the bottom of the dialogue column
			return
		}
		ui.List(c, list, len(msgs)).Fill().Rows(func(row ui.ListRow) {
			msg := msgs[row.Index]
			padT, padR, padB, padL := float32(12), float32(16), float32(8), float32(16)
			isSystem := msg.Kind == session.MessageKindText && msg.Role == "system"
			ui.Column(row.Context).Padding(padT, padR, padB, padL).Gap(4).Fill().Children(func() {
				// Electron turn/end strip: 「已完成，用时 … ▾」.
				if msg.Role == "status" && msg.EventType == "turn/end" {
					ui.Text(row.Context, formatTurnStatus(msg.Body)+" ▾").FontSize(12).TextColor(muted)
					return
				}
				// Electron chat: system failures as red banner; no role chips on user/assistant.
				if isSystem {
					body := msg.Body
					code := ""
					if i := strings.LastIndex(body, " MISSING_"); i >= 0 {
						code = strings.TrimSpace(body[i+1:])
						body = strings.TrimSpace(body[:i])
					}
					title := i18n.T(i18n.KeyChatRunFailed)
					detail := body
					if strings.HasPrefix(body, title) {
						detail = strings.TrimSpace(strings.TrimPrefix(body, title))
						detail = strings.TrimLeft(detail, "：: ")
					} else if strings.HasPrefix(body, "本轮运行失败") {
						detail = strings.TrimSpace(strings.TrimPrefix(body, "本轮运行失败"))
						detail = strings.TrimLeft(detail, "：: ")
					}
					fail := chrome.Color("rgb(220, 38, 38)")
					ui.Row(row.Context).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
						ui.Text(row.Context, "●").FontSize(12).TextColor(fail)
						ui.Text(row.Context, title).FontSize(13).FontWeight(700).TextColor(fail)
						if detail != "" {
							ui.Text(row.Context, detail).FontSize(12).TextColor(fg).Grow(1)
						} else {
							ui.Box(row.Context).Grow(1)
						}
						if code != "" {
							ui.Text(row.Context, code).FontSize(11).FontWeight(600).TextColor(muted)
						}
					})
					return
				}
				if msg.Kind != session.MessageKindText || (msg.Role != "user" && msg.Role != "assistant") {
					if msg.Kind != session.MessageKindThinking {
						role := msg.Role
						if msg.Streaming {
							role += " …"
						}
						ui.Text(row.Context, role).FontSize(11).Bold().TextColor(muted)
					}
				}
				switch msg.Kind {
				case session.MessageKindTool:
					tc := DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent))
					ToolCardView(row.Context, msg.EventType, msg.Raw, tc)
					CitationsView(row.Context, CitationsForMessage(msg), muted, chrome.Color(colors.Accent), m.OnOpenCitationURL)
				case session.MessageKindDeliverables:
					DeliverablesView(row.Context, msg, fg, muted, border, chrome.Color(colors.Field), m.OnOpenDeliverable)
				case session.MessageKindCompaction:
					CompactionView(row.Context, msg, fg, muted, chrome.Color(colors.Field))
				case session.MessageKindThinking:
					thSt := ThinkingStyle{
						FG:     fg,
						Muted:  muted,
						Border: border,
						Field:  chrome.Color(colors.Field),
						Size:   12,
					}
					ThinkingBlockView(row.Context, msg.ID, msg.Body, m.ThinkingExpanded, thSt)
				default:
					refs := session.ImageRefs(msg.Body)
					body := session.BodyWithoutImagePlaceholders(msg.Body)
					onOpenImage := func(ref string) {
						p := DisplayImagePath(ref)
						if p == "" {
							return
						}
						if m.OnOpenImageLightbox != nil {
							m.OnOpenImageLightbox(p)
						} else if m.ImageLightboxPath != nil {
							*m.ImageLightboxPath = p
						}
					}
					for _, ref := range refs {
						var open func(string)
						if m.OnOpenImageLightbox != nil || m.ImageLightboxPath != nil {
							open = onOpenImage
						}
						ImageInlineView(row.Context, ref, fg, muted, open)
					}
					patch := diff.ExtractPatchText(body)
					if patch != "" {
						diff.View(row.Context, patch, diff.Style{
							FontSize: 12,
							FG:       fg,
							Add:      ui.RGB(61, 153, 112),
							Del:      ui.RGB(224, 102, 102),
							Hunk:     chrome.Color(colors.Accent),
							File:     muted,
							Muted:    muted,
						})
					} else if body != "" {
						md := markdown.Style{
							FontSize: 14,
							FG:       fg,
							Muted:    muted,
							CodeBG:   chrome.Color(colors.Field),
							Accent:   chrome.Color(colors.Accent),
						}
						paintMD := func() {
							if m.OnOpenCitationURL != nil {
								markdown.ViewInteractive(row.Context, body, md, m.OnOpenCitationURL)
							} else {
								markdown.View(row.Context, body, md)
							}
						}
						if msg.Role == "user" {
							// Electron user bubble: right-aligned off-white card on cool-grey wash.
							userBG := chrome.Color("rgb(243, 245, 249)")
							if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
								userBG = chrome.Color(colors.Hover)
								if userBG.A == 0 {
									userBG = chrome.Color(colors.Field)
								}
							}
							ui.Row(row.Context).FillWidth().AlignItems(ui.End).Children(func() {
								ui.Box(row.Context).Grow(1).MinWidth(40)
								ui.Column(row.Context).Background(userBG).
									Radius(14).Padding(12, 14, 12, 14).Gap(6).Children(func() {
									paintMD()
									// Electron user card footer: time + compact actions on the bubble.
									ui.Row(row.Context).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Box(row.Context).Grow(1).MinWidth(0)
										if msg.EventTime > 0 {
											ui.Text(row.Context, formatEventTime(msg.EventTime)).FontSize(11).TextColor(muted)
										}
										if showMessageCopy(msg) {
											if ui.Button(row.Context, "⎘").Tooltip(i18n.T(i18n.KeyChatCopy)).
												FontSize(11).BorderWidth(0).Background(ui.Transparent).
												TextColor(muted).Clicked() {
												mygo.Clipboard.WriteText(msg.Body)
											}
										}
										if m.OnQuoteMessage != nil {
											if ui.Button(row.Context, "↗").Tooltip(i18n.T(i18n.KeyChatQuote)).
												FontSize(11).BorderWidth(0).Background(ui.Transparent).
												TextColor(muted).Clicked() {
												m.OnQuoteMessage(msg.Body)
											}
										}
									})
								})
							})
						} else {
							paintMD()
						}
					}
					if msg.Role == "assistant" {
						CitationsView(row.Context, CitationsForMessage(msg), muted, chrome.Color(colors.Accent), m.OnOpenCitationURL)
					}
				}
				// User actions live on the bubble footer; assistant keeps the text button row.
				if msg.Role != "user" && (showMessageCopy(msg) || m.OnQuoteMessage != nil) {
					ui.Row(row.Context).Gap(8).Children(func() {
						if showMessageCopy(msg) {
							if ui.Button(row.Context, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
								mygo.Clipboard.WriteText(msg.Body)
							}
						}
						if showMessageCopy(msg) && m.OnQuoteMessage != nil {
							if ui.Button(row.Context, i18n.T(i18n.KeyChatQuote)).FontSize(11).Clicked() {
								m.OnQuoteMessage(msg.Body)
							}
						}
					})
				}
				if showMessageEdit(msg) && m.OnEditMessage != nil {
					ui.Row(row.Context).Gap(8).Children(func() {
						if ui.Button(row.Context, i18n.T(i18n.KeyChatEdit)).Clicked() {
							m.OnEditMessage(msg.Seq, msg.Body)
						}
					})
				}
				if showMessageFeedback(msg) && m.OnMessageFeedback != nil {
					selected := ""
					if m.Snapshot.MessageRatings != nil {
						selected = m.Snapshot.MessageRatings[msg.ID]
					}
					commentOpen := m.FeedbackCommentMsgID != nil && *m.FeedbackCommentMsgID == msg.ID
					feedbackRow(row.Context, FeedbackRowModel{
						MessageID:   msg.ID,
						Selected:    selected,
						Busy:        m.Snapshot.FeedbackBusyID != "",
						CommentOpen: commentOpen,
						Comment:     m.FeedbackCommentDraft,
						OnVote:      m.OnMessageFeedback,
						OnClear:     m.OnClearMessageFeedback,
						OnOpenDown: func(messageID string) {
							if m.FeedbackCommentMsgID != nil {
								if *m.FeedbackCommentMsgID != messageID && m.FeedbackCommentDraft != nil {
									*m.FeedbackCommentDraft = ""
								}
								*m.FeedbackCommentMsgID = messageID
							}
						},
						OnCancelDown: func() {
							if m.FeedbackCommentMsgID != nil {
								*m.FeedbackCommentMsgID = ""
							}
							if m.FeedbackCommentDraft != nil {
								*m.FeedbackCommentDraft = ""
							}
						},
					})
				}
			})
		})
	})
	if m.ImageLightboxPath != nil && m.OnCloseImageLightbox != nil {
		ImageLightboxOverlay(c, *m.ImageLightboxPath, m.ThemeID, m.SystemDark, m.OnCloseImageLightbox)
	}
}

func formatTurnStatus(elapsedMsBody string) string {
	ms, err := strconv.ParseFloat(strings.TrimSpace(elapsedMsBody), 64)
	if err != nil || ms <= 0 {
		return i18n.T(i18n.KeyChatTurnDone)
	}
	totalSec := int(ms/1000 + 0.5)
	if totalSec < 60 {
		return fmt.Sprintf(i18n.T(i18n.KeyChatTurnDoneSec), totalSec)
	}
	min := totalSec / 60
	sec := totalSec % 60
	if sec == 0 {
		return fmt.Sprintf(i18n.T(i18n.KeyChatTurnDoneMin), min)
	}
	return fmt.Sprintf(i18n.T(i18n.KeyChatTurnDoneMinSec), min, sec)
}

func showMessageEdit(msg session.Message) bool {
	return msg.Role == "user" && msg.Seq > 0 && msg.Kind == session.MessageKindText && !msg.Streaming
}

func showMessageCopy(msg session.Message) bool {
	if msg.Streaming || strings.TrimSpace(msg.Body) == "" {
		return false
	}
	return msg.Kind == session.MessageKindText && (msg.Role == "user" || msg.Role == "assistant")
}

func showMessageFeedback(msg session.Message) bool {
	return msg.Role == "assistant" && !msg.Streaming && msg.Kind != session.MessageKindTool
}
