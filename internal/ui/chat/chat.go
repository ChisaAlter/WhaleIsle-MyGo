// Package chat renders the conversation message list: Markdown, tool cards,
// diffs, thinking blocks, images, feedback, usage and plan panels.
package chat

import (
	"bytes"
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
	nativeUserEditor      bool
	ReasoningBody         func(*ui.Context, session.Message, func())
	RenderSlot            func(*ui.Context, session.ViewState, session.Message, string, func())
	ToolRenderSlot        func(*ui.Context, string, *session.ToolCallView, func())
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
	// TurnContributions adds native entry points; original plugin bodies stay in
	// their retained browser owner. The callback receives the actual closing row.
	TurnContributions func(*ui.Context, session.ViewState, session.Message)
	// OnQuoteMessage inserts a markdown quote of the message into the composer draft.
	OnQuoteMessage       func(body string)
	OnLoadEarlier        func()
	ImageLightboxPath    *string
	OnOpenImageLightbox  func(displayPath string)
	OnCloseImageLightbox func()
	OnOpenCitationURL    func(url string)
	OnOpenPlanResource   func(planpreview.Document)
	// OnOpenDeliverable retains the existing open behavior for message references.
	OnOpenDeliverable func(path string)
	// OnPreviewDeliverable is a separate read-only entry point for an authorized
	// delivery. It must return an error rather than invoking a Host opener.
	OnPreviewDeliverable             func(path string) error
	OnPreviewDeliverablePresentation func(path, presentation string) error
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
	renderAttachmentSnapshot(m.Snapshot, func() { conversationView(c, m) })
}

