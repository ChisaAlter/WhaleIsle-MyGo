package chat

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// MessageIconActions.module.css shares a 28px seat and an 8px gap; earlier
// user rows and turn tails reveal on hover/focus without changing geometry.
func messageAction(c *ui.Context, icon *ui.SVG, label string, assistant bool, muted ui.Color, semanticID ...string) ui.Element {
	delta := theme.CurrentPreferences().Chat.Size - 14
	edge, glyph := float32(28)+delta, float32(15)+delta
	if assistant {
		glyph += 2
	}
	key := label
	if len(semanticID) > 0 {
		key = semanticID[0]
	}
	b := ui.Button(c.Key("message-action-"+key), "").Label(label).Tooltip(label).Width(edge).Height(edge).Padding(0).
		Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).
		Justify(ui.Center).AlignItems(ui.Center)
	if b.Hovered() {
		b.Background(c.Theme().SurfaceHover)
	}
	b.Children(func() { ui.Icon(c, icon).Width(glyph).Height(glyph).PassThrough() })
	return b
}

func messageCopyAction(c *ui.Context, msg session.Message, assistant bool, muted ui.Color) {
	seat := ui.Box(c.Key("copy-action"))
	copiedUntil := ui.Local(seat, "copied-until", func() time.Time { return time.Time{} })
	seat.Children(func() {
		icon := messageIconCopyOutline
		if time.Now().Before(*copiedUntil) {
			icon = messageIconCheckOutline
		}
		b := messageAction(c, icon, i18n.T(i18n.KeyChatCopy), assistant, muted)
		if b.Clicked() && time.Now().After(*copiedUntil) {
			c.Services().WriteClipboard(msg.Body)
			*copiedUntil = time.Now().Add(time.Second)
		}
		if time.Now().Before(*copiedUntil) {
			b.DrawOver(func(p *ui.Painter, _ ui.Rect) {
				if time.Now().After(*copiedUntil) {
					c.Services().Invalidate()
				} else {
					p.After(time.Until(*copiedUntil))
				}
			})
		}
	})
}

type assistantActionTarget struct {
	message session.Message
	endSeq  float64
	latest  bool
}

// TurnTailNodeView owns completed assistant actions, not each intermediate
// text block. Legacy transcripts without turn/end retain a final-message seat.
func assistantActionTargets(messages []session.Message) map[int]assistantActionTarget {
	out, _ := assistantActionTargetsWithLastSeat(messages)
	return out
}

func assistantActionTargetsWithLastSeat(messages []session.Message) (map[int]assistantActionTarget, int) {
	out := make(map[int]assistantActionTarget)
	last := -1
	lastSeat := -1
	// Only projected lifecycles need their text assembled. Build prefix text in
	// one pass; a candidate retains the prefix at its own occurrence, before any
	// later user/turn row can append text with the same source ID.
	textBySource := make(map[string]*strings.Builder)
	for _, msg := range messages {
		if msg.SourceMessageID != "" && msg.Role == "assistant" && showMessageCopy(msg) {
			if textBySource[msg.SourceMessageID] == nil {
				textBySource[msg.SourceMessageID] = &strings.Builder{}
			}
		}
	}
	lastBody := ""
	flush := func(seat int, seq float64) {
		if last < 0 {
			return
		}
		message := messages[last]
		// Snapshot projects one durable Assistant lifecycle into text/reasoning
		// block rows. Copy all its visible text, as turn-assistant.assistantText
		// does, while keeping earlier tool-step Assistant lifecycles separate.
		if message.SourceMessageID != "" {
			message.Body = lastBody
		}
		out[seat] = assistantActionTarget{message: message, endSeq: seq}
		lastSeat = seat
		last = -1
	}
	for index, msg := range messages {
		if msg.Role == "user" {
			flush(last, 0)
		}
		copyable := showMessageCopy(msg)
		if copyable {
			if text := textBySource[msg.FeedbackMessageID()]; text != nil {
				text.WriteString(msg.Body)
			}
		}
		if msg.Role == "assistant" && copyable {
			last = index
			if msg.SourceMessageID != "" {
				lastBody = textBySource[msg.SourceMessageID].String()
			}
		}
		if msg.EventType == "turn/end" {
			flush(index, msg.Seq)
		}
	}
	flush(last, 0)
	if target, ok := out[lastSeat]; ok {
		// A later user/steering entry makes this a historical tail.
		target.latest = true
		for _, msg := range messages[lastSeat+1:] {
			if msg.Kind == session.MessageKindTurnDelivery || msg.TurnDelivery != nil {
				continue
			}
			if msg.Role == "user" || msg.Role == "assistant" {
				target.latest = false
				break
			}
		}
		out[lastSeat] = target
	}
	return out, lastSeat
}

