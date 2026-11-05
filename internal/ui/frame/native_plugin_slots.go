package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func renderGlobalShellSlot(c *ui.Context, m Model, slot string, fallback func()) {
	if m.GlobalPluginSlot != nil {
		m.GlobalPluginSlot(c, slot, fallback)
	} else if fallback != nil {
		fallback()
	}
}

func nativeComposerSlot(m Model) func(*ui.Context, string, func()) {
	if m.PluginSlot == nil {
		return nil
	}
	return func(c *ui.Context, slot string, fallback func()) {
		if slot == "composer" {
			slot = "conversation.composer"
		}
		m.PluginSlot(c, slot, "", -1, fallback)
	}
}

func nativeReasoningSlot(m Model) func(*ui.Context, session.Message, func()) {
	if m.PluginSlot == nil {
		return nil
	}
	return func(c *ui.Context, message session.Message, fallback func()) {
		m.PluginSlot(c, "conversation.chat.reasoning.body", "", message.Seq, fallback)
	}
}

func nativeMessageSlot(m Model) func(*ui.Context, session.ViewState, session.Message, string, func()) {
	if m.PluginSlot == nil {
		return nil
	}
	return func(c *ui.Context, _ session.ViewState, message session.Message, slot string, fallback func()) {
		m.PluginSlot(c, slot, "", message.Seq, fallback)
	}
}

func nativeToolSlot(m Model) func(*ui.Context, string, *session.ToolCallView, func()) {
	if m.PluginSlot == nil {
		return nil
	}
	return func(c *ui.Context, slot string, tool *session.ToolCallView, fallback func()) {
		if tool == nil {
			if fallback != nil {
				fallback()
			}
			return
		}
		m.PluginSlot(c, slot, tool.CallID, -1, fallback)
	}
}

func selectedConversationParent(snap session.ViewState) string {
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			return item.ParentID
		}
	}
	return ""
}
