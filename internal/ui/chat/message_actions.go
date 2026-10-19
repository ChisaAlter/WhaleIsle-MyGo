package chat

import (
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
	out := make(map[int]assistantActionTarget)
	last := -1
	lastSeat := -1
	flush := func(seat int, seq float64) {
		if last < 0 {
			return
		}
		message := messages[last]
		// Snapshot projects one durable Assistant lifecycle into text/reasoning
		// block rows. Copy all its visible text, as turn-assistant.assistantText
		// does, while keeping earlier tool-step Assistant lifecycles separate.
		if message.SourceMessageID != "" {
			message.Body = ""
			for _, block := range messages[:last+1] {
				if block.FeedbackMessageID() == message.SourceMessageID && showMessageCopy(block) {
					message.Body += block.Body
				}
			}
		}
		out[seat] = assistantActionTarget{message: message, endSeq: seq}
		lastSeat = seat
		last = -1
	}
	for index, msg := range messages {
		if msg.Role == "user" {
			flush(last, 0)
		}
		if msg.Role == "assistant" && showMessageCopy(msg) {
			last = index
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
			if msg.Role == "user" || msg.Role == "assistant" {
				target.latest = false
				break
			}
		}
		out[lastSeat] = target
	}
	return out
}

func assistantMessageActions(c *ui.Context, m Model, target assistantActionTarget, owner ui.Element, muted ui.Color) {
	msg := target.message
	row := ui.Row(c).Key("assistant-actions").Label("Assistant message actions").Gap(8).
		Height(28+theme.CurrentPreferences().Chat.Size-14).AlignItems(ui.Center).Margin(4, 0, 0, -6)
	messageQuoteMenu(row, m, msg)
	if !target.latest && !owner.Hovered() && !owner.FocusWithin() {
		row.Opacity(0)
	}
	row.Children(func() {
		messageCopyAction(c, msg, true, muted)
		if showMessageFeedback(msg) && m.FeedbackStore != nil {
			authoritativeFeedbackActions(c, m.FeedbackStore, msg.FeedbackMessageID(), muted)
		}
		if target.endSeq > 0 && m.OnForkTurn != nil {
			allowed := canForkAssistantTarget(m.Snapshot.Messages, target) && !m.Snapshot.HistoryBranchBusy && m.Snapshot.ClientReady
			b := messageAction(c, messageIconBranchOutline, branchActionLabel(), true, muted)
			if !allowed {
				b.Opacity(.4)
			}
			if allowed && b.Clicked() {
				m.OnForkTurn(target.endSeq)
			}
		}
		if msg.EventTime > 0 {
			ui.Text(c, formatEventTime(msg.EventTime)).FontSize(theme.CurrentPreferences().Chat.Size-2).FixedLineHeight(theme.CurrentPreferences().Chat.Size+10).Margin(0, 0, 0, 8).TextColor(muted)
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
func messageQuoteMenu(row ui.Element, m Model, msg session.Message) {
	if m.OnQuoteMessage == nil || !showMessageCopy(msg) {
		return
	}
	row.ContextMenu(func(menu *ui.Menu) {
		if menu.Item(i18n.T(i18n.KeyChatQuote)).Chosen() {
			m.OnQuoteMessage(msg.Body)
		}
	})
}

func branchActionLabel() string {
	if i18n.Locale() == i18n.LocaleEN {
		return "Branch here"
	}
	return "从此处分支"
}