func assistantActionTargetsForTurnPrefix(messages []session.Message) (map[int]assistantActionTarget, map[int]assistantActionTarget) {
	latestMap, lastSeat := assistantActionTargetsWithLastSeat(messages)
	if target, ok := latestMap[lastSeat]; ok && target.latest {
		historicalMap := make(map[int]assistantActionTarget, len(latestMap))
		for k, v := range latestMap {
			historicalMap[k] = v
		}
		target.latest = false
		historicalMap[lastSeat] = target
		return latestMap, historicalMap
	}
	return latestMap, latestMap
}

func combineTurnPrefixActionTargets(
	latestMap map[int]assistantActionTarget,
	historicalMap map[int]assistantActionTarget,
	prefixLen int,
	tailMsgs []session.Message,
) map[int]assistantActionTarget {
	if len(tailMsgs) == 0 {
		return latestMap
	}
	tailTargets, _ := assistantActionTargetsWithLastSeat(tailMsgs)
	if len(tailTargets) == 0 {
		for _, msg := range tailMsgs {
			if msg.Kind == session.MessageKindTurnDelivery || msg.TurnDelivery != nil {
				continue
			}
			if msg.Role == "user" || msg.Role == "assistant" {
				return historicalMap
			}
		}
		return latestMap
	}
	out := make(map[int]assistantActionTarget, len(historicalMap)+len(tailTargets))
	for k, v := range historicalMap {
		out[k] = v
	}
	for k, v := range tailTargets {
		out[prefixLen+k] = v
	}
	return out
}

func assistantMessageActions(c *ui.Context, m Model, target assistantActionTarget, owner ui.Element, muted ui.Color) {
	if m.RenderSlot != nil {
		native := m
		native.RenderSlot = nil
		m.RenderSlot(c, m.Snapshot, target.message, "assistant-actions", func() { assistantMessageActions(c, native, target, owner, muted) })
		return
	}
	msg := target.message
	row := ui.Row(c).Key("assistant-actions").Label("Assistant message actions").Gap(8).
		Height(28+theme.CurrentPreferences().Chat.Size-14).AlignItems(ui.Center).Margin(4, 0, 0, -6)
	messageActionMenu(c, row, m, msg)
	revealed := target.latest || owner.Hovered() || owner.FocusWithin()
	row.Children(func() {
		copySeat := ui.Box(c.Key("copy-reveal"))
		if !revealed {
			copySeat.Opacity(0)
		}
		copySeat.Children(func() { messageCopyAction(c, msg, true, muted) })
		if showMessageFeedback(msg) && m.FeedbackStore != nil {
			authoritativeFeedbackActions(c, m.FeedbackStore, msg.FeedbackMessageID(), muted, revealed)
		}
		if target.endSeq > 0 && m.OnForkTurn != nil {
			allowed := canForkAssistantTarget(m.Snapshot.Messages, target) && !m.Snapshot.HistoryBranchBusy && m.Snapshot.ClientReady
			b := messageAction(c, messageIconBranchOutline, branchActionLabel(), true, muted)
			if !allowed {
				b.Opacity(.4)
			}
			if !revealed {
				b.Opacity(0)
			}
			if allowed && b.Clicked() {
				m.OnForkTurn(target.endSeq)
			}
		}
		if target.endSeq > 0 {
			TurnUsageView(c, m, session.Message{Seq: target.endSeq}, muted)
		}
		if msg.EventTime > 0 {
			stamp := ui.Text(c, formatMessageClock(msg.EventTime, c.Now())).FontSize(theme.CurrentPreferences().Chat.Size-2).FixedLineHeight(theme.CurrentPreferences().Chat.Size+10).Margin(0, 0, 0, 8).TextColor(muted)
			if !revealed {
				stamp.Opacity(0)
			}
		}
	})
	if showMessageFeedback(msg) && m.FeedbackStore != nil {
		authoritativeFeedbackStatus(c, m.FeedbackStore, muted)
	}
}

// The Store validates its canonical journal. Snapshot instead splits a durable
// assistant/message into text/image/structured block rows with one source ID
// and seq. Reconstitute only that same lifecycle before using the existing
// boundary guard, so a trailing image block does not impersonate a later reply.
// A separate later image event, tool result or failed boundary stays intact.
func canForkAssistantTarget(messages []session.Message, target assistantActionTarget) bool {
	if session.CanForkTurn(messages, target.endSeq) {
		return true
	}
	message := target.message
	if message.EventType != "assistant/message" || message.SourceMessageID == "" {
		return false
	}
	canonical := make([]session.Message, 0, len(messages))
	inserted := false
	for _, block := range messages {
		if block.SourceMessageID == message.SourceMessageID && block.Seq == message.Seq && block.EventType == message.EventType {
			message.Streaming = message.Streaming || block.Streaming
			if !inserted {
				canonical = append(canonical, message)
				inserted = true
			}
			continue
		}
		canonical = append(canonical, block)
	}
	if message.Streaming {
		return false
	}
	return session.CanForkTurn(canonical, target.endSeq)
}

