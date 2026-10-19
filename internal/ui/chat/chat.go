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
	OpenUIParent          uintptr
	OpenUIWindow          *mygo.Window
	OpenUIMediaOcclusions []mygo.Rectangle
	OnInspectToolCall     func(string)
	Inspector             *InspectorNavigation
	DisableDeveloperTools bool
	GroupRoom             bool
	GroupMembers          []session.GroupMember
	OnReplyGroupThread    func(string)
	Preferences           settings.ChatPreferences
	ScheduleStore         *schedstore.Store
	OnOpenScheduleTask    func(schedstore.Entry)
	HistoryStore          *session.Store
	FeedbackStore         *session.Store
	EditReturnSeq         *float64
	EditingSeq            float64
	OnCancelEdit          func()
	Snapshot              session.ViewState
	ThemeID               string
	SystemDark            bool
	// ContentWidth is the shared chat width axis; zero uses the original 748px fallback.
	ContentWidth float32
	// FullColumnViewport keeps the main transcript's scrollbar at the column edge.
	FullColumnViewport bool
	List               *ui.ListState
	ThinkingExpanded   *map[string]bool
	ConnectionBanner   string
	OnReconnect        func()
	// SuppressStatus lets the conversation shell keep status visible in Hero,
	// settling and transcript phases without rendering it a second time here.
	SuppressStatus bool
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
	OnOpenChangesReview          func(session.TurnFileScope, session.TurnFileTarget)
	VisualReply                  session.VisualReplyState
	VisualReplyBitmap            *ui.Bitmap
	OnPreviewVisualReply         func(attachmentID string)
	OnInteractVisualReply        func(attachmentID string)
	OnInlineVisualReply          func(attachmentID string) session.InlineVisualReply
	OnStopInteractiveVisualReply func()
	OnReadVisualReply            func(attachmentID string)
	OnSaveVisualReply            func(attachmentID string)
	OnCloseVisualReply           func()
}

// View paints the message list with FollowEnd for future streaming.
func View(c *ui.Context, m Model) {
	if m.GroupRoom {
		groupThreadsView(c, m)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	// Let the frame desk-wash show through (Electron chat pane is translucent mica).
	_ = colors.BG

	msgs := session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(m.Snapshot.Messages), m.Snapshot.WorkflowRuns)))
	msgs, transcriptControls := projectTranscript(msgs, m)
	actionTargets := assistantActionTargets(msgs)
	list := m.List
	if list == nil {
		list = &ui.ListState{FollowEnd: true}
	}

	cards := prepareScheduleContext(m)
	prepareHistoryNavigation(m, msgs, list)
	prepareSessionInspector(m, msgs, transcriptControls, list)
	if m.Inspector != nil && m.Inspector.SessionID == m.Snapshot.SelectedID && m.Inspector.Target != nil {
		c.Services().Invalidate()
	}
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		if !m.SuppressStatus {
			StatusView(c, m)
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
		transcript := ui.List(c, list, len(msgs)).Fill()
		if m.FullColumnViewport {
			transcript.ScrollbarInsets(0, -3, 0, 0)
		}
		transcript.Rows(func(row ui.ListRow) {
			paintMessage := func() {
				msg := msgs[row.Index]
				padT, padR, padB, padL := float32(12), float32(16), float32(8), float32(16)
				isSystem := msg.Kind == session.MessageKindText && msg.Role == "system"
				messageRoot := ui.Column(row.Context).Padding(padT, padR, padB, padL).Gap(4).Fill()
				messageRoot.Children(func() {
					sessionInspectorChatControl(row.Context, m, msg)
					if control, ok := transcriptControls[msg.ID]; ok {
						transcriptControlView(row.Context, control, m, muted)
						return
					}
					if msg.Kind == session.MessageKindQuestionReply {
						QuestionReplyView(row.Context, msg, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, fg, muted)
						return
					}
					if msg.Kind == session.MessageKindContext {
						ContextInjectionView(row.Context, msg, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, fg, muted)
						return
					}
					if msg.Kind == session.MessageKindSystemPrompt {
						SystemPromptView(row.Context, m, msg, fg, muted)
						return
					}
					if msg.Retry != nil {
						RetryStatusView(row.Context, m, msg, fg, muted)
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
						if detail := terminalReasonDetail(msg); detail != "" {
							ui.Text(row.Context, detail).FontSize(12).TextColor(fg)
						}
						if target, ok := actionTargets[row.Index]; ok {
							assistantMessageActions(row.Context, m, target, messageRoot, muted)
						}
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
							OnOpenURL:     m.OnOpenCitationURL,
							OnOpenFile:    m.OnOpenDeliverable,
							OnInspectCall: m.OnInspectToolCall,
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
							md.RenderFence = func(ctx *ui.Context, language, program string, offset int) bool {
								return openUIFence(ctx, m, msg, language, program, offset)
							}
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
					if target, ok := actionTargets[row.Index]; ok {
						assistantMessageActions(row.Context, m, target, messageRoot, muted)
					}
				})
			}
			if m.FullColumnViewport {
				axis := m.ContentWidth
				if axis <= 0 {
					axis = 748
				}
				// ConversationRoot scrollBody clears 32px on each side, with a
				// stable 5px native scrollbar reservation at the trailing edge.
				ui.Row(row.Context).FillWidth().Padding(0, 37, 0, 32).Children(func() {
					ui.Box(row.Context).Grow(1).MinWidth(0)
					ui.Column(row.Context).FillWidth().MaxWidth(axis).MinWidth(0).Children(paintMessage)
					ui.Box(row.Context).Grow(1).MinWidth(0)
				})
			} else {
				paintMessage()
			}
		})
	})
	// FeedbackDialog's overlay is session-owned, outside the virtualized rows.
	// Scrolling the voted message away must not unmount its open modal.
	if m.FeedbackStore != nil {
		messageFeedbackDialog(c, m)
	}
	if m.ImageLightboxPath != nil && m.OnCloseImageLightbox != nil {
		ImageLightboxOverlay(c, *m.ImageLightboxPath, m.ThemeID, m.SystemDark, m.OnCloseImageLightbox)
	}
}

// StatusView belongs to the conversation shell as well as stand-alone chat
// surfaces: connection failures and action feedback must survive an empty body.
func StatusView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	border := chrome.Color(colors.Border)
	ConnBannerView(c, m.ConnectionBanner, muted, field, border, m.OnReconnect)
	UsageLabelView(c, FormatUsageLabel(m.UsageJobsToday, m.UsageBalance), muted, field, border, m.OnOpenUsage)
	if m.Snapshot.LastAction != "" {
		ui.Text(c, m.Snapshot.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 12, 0, 12)
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