func conversationView(c *ui.Context, m Model) {
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

	// The snapshot carries the tool/workflow/delivery projection built with it;
	// re-projecting here re-decoded every message on every render pass. Fixtures
	// that assemble a ViewState by hand go through the memo in render_memo.go.
	root := ui.Column(c).Fill().MinHeight(0)
	list := m.List
	if list == nil {
		list = ui.Local(root, "transcript-list", func() ui.ListState { return ui.ListState{FollowEnd: true} })
	}

	projected := projectChatMessages(m.Snapshot)
	viewMemo := ui.Local(root, "transcript-view-memo", func() transcriptViewMemo { return transcriptViewMemo{} })
	msgs, transcriptControls, actionTargets, messageKeys := viewMemo.resolve(m)

	derived := ui.Local(root, "history-derived", func() session.HistoryDerivedMemo { return session.HistoryDerivedMemo{} })
	cards := prepareScheduleContext(m, derived)
	// Index each todo_write call's predecessor from the same loaded event window
	// the transcript draws, as the installed todo history does.
	todoBaselines := projectedTodoBaselines(m.Snapshot)
	historyRows := transcriptHistoryRows(m)
	prepareTranscriptListWithKeys(m, messageKeys, list)
	history := ui.Local(root, "transcript-history", func() transcriptHistoryState { return transcriptHistoryState{} })
	history.prepare(m, msgs, list)
	history.observeInput(c, list)
	prepareHistoryNavigationLanding(m, msgs, list)
	prepareSessionInspector(m, msgs, transcriptControls, list, projected)
	if m.Inspector != nil && m.Inspector.SessionID == m.Snapshot.SelectedID && m.Inspector.Target != nil {
		c.Services().Invalidate()
	}
	root.Children(func() {
		if !m.SuppressStatus {
			StatusView(c, m)
		}
		HistoryNavigationView(c, m, list, muted)
		whaleUIPanelView(c, m)
		ui.Box(c).Fill().Height(1).Background(border)
		if len(msgs) == 0 && historyRows == 0 {
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
		services := c.Services()
		transcript := ui.List(c, list, len(msgs)+historyRows).Fill().ObserveLayout(func(box ui.Rect) {
			history.observeViewport(box)
			if history.requested && !m.Snapshot.HistoryPageBusy {
				services.Invalidate()
			}
		})
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
			if row.Index < historyRows {
				transcriptRowContent(row.Context, m, func() {
					transcriptLoadEarlierView(row.Context, m, history, colors, muted)
				})
				return
			}
			messageIndex := row.Index - historyRows
			ui.Column(row.Context).FillWidth().ObserveLayout(func(box ui.Rect) {
				history.observeMessage(messageKeys[messageIndex], messageIndex, box)
			}).Children(func() {
				paintMessage := func() {
					msg := msgs[messageIndex]
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
							if hasVisibleTurnDelivery(*msg.TurnDelivery, m) && messageIndex > 0 && msgs[messageIndex-1].EventType == "turn/end" {
								if target, ok := actionTargets[messageIndex-1]; ok {
									assistantMessageActions(row.Context, m, target, messageRoot, muted)
								} else {
									TurnUsageView(row.Context, m, msgs[messageIndex-1], muted)
								}
							}
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
							nativeTail := func() {
								if m.TurnContributions != nil {
									m.TurnContributions(row.Context, m.Snapshot, msg)
								}
							}
							if m.RenderSlot != nil {
								m.RenderSlot(row.Context, m.Snapshot, msg, "turnTail", nativeTail)
							} else {
								nativeTail()
							}
							ScheduleTurnCardsView(row.Context, m, cards[msg.Seq])
							key := "turn-end:" + msg.ID
							open := isThinkingExpanded(m.ThinkingExpanded, key)
							detail := terminalReasonDetail(msg)
							if failure, ok := terminalFailure(msg); ok {
								paintTurnFailure(row.Context, failure, colors)
								TurnUsageView(row.Context, m, msg, muted)
								return
							}
							status := turnStatusRowStatus(msg)
							if status == statusDone {
								status = statusNone
							}
							hasTurnHeader := false
							if turn, _, ok := transcriptEventCoordinates(msg); ok {
								turnKey := fmt.Sprintf("transcript-turn:%s:%.0f", m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID, turn)
								_, hasTurnHeader = transcriptControls[turnKey]
							}
							if status == statusNone && detail == "" {
								if !hasTurnHeader {
									ui.Row(row.Context).FillWidth().MinWidth(0).MinHeight(24).AlignItems(ui.Center).Children(func() {
										ui.Text(row.Context, turnStatusLabel(msg)).FontSize(13).FixedLineHeight(24).TextColor(muted).SingleLine()
									})
								}
							} else {
								root := ui.Column(row.Context).FillWidth().MinWidth(0).Gap(2)
								root.Children(func() {
									paintDisclosureRow(row.Context, disclosureRow{
										Key:        key,
										Title:      turnStatusLabel(msg),
										TitleColor: muted,
										Status:     status,
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
							}
							deferredToDelivery := messageIndex+1 < len(msgs) && msgs[messageIndex+1].TurnDelivery != nil && hasVisibleTurnDelivery(*msgs[messageIndex+1].TurnDelivery, m)
							if !deferredToDelivery {
								if target, ok := actionTargets[messageIndex]; ok {
									assistantMessageActions(row.Context, m, target, messageRoot, muted)
								} else {
									TurnUsageView(row.Context, m, msg, muted)
								}
							}
							return
						}
						// Electron chat: system failures as red banner; no role chips on user/assistant.
						if isSystem {
							text, code := splitTurnFailure(msg.Body)
							paintTurnFailure(row.Context, turnFailure{Message: text, Code: code}, colors)
							return
						}
						rowMeta := ui.Local(messageRoot, "row-meta", func() rowContentMemo { return rowContentMemo{} })
						rowMeta.update(msg)
						if rowMeta.hasPublication {
							VisualReplyView(row.Context, rowMeta.publication, m, fg, muted, border, chrome.Color(colors.Field))
						}
						switch msg.Kind {
						case session.MessageKindCommand:
							CommandCardView(row.Context, msg, m, toolCardStyle(colors, fg, muted, border))
						case session.MessageKindTool:
							if whaleUIToolView(row.Context, m, msg) {
								break
							}
							tc := toolCardStyle(colors, fg, muted, border)
							ToolMessageView(row.Context, msg, tc, ToolImageGalleryOptions{
								ID: msg.ID, Cwd: m.Snapshot.SelectedCwd, Expanded: m.ThinkingExpanded, Inspector: inspectorChatNavigation(m),
								OnOpenURL:     m.OnOpenCitationURL,
								OnOpenFile:    m.OnOpenDeliverable,
								OnInspectCall: inspectCallHandler(m.OnInspectToolCall, msg.ID, msg.Seq),
								Todo:          todoBaselineForCall(todoBaselines, msg),
								TodoBaselines: todoBaselines,
								RenderSlot:    m.ToolRenderSlot,
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
							if m.ReasoningBody != nil || m.RenderSlot != nil {
								thSt.Body = func(ctx *ui.Context) {
									native := func() {
										markdown.View(ctx, msg.Body, transcriptMarkdownStyle(muted, muted, chrome.Color(colors.Field), fg, secondaryChatSize(), darkSurface(muted)))
									}
									if m.ReasoningBody != nil {
										m.ReasoningBody(ctx, msg, native)
									} else {
										m.RenderSlot(ctx, m.Snapshot, msg, "reasoning.body", native)
									}
								}
							}
							ThinkingBlockView(row.Context, msg.ID, msg.Body, m.ThinkingExpanded, thSt)
						case session.MessageKindUnknown:
							ui.Text(row.Context, i18n.T(i18n.KeyChatUnknownContent)).TextColor(muted)
							ui.Text(row.Context, string(msg.ContentBlock)).TextColor(fg)
						default:
							body := msg.Body
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
								userMessageView(row.Context, m, msg, colors, rowMeta.userRefs, rowMeta.userBody, onOpenImage)
								break
							}
							var openImage func(string)
							if m.OnOpenImageLightbox != nil || m.ImageLightboxPath != nil {
								openImage = onOpenImage
							}
							if msg.ImageRef != "" {
								ImageInlineView(row.Context, msg.ImageRef, fg, muted, openImage)
							}
							patch := rowMeta.patch
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
								md.Streaming = msg.Streaming
								md.OnOpenURL = m.OnOpenCitationURL
								md.RenderImage = func(ctx *ui.Context, ref, _ string) bool {
									ImageInlineView(ctx, ref, fg, muted, openImage)
									return true
								}
								// Tables size their columns from measured content and
								// only scroll once they no longer fit the message axis.
								if m.ContentWidth > 0 {
									md.AvailableWidth = m.ContentWidth
								}
								md.ResolveFile = projectedFileMentionResolver(m.Snapshot, msg)
								md.OnOpenFile = m.OnOpenDeliverable
								md.RenderFence = func(ctx *ui.Context, language, program string, offset int) bool {
									return openUIFence(ctx, m, msg, language, program, offset)
								}
								markdown.View(row.Context, body, md)
							}
							if msg.Role == "assistant" {
								CitationsView(row.Context, rowMeta.citations, muted, chrome.Color(colors.Accent), m.OnOpenCitationURL)
							}
						}
						if target, ok := actionTargets[messageIndex]; ok {
							assistantMessageActions(row.Context, m, target, messageRoot, muted)
						}
					})
				}
				transcriptRowContent(row.Context, m, paintMessage)
			})
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
	// Host bookkeeping strings (e.g. "workspace ready: <title>") are internal
	// state, not transcript copy; the shell shows user-facing status only.
	if action := displayableAction(m.Snapshot.LastAction); action != "" {
		ui.Text(c, action).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 12, 0, 12)
	} else if notice := conversationSessionNotice(m.Snapshot); notice != "" {
		ui.Text(c, notice).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 12, 0, 12)
	} else if false {
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
	return normalizeConversationNotice(trimmed)
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
	case msg.Kind == session.MessageKindContext, msg.Kind == session.MessageKindCompaction:
		// Context snapshots and compaction rows read as quiet disclosure lines,
		// not as separate full-weight transcript entries; keep their rhythm tight.
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

type transcriptViewMemoKey struct {
	projKey        session.ProjectionKey
	messages       int
	runs           int
	firstID        string
	lastID         string
	binding        string
	selectedID     string
	mode           string
	collapseTiming string
	locale         string
	expandedLen    int
	expandedXOR    uint64
	expandedSum    uint64
}

type transcriptViewMemo struct {
	ready         bool
	prefixGen     uint64
	key           transcriptViewMemoKey
	msgs          []session.Message
	controls      map[string]transcriptControl
	actionTargets map[int]assistantActionTarget
	messageKeys   []transcriptMessageKey
	turnPrefix    transcriptTurnPrefixViewMemo
}

type transcriptTurnPrefixViewMemo struct {
	ready                   bool
	turnPrefixGen           uint64
	visPrefixLen            int
	key                     transcriptViewMemoKey
	prefixMsgs              []session.Message
	prefixKeys              []transcriptMessageKey
	prefixKeySet            map[transcriptMessageKey]struct{}
	workingControls         map[string]transcriptControl
	lastTailControlKeys     []string
	actionTargetsLatest     map[int]assistantActionTarget
	actionTargetsHistorical map[int]assistantActionTarget
	msgsBuf                 []session.Message
	keysBuf                 []transcriptMessageKey
}

func (k transcriptViewMemoKey) sameExceptProjKey(o transcriptViewMemoKey) bool {
	k.projKey = session.ProjectionKey{}
	o.projKey = session.ProjectionKey{}
	return k == o
}

func (k transcriptViewMemoKey) samePrefixConfig(o transcriptViewMemoKey) bool {
	return k.runs == o.runs &&
		k.firstID == o.firstID &&
		k.binding == o.binding &&
		k.selectedID == o.selectedID &&
		k.mode == o.mode &&
		k.collapseTiming == o.collapseTiming &&
		k.locale == o.locale &&
		k.expandedLen == o.expandedLen &&
		k.expandedXOR == o.expandedXOR &&
		k.expandedSum == o.expandedSum
}

func canReuseStreamingTranscriptTail(prev, curr session.Message) bool {
	return prev.Streaming && curr.Streaming &&
		prev.Kind == session.MessageKindText && curr.Kind == session.MessageKindText &&
		prev.Role == "assistant" && curr.Role == "assistant" &&
		prev.Seq == 0 && curr.Seq == 0 &&
		len(prev.Raw) == 0 && len(curr.Raw) == 0 &&
		prev.Tool == nil && curr.Tool == nil &&
		prev.ID == curr.ID && prev.TurnSeq == curr.TurnSeq && prev.EventType == curr.EventType
}

func makeTranscriptViewMemoKey(m Model) (transcriptViewMemoKey, bool) {
	projKey := m.Snapshot.HistoryProjectionKey()
	if !projKey.Valid() {
		return transcriptViewMemoKey{}, false
	}
	var firstID, lastID string
	if n := len(m.Snapshot.Messages); n > 0 {
		firstID = m.Snapshot.Messages[0].ID
		lastID = m.Snapshot.Messages[n-1].ID
	}
	var expLen int
	var expXOR, expSum uint64
	if m.ThinkingExpanded != nil && *m.ThinkingExpanded != nil {
		expLen = len(*m.ThinkingExpanded)
		for k, v := range *m.ThinkingExpanded {
			h := uint64(14695981039346656037)
			for i := 0; i < len(k); i++ {
				h ^= uint64(k[i])
				h *= 1099511628211
			}
			if v {
				h ^= 1
			} else {
				h ^= 2
			}
			h *= 1099511628211
			expXOR ^= h
			expSum += h
		}
	}
	return transcriptViewMemoKey{
		projKey:        projKey,
		messages:       len(m.Snapshot.Messages),
		runs:           len(m.Snapshot.WorkflowRuns),
		firstID:        firstID,
		lastID:         lastID,
		binding:        m.Snapshot.PlanHistoryBinding,
		selectedID:     m.Snapshot.SelectedID,
		mode:           m.Preferences.TranscriptView,
		collapseTiming: m.Preferences.CollapseTiming,
		locale:         i18n.Locale(),
		expandedLen:    expLen,
		expandedXOR:    expXOR,
		expandedSum:    expSum,
	}, true
}

// resolve returns the rows to draw. The slices belong to the memo: while a reply
// streams, the last row is replaced in place on the next revision, so they are
// read within the pass on the UI thread and copied before any goroutine gets them.
func (memo *transcriptViewMemo) resolve(m Model) ([]session.Message, map[string]transcriptControl, map[int]assistantActionTarget, []transcriptMessageKey) {
	key, cacheable := makeTranscriptViewMemoKey(m)
	if cacheable && memo.ready && memo.key == key {
		return memo.msgs, memo.controls, memo.actionTargets, memo.messageKeys
	}
	visible := visibleProjectedMessages(m.Snapshot)
	prefixGen := chatProjectionOf(m.Snapshot).revisionPrefixGen(m.Snapshot)
	if cacheable && memo.ready && prefixGen > 0 && memo.prefixGen == prefixGen &&
		key.sameExceptProjKey(memo.key) && len(visible) > 0 && len(memo.msgs) > 0 &&
		canReuseStreamingTranscriptTail(memo.msgs[len(memo.msgs)-1], visible[len(visible)-1]) {
		memo.msgs[len(memo.msgs)-1] = visible[len(visible)-1]
		memo.key = key
		return memo.msgs, memo.controls, memo.actionTargets, memo.messageKeys
	}
	turnPrefixGen, visPrefixLen := chatProjectionOf(m.Snapshot).revisionTurnPrefix(m.Snapshot)
	if cacheable && turnPrefixGen > 0 && visPrefixLen > 0 && visPrefixLen <= len(visible) &&
		visible[visPrefixLen-1].Kind != session.MessageKindThinking &&
		visible[visPrefixLen-1].Kind != session.MessageKindTool {
		tp := &memo.turnPrefix
		if !tp.ready || tp.turnPrefixGen != turnPrefixGen || tp.visPrefixLen != visPrefixLen || !key.samePrefixConfig(tp.key) {
			visPrefix := visible[:visPrefixLen]
			prefixMsgs, prefixControls := projectTranscript(visPrefix, m)
			latestTargets, historicalTargets := assistantActionTargetsForTurnPrefix(prefixMsgs)
			prefixKeys := transcriptMessageKeys(m, prefixMsgs)
			prefixKeySet := make(map[transcriptMessageKey]struct{}, len(prefixKeys))
			for _, pk := range prefixKeys {
				pk.Occurrence = 0
				prefixKeySet[pk] = struct{}{}
			}
			msgsBuf := make([]session.Message, len(prefixMsgs), len(prefixMsgs)+32)
			copy(msgsBuf, prefixMsgs)
			keysBuf := make([]transcriptMessageKey, len(prefixKeys), len(prefixKeys)+32)
			copy(keysBuf, prefixKeys)
			*tp = transcriptTurnPrefixViewMemo{
				ready:                   true,
				turnPrefixGen:           turnPrefixGen,
				visPrefixLen:            visPrefixLen,
				key:                     key,
				prefixMsgs:              prefixMsgs,
				prefixKeys:              prefixKeys,
				prefixKeySet:            prefixKeySet,
				workingControls:         prefixControls,
				actionTargetsLatest:     latestTargets,
				actionTargetsHistorical: historicalTargets,
				msgsBuf:                 msgsBuf,
				keysBuf:                 keysBuf,
			}
		}
		for _, k := range tp.lastTailControlKeys {
			delete(tp.workingControls, k)
		}
		tp.lastTailControlKeys = tp.lastTailControlKeys[:0]

		visTail := visible[visPrefixLen:]
		tailMsgs, tailControls := projectTranscript(visTail, m)
		for k, v := range tailControls {
			if _, inPrefix := tp.workingControls[k]; inPrefix {
				continue
			}
			tp.workingControls[k] = v
			tp.lastTailControlKeys = append(tp.lastTailControlKeys, k)
		}

		prevMsgsLen := len(tp.msgsBuf)
		needMsgs := len(tp.prefixMsgs) + len(tailMsgs)
		if cap(tp.msgsBuf) < needMsgs {
			tp.msgsBuf = make([]session.Message, needMsgs, needMsgs+32)
			copy(tp.msgsBuf, tp.prefixMsgs)
		} else if needMsgs < prevMsgsLen {
			clear(tp.msgsBuf[needMsgs:prevMsgsLen])
		}
		tp.msgsBuf = tp.msgsBuf[:needMsgs]
		copy(tp.msgsBuf[len(tp.prefixMsgs):], tailMsgs)

		targets := combineTurnPrefixActionTargets(
			tp.actionTargetsLatest,
			tp.actionTargetsHistorical,
			len(tp.prefixMsgs),
			tailMsgs,
		)

		tailKeys := transcriptMessageKeys(m, tailMsgs)
		keyCollision := false
		for _, tk := range tailKeys {
			probe := tk
			probe.Occurrence = 0
			if _, exists := tp.prefixKeySet[probe]; exists {
				keyCollision = true
				break
			}
		}
		var keys []transcriptMessageKey
		if keyCollision {
			keys = transcriptMessageKeys(m, tp.msgsBuf)
		} else {
			needKeys := len(tp.prefixKeys) + len(tailKeys)
			if cap(tp.keysBuf) < needKeys {
				tp.keysBuf = make([]transcriptMessageKey, needKeys, needKeys+32)
				copy(tp.keysBuf, tp.prefixKeys)
			}
			tp.keysBuf = tp.keysBuf[:needKeys]
			copy(tp.keysBuf[len(tp.prefixKeys):], tailKeys)
			keys = tp.keysBuf
		}

		memo.ready = true
		memo.prefixGen = prefixGen
		memo.key = key
		memo.msgs = tp.msgsBuf
		memo.controls = tp.workingControls
		memo.actionTargets = targets
		memo.messageKeys = keys
		return memo.msgs, memo.controls, memo.actionTargets, memo.messageKeys
	}
	msgs, controls := projectTranscript(visible, m)
	targets := assistantActionTargets(msgs)
	keys := transcriptMessageKeys(m, msgs)
	if cacheable {
		*memo = transcriptViewMemo{
			ready:         true,
			prefixGen:     prefixGen,
			key:           key,
			msgs:          msgs,
			controls:      controls,
			actionTargets: targets,
			messageKeys:   keys,
		}
	}
	return msgs, controls, targets, keys
}

type rowContentMemo struct {
	ready          bool
	id             string
	seq            float64
	role           string
	kind           session.MessageKind
	streaming      bool
	body           string
	raw            []byte
	hasWeb         bool
	webURLs        int
	publication    session.VisualReplyPublication
	hasPublication bool
	userRefs       []string
	userBody       string
	patch          string
	citations      []Citation
}

func (memo *rowContentMemo) matches(msg session.Message) bool {
	if !memo.ready || memo.id != msg.ID || memo.seq != msg.Seq || memo.role != msg.Role ||
		memo.kind != msg.Kind || memo.streaming != msg.Streaming || memo.body != msg.Body {
		return false
	}
	hasWeb := msg.Web != nil
	webURLs := 0
	if hasWeb {
		webURLs = len(msg.Web.Sources)
		if msg.Web.URL != "" {
			webURLs++
		}
	}
	if memo.hasWeb != hasWeb || memo.webURLs != webURLs {
		return false
	}
	if len(memo.raw) != len(msg.Raw) {
		return false
	}
	if len(memo.raw) > 0 && &memo.raw[0] != &msg.Raw[0] && !bytes.Equal(memo.raw, msg.Raw) {
		return false
	}
	return true
}

func (memo *rowContentMemo) update(msg session.Message) {
	if memo.matches(msg) {
		return
	}
	pub, hasPub := session.PublishedVisualReply(msg.Raw)
	var userRefs []string
	var userBody, patch string
	var citations []Citation
	switch msg.Kind {
	case session.MessageKindCommand, session.MessageKindTool, session.MessageKindDeliverables,
		session.MessageKindCompaction, session.MessageKindThinking, session.MessageKindUnknown:
	default:
		if msg.Role == "user" {
			userRefs, userBody = userMessageParts(msg.Body)
		} else {
			patch = diff.ExtractPatchText(msg.Body)
			if msg.Role == "assistant" {
				citations = CitationsForMessage(msg)
			}
		}
	}
	hasWeb := msg.Web != nil
	webURLs := 0
	if hasWeb {
		webURLs = len(msg.Web.Sources)
		if msg.Web.URL != "" {
			webURLs++
		}
	}
	*memo = rowContentMemo{
		ready:          true,
		id:             msg.ID,
		seq:            msg.Seq,
		role:           msg.Role,
		kind:           msg.Kind,
		streaming:      msg.Streaming,
		body:           msg.Body,
		raw:            msg.Raw,
		hasWeb:         hasWeb,
		webURLs:        webURLs,
		publication:    pub,
		hasPublication: hasPub,
		userRefs:       userRefs,
		userBody:       userBody,
		patch:          patch,
		citations:      citations,
	}
}
