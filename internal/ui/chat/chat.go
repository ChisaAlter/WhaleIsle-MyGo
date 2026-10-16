// Package chat is the virtualized message list stub for M4.
package chat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
	"whaleisle/internal/ui/widgets/markdown"
)

// Model is chat state for one frame.
type Model struct {
	Preferences        settings.ChatPreferences
	ScheduleStore      *schedstore.Store
	OnOpenScheduleTask func(schedstore.Entry)
	HistoryStore       *session.Store
	FeedbackStore      *session.Store
	EditReturnSeq      *float64
	EditingSeq         float64
	OnCancelEdit       func()
	Snapshot           session.ViewState
	ThemeID            string
	SystemDark         bool
	// ContentWidth is the shared chat width axis; zero uses the original 748px fallback.
	ContentWidth     float32
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
	OnForkTurn           func(seq float64)
	// OnQuoteMessage inserts a markdown quote of the message into the composer draft.
	OnQuoteMessage       func(body string)
	OnLoadEarlier        func()
	ImageLightboxPath    *string
	OnOpenImageLightbox  func(displayPath string)
	OnCloseImageLightbox func()
	OnOpenCitationURL    func(url string)
	OnOpenPlanResource   func(planpreview.Document)
	// OnOpenDeliverable opens a presented file path in the files dock.
	OnOpenDeliverable            func(path string)
	VisualReply                  session.VisualReplyState
	VisualReplyBitmap            *ui.Bitmap
	OnPreviewVisualReply         func(attachmentID string)
	OnInteractVisualReply        func(attachmentID string)
	OnStopInteractiveVisualReply func()
	OnReadVisualReply            func(attachmentID string)
	OnSaveVisualReply            func(attachmentID string)
	OnCloseVisualReply           func()
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

	msgs := session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(m.Snapshot.Messages), m.Snapshot.WorkflowRuns)))
	msgs, transcriptControls := projectTranscript(msgs, m)
	list := m.List
	if list == nil {
		list = &ui.ListState{FollowEnd: true}
	}

	cards := prepareScheduleContext(m)
	prepareHistoryNavigation(m, msgs, list)
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		ConnBannerView(c, m.ConnectionBanner, muted, chrome.Color(colors.Field), border, m.OnReconnect)
		UsageLabelView(c, FormatUsageLabel(m.UsageJobsToday, m.UsageBalance), muted, chrome.Color(colors.Field), border, m.OnOpenUsage)
		if m.Snapshot.LastAction != "" {
			ui.Text(c, m.Snapshot.LastAction).FontSize(11).TextColor(muted).Padding(8, 12, 0, 12)
		}
		ScheduleHeaderView(c, m)
		HistoryNavigationView(c, m, list, muted)
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
				if control, ok := transcriptControls[msg.ID]; ok {
					transcriptControlView(row.Context, control, m, muted)
					return
				}
				if msg.Submission != nil {
					paintPendingSubmission(row.Context, *msg.Submission, fg, muted)
					return
				}
				if msg.TurnDelivery != nil {
					TurnDeliveryView(row.Context, *msg.TurnDelivery, m, DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent)))
					return
				}
				if msg.Workflow != nil {
					WorkflowRunView(row.Context, *msg.Workflow, m, DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent)))
					return
				}
				if msg.Kind == session.MessageKindSubmittedPlan {
					SubmittedPlanCardView(row.Context, msg, m.Snapshot.SelectedID, planpreview.Style{FG: fg, Muted: muted, Border: border, Field: chrome.Color(colors.Field), Accent: chrome.Color(colors.Accent)}, m.OnOpenPlanResource)
					return
				}
				// Electron turn/end strip: 「已完成，用时 … ▾」.
				if msg.Role == "status" && msg.EventType == "turn/end" {
					ScheduleTurnCardsView(row.Context, m, cards[msg.Seq])
					TurnUsageView(row.Context, m, msg, muted)
					ui.Row(row.Context).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(row.Context, completedTurnStatus(msg)).FontSize(12).TextColor(muted)
						if m.OnForkTurn != nil && session.CanForkTurn(m.Snapshot.Messages, msg.Seq) {
							if ui.Button(row.Context, "从这里分支 / Branch here").FontSize(11).Disabled(m.Snapshot.HistoryBranchBusy || !m.Snapshot.ClientReady).Clicked() {
								m.OnForkTurn(msg.Seq)
							}
						}
					})
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
					if msg.Kind != session.MessageKindThinking && msg.Kind != session.MessageKindCommand {
						role := msg.Role
						if msg.Streaming {
							role += " …"
						}
						ui.Text(row.Context, role).FontSize(11).Bold().TextColor(muted)
					}
				}
				if publication, ok := session.PublishedVisualReply(msg.Raw); ok {
					VisualReplyView(row.Context, publication, m, fg, muted, border, chrome.Color(colors.Field))
				}
				switch msg.Kind {
				case session.MessageKindCommand:
					CommandCardView(row.Context, msg, m, DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent)))
				case session.MessageKindTool:
					tc := DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent))
					ToolMessageView(row.Context, msg, tc, ToolImageGalleryOptions{
						ID: msg.ID, Expanded: m.ThinkingExpanded,
						OnOpenURL: m.OnOpenCitationURL,
						OnOpen: func(path string) {
							if m.OnOpenImageLightbox != nil {
								m.OnOpenImageLightbox(path)
							} else if m.ImageLightboxPath != nil {
								*m.ImageLightboxPath = path
							}
						},
					})
				case session.MessageKindDeliverables:
					// Unbound/legacy declarations remain readable without path-open authority.
					DeliverablesView(row.Context, msg, fg, muted, border, chrome.Color(colors.Field), nil)
				case session.MessageKindCompaction:
					CompactionView(row.Context, msg, fg, muted, chrome.Color(colors.Field))
				case session.MessageKindThinking:
					thSt := ThinkingStyle{
						HidePreview: m.Preferences.TranscriptView == "compact" && !msg.Streaming,
						FG:          fg,
						Muted:       muted,
						Border:      border,
						Field:       chrome.Color(colors.Field),
						Size:        12,
					}
					ThinkingBlockView(row.Context, msg.ID, msg.Body, m.ThinkingExpanded, thSt)
				case session.MessageKindUnknown:
					ui.Text(row.Context, "未知内容 / Unknown content").TextColor(muted)
					ui.Text(row.Context, string(msg.ContentBlock)).TextColor(fg)
				default:
					refs := session.ImageRefs(msg.Body)
					if msg.ImageRef != "" {
						refs = []string{msg.ImageRef}
					}
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
					if msg.Role == "user" {
						refs, body := userMessageParts(msg.Body)
						userMessageView(row.Context, m, msg, colors, refs, body, onOpenImage)
						break
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
						md.ResolveFile = FileMentionResolver(m.Snapshot.Messages, msg)
						md.OnOpenFile = m.OnOpenDeliverable
						paintMD := func() {
							if m.OnOpenCitationURL != nil {
								markdown.ViewInteractive(row.Context, body, md, m.OnOpenCitationURL)
							} else {
								markdown.View(row.Context, body, md)
							}
						}
						paintMD()
					}
					if msg.Role == "assistant" {
						CitationsView(row.Context, CitationsForMessage(msg), muted, chrome.Color(colors.Accent), m.OnOpenCitationURL)
					}
				}
				// User actions are already painted outside its bubble; assistant keeps its row.
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
				if msg.Role != "user" && showMessageEdit(msg) && m.OnEditMessage != nil {
					ui.Row(row.Context).Gap(8).Children(func() {
						if ui.Button(row.Context, i18n.T(i18n.KeyChatEdit)).Clicked() {
							m.OnEditMessage(msg.Seq, msg.Body)
						}
					})
				}
				if showMessageFeedback(msg) && m.FeedbackStore != nil {
					authoritativeFeedbackRow(row.Context, m.FeedbackStore, msg.FeedbackMessageID())
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