// Keep the old port's quote command accessible by context/menu key without
// adding a control to MessageIconActions' original copy/feedback/branch row.
// Copy stays available on the same menu so a keyboard/context-menu user never
// gets a menu that only quotes (or no menu at all when quoting is unavailable).
func messageActionMenu(c *ui.Context, row ui.Element, m Model, msg session.Message) {
	if !showMessageCopy(msg) {
		return
	}
	row.ContextMenu(func(menu *ui.Menu) {
		if menu.Item(i18n.T(i18n.KeyChatCopy)).Chosen() {
			c.Services().WriteClipboard(msg.Body)
		}
		if m.OnQuoteMessage != nil && menu.Item(i18n.T(i18n.KeyChatQuote)).Chosen() {
			m.OnQuoteMessage(msg.Body)
		}
	})
}

func branchActionLabel() string {
	if i18n.Locale() == i18n.LocaleEN {
		return "Branch in a new conversation"
	}
	return "在新对话中分支"
}

// messageEditText returns verbatim zh/en copy from dsh-client-ui-message-edit/locales.ts.
func messageEditText(key string) string {
	copy := map[string][2]string{
		"edit":                {"编辑", "Edit"},
		"action.edit":         {"编辑", "Edit"},
		"running":             {"等待当前回复结束后编辑", "Wait for the current response to finish before editing"},
		"action.running":      {"等待当前回复结束后编辑", "Wait for the current response to finish before editing"},
		"unsupported":         {"包含非文本内容，暂不支持编辑", "Messages with non-text content cannot be edited yet"},
		"action.unsupported":  {"包含非文本内容，暂不支持编辑", "Messages with non-text content cannot be edited yet"},
		"cancel":              {"取消", "Cancel"},
		"action.cancel":       {"取消", "Cancel"},
		"banner":              {"正在重新编辑此消息", "Re-editing this message"},
		"editor.banner":       {"正在重新编辑此消息", "Re-editing this message"},
		"editing":             {"正在下方输入框中重新编辑", "Re-editing in the input box below"},
		"editor.editing":      {"正在下方输入框中重新编辑", "Re-editing in the input box below"},
		"hint.running":        {"当前回复尚未结束，结束后可发送", "The current response has not finished; send once it settles"},
		"editor.hint.running": {"当前回复尚未结束，结束后可发送", "The current response has not finished; send once it settles"},
		"stale":               {"会话已有更新的消息，此条不能再重新发送", "Newer messages arrived, so this one can no longer be resent"},
		"editor.hint.stale":   {"会话已有更新的消息，此条不能再重新发送", "Newer messages arrived, so this one can no longer be resent"},
		"busy":                {"输入框正忙，暂时无法编辑", "The input box is busy; editing is unavailable right now"},
		"error.busy":          {"输入框正忙，暂时无法编辑", "The input box is busy; editing is unavailable right now"},
		"error":               {"无法重新发送编辑后的消息，请重试", "Could not resend the edited message. Try again."},
		"error.generic":       {"无法重新发送编辑后的消息，请重试", "Could not resend the edited message. Try again."},
	}
	if i18n.Locale() == i18n.LocaleEN {
		return copy[key][1]
	}
	return copy[key][0]
}

// messageEditStatus resolves the edit status copy for a user message row.
func messageEditStatus(snap session.ViewState, msg session.Message, editingSeq float64) string {
	editSeq, editTextOnly := projectedLatestEditable(snap)
	if editingSeq > 0 && msg.Seq == editingSeq {
		if editSeq != msg.Seq {
			return messageEditText("stale")
		}
		if snap.PromptBusy {
			return messageEditText("hint.running")
		}
		if snap.MessageEditBusy {
			return messageEditText("busy")
		}
		return messageEditText("editing")
	}
	if msg.Seq <= 0 || msg.Seq != editSeq {
		if msg.Seq > 0 && editSeq > msg.Seq {
			return messageEditText("stale")
		}
		return ""
	}
	if snap.PromptBusy {
		return messageEditText("running")
	}
	if snap.MessageEditBusy {
		return messageEditText("busy")
	}
	if !editTextOnly || msg.ImageRef != "" {
		return messageEditText("unsupported")
	}
	if refs, _ := userMessageParts(msg.Body); len(refs) > 0 {
		return messageEditText("unsupported")
	}
	return messageEditText("edit")
}

func unusedBranchActionTail() {
}
