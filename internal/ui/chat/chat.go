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
	// OnInspectToolCall opens the session inspector at the record that carries a
	// tool call. The message ID and sequence travel with the call ID so the
	// inspector can select the exact log row instead of guessing.
	OnInspectToolCall     func(callID, messageID string, seq float64)
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
	Scroll             *ui.ScrollState
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
	OnOpenDeliverable func(path string)
	// Starts on the UI thread and completes on the UI thread. The local
	// reference owns failure/retry presentation; later preview loading does not.
	OnOpenHistoryReference       func(path string, complete func(error))
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
		if m.Inspector != nil {
			m.Inspector.CancelPicking()
			m.Inspector.CancelReveal()
		}
		groupThreadsView(c, m)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	dark := th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark)
	// Let the frame desk-wash show through (Electron chat pane is translucent mica).
	_ = colors.BG

	msgs := session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(m.Snapshot.Messages), m.Snapshot.WorkflowRuns)))
	projected := msgs
	msgs, transcriptControls := projectTranscript(visibleChatMessages(msgs), m)
	actionTargets := assistantActionTargets(msgs)
	list := m.List
	if list == nil {
		list = &ui.ListState{FollowEnd: true}
	}

	cards := prepareScheduleContext(m)
	prepareHistoryNavigation(m, msgs, list)
	prepareSessionInspector(m, msgs, transcriptControls, list, projected)
	if m.Inspector != nil && m.Inspector.SessionID == m.Snapshot.SelectedID && m.Inspector.Target != nil {
		c.Services().Invalidate()
	}
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		if !m.SuppressStatus {
			StatusView(c, m)
		}
		HistoryNavigationView(c, m, list, muted)
		ui.Box(c).Fill().Height(1).Background(border)
		if ShowLoadEarlier(m.Snapshot) && m.OnLoadEarlier != nil {
			label := i18n.T(i18n.KeyChatLoadEarlier)
			if m.Snapshot.HistoryPageBusy {
				label = i18n.T(i18n.KeyChatLoadEarlierBusy)
			}
			ui.Row(c).Padding(8, 12, 0, 12).Children(func() {
				if stdButton(c, colors, label, buttonGhost, m.Snapshot.HistoryPageBusy).Clicked() {
					m.OnLoadEarlier()
				}
			})
		}
		if len(msgs) == 0 {
			if m.Inspector != nil {
				m.Inspector.CancelPicking()
				m.Inspector.CancelReveal()
			}
			// The empty Session is the Hero's job (conversationPhase), so this
			// only paints for a standalone surface or an errored settle: center
			// the hint instead of pinning a 13px line to the top-left corner.
			if m.Snapshot.HistoryError == "" {
				ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyChatEmpty)).FontSize(theme.InterfaceSize(14)).TextColor(muted)
				})
			}
			ui.Box(c).Grow(1) // keep composer pinned to the bottom of the dialogue column
			return
		}
		transcript := ui.List(c, list, len(msgs)).Fill()
		scroll := m.Scroll
		if scroll == nil {
			scroll = ui.Local(transcript, "inspector-scroll", func() ui.ScrollState { return ui.ScrollState{} })
		}
		transcript.TrackScroll(scroll)
		inspectorChatViewport(c, m, transcript, list)
		if m.FullColumnViewport {
			transcript.ScrollbarInsets(0, -3, 0, 0)
		}
		inspectorChatPickerOverlay(c, m, colors, list, scroll)
		transcript.Rows(func(row ui.ListRow) {
			paintMessage := func() {
				msg := msgs[row.Index]
				// The transcript column already clears the Original 32 DIP side
				// gutter when it owns the scrollbar, so a row spends its padding
				// vertically only there: process rows sit 6 apart, an answer 12,
				// a turn header 16. A standalone surface keeps its own 16 DIP side
				// inset because no column provides one.
				padT, padB := messageSpacing(msg)
				padL, padR := float32(16), float32(16)
				if m.FullColumnViewport {
					padL, padR = 0, 0
				}
				isSystem := msg.Kind == session.MessageKindText && msg.Role == "system"
				messageRoot := ui.Column(row.Context).Padding(padT, padR, padB, padL).Gap(4).Fill()
				inspectorChatRowGeometry(row.Context, m, msg, messageRoot, padT, padR, padB, padL)
				messageRoot.Children(func() {
					if control, ok := transcriptControls[msg.ID]; ok {
						transcriptControlView(row.Context, control, m, muted)
						return
					}
					if msg.Kind == session.MessageKindQuestionReply {
						QuestionReplyView(row.Context, msg, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, colors, fg, muted)
						return
					}
					if msg.Kind == session.MessageKindContext {
						if msg.ContextWaking {
							TurnTriggerView(row.Context, msg, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, colors)
						} else {
							ContextInjectionView(row.Context, msg, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, colors)
						}
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
						TurnDeliveryView(row.Context, *msg.TurnDelivery, m, toolCardStyle(colors, fg, muted, border))
						return
					}
					if msg.Workflow != nil {
						WorkflowRunView(row.Context, *msg.Workflow, m, toolCardStyle(colors, fg, muted, border))
						return
					}
					if msg.Kind == session.MessageKindSubmittedPlan {
						SubmittedPlanCardView(row.Context, msg, m.Snapshot.SelectedID, planpreview.Style{FG: fg, Muted: muted, Border: border, Field: chrome.Color(colors.Field), Accent: chrome.Color(colors.Accent)}, m.OnOpenPlanResource)
						return
					}
					// Electron turn/end strip: 「已完成，用时 …」 with the usage row
					// below it. The store writes elapsed milliseconds into Body.
					if msg.Role == "status" && msg.EventType == "turn/end" {
						ScheduleTurnCardsView(row.Context, m, cards[msg.Seq])
						key := "turn-end:" + msg.ID
						open := isThinkingExpanded(m.ThinkingExpanded, key)
						detail := terminalReasonDetail(msg)
						if failure, ok := terminalFailure(msg); ok {
							paintTurnFailure(row.Context, failure, colors)
							TurnUsageView(row.Context, m, msg, muted)
							return
						}
						root := ui.Column(row.Context).FillWidth().MinWidth(0).Gap(2)
						root.Children(func() {
							paintDisclosureRow(row.Context, disclosureRow{
								Key:        key,
								Title:      turnStatusLabel(msg),
								TitleColor: muted,
								Status:     turnStatusRowStatus(msg),
								Expandable: detail != "",
								Open:       open,
								OnToggle:   func() { toggleThinkingExpanded(m.ThinkingExpanded, key) },
							}, disclosureStyle{FG: fg, Muted: muted, Hover: chrome.Color(colors.Hover), Accent: chrome.Color(colors.Accent), Size: 12})
							if open && detail != "" {
								ui.Column(row.Context).Padding(0, 0, 2, 22).Children(func() {
									ui.Text(row.Context, detail).FontSize(12).TextColor(fg)
								})
							}
						})
						if target, ok := actionTargets[row.Index]; ok {
							assistantMessageActions(row.Context, m, target, messageRoot, muted)
						}
						TurnUsageView(row.Context, m, msg, muted)
						return
					}
					// Electron chat: system failures as red banner; no role chips on user/assistant.
					if isSystem {
						text, code := splitTurnFailure(msg.Body)
						paintTurnFailure(row.Context, turnFailure{Message: text, Code: code}, colors)
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
						CommandCardView(row.Context, msg, m, toolCardStyle(colors, fg, muted, border))
					case session.MessageKindTool:
						tc := DefaultToolCardStyle(fg, muted, border, chrome.Color(colors.Field), chrome.Color(colors.Accent))
						ToolMessageView(row.Context, msg, tc, ToolImageGalleryOptions{
							ID: msg.ID, Expanded: m.ThinkingExpanded,
							OnOpenURL:     m.OnOpenCitationURL,
							OnOpenFile:    m.OnOpenDeliverable,
							OnInspectCall: inspectCallHandler(m.OnInspectToolCall, msg.ID, msg.Seq),
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
							Streaming:   msg.Streaming,
							FG:          fg,
							Muted:       muted,
							Border:      border,
							Field:       chrome.Color(colors.Field),
							Size:        secondaryChatSize(),
							Accent:      chrome.Color(colors.Accent),
							Hover:       chrome.Color(colors.Hover),
						}
						ThinkingBlockView(row.Context, msg.ID, msg.Body, m.ThinkingExpanded, thSt)
					case session.MessageKindUnknown:
						ui.Text(row.Context, i18n.T(i18n.KeyChatUnknownContent)).TextColor(muted)
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
							md := transcriptMarkdownStyle(fg, muted, chrome.Color(colors.Field), chrome.Color(colors.Accent), theme.CurrentPreferences().Chat.Size, dark)
							md.TableHeader = chrome.Color(colors.Sidebar)
							// Tables size their columns from measured content and
							// only scroll once they no longer fit the message axis.
							if m.ContentWidth > 0 {
								md.AvailableWidth = m.ContentWidth
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
	inspectorChatRevealOverlay(c, m, colors)
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
	ConnBannerView(c, colors, m.ConnectionBanner, muted, field, border, m.OnReconnect)
	UsageLabelView(c, FormatUsageLabel(m.UsageJobsToday, m.UsageBalance), muted, field, border, m.OnOpenUsage)
	// Host bookkeeping strings (e.g. "workspace ready: <title>") are internal
	// state, not transcript copy; the shell shows user-facing status only.
	if action := displayableAction(m.Snapshot.LastAction); action != "" {
		ui.Text(c, action).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 12, 0, 12)
	}
}

// internalActionPrefixes are Host-side bookkeeping phrases that must never be
// painted as conversation status.
var internalActionPrefixes = []string{
	"workspace ready:",
	"workspace ready",
	"session ready:",
	"session ready",
	"session created",
	"session deleted",
	"session archived",
	"session unarchived",
	"blank session",
	"agent preset selected",
	"prompt accepted",
	"goal created",
	"goal edited",
	"creator task:",
	"feedback saved",
	"feedback:",
	"band ",
	"session forked",
	"session pinned",
	"session renamed",
	"session reordered",
	"session unpinned",
	"prompt edited",
	"prompt sent",
	"cancel accepted",
	"cancel rejected",
	"interrupt accepted",
	"file uploaded",
	"skill created",
	"skill update",
	"skill updated:",
	"skill invocation",
	"skill hub",
	"skill directory opened",
	"speech:",
	"speech catalog:",
	"speech configure:",
	"subagent prompt sent",
	"session feedback recorded",
	"goal action returned",
	"current host",
}

// displayableAction drops internal status phrases and keeps user-facing ones.
func displayableAction(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	for _, prefix := range internalActionPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	return trimmed
}

// messageSpacing grades the vertical rhythm of one transcript row. The side
// gutter is the column's job; this only owns the block gap above and below, as
// the Original flow does (process rows 6, an answer 12, a turn header 16).
func messageSpacing(msg session.Message) (top, bottom float32) {
	switch {
	case msg.Role == "user":
		return 16, 2
	case msg.Role == "status":
		return 16, 2
	case msg.Kind == session.MessageKindContext:
		// Context snapshots read as one quiet disclosure stack, not as
		// separate transcript entries; keep their rhythm tight.
		return 4, 0
	case msg.Kind == session.MessageKindThinking, msg.Kind == session.MessageKindTool:
		return 6, 0
	case msg.Role == "assistant" && msg.Kind == session.MessageKindText:
		return 12, 4
	default:
		return 12, 2
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

// inspectCallHandler binds one tool call to the message that carries it. The
// inspector selects log rows by chat message ID (or sequence), so the call ID
// alone would not be enough to open the right record. Nil keeps the card's
// inspect action hidden rather than offering a dead button.
func inspectCallHandler(inspect func(callID, messageID string, seq float64), messageID string, seq float64) func(string) {
	if inspect == nil {
		return nil
	}
	return func(callID string) {
		inspect(callID, messageID, seq)
	}
}
