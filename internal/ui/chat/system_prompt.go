package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// SystemPromptView is an opaque, collapsed disclosure, never a diagnostic or a
// Markdown assistant bubble. Preserve the model-facing text's real line breaks.
func SystemPromptView(c *ui.Context, m Model, msg session.Message, fg, muted ui.Color) {
	key := "system-prompt:" + m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID + ":" + msg.ID
	open := isThinkingExpanded(m.ThinkingExpanded, key)
	title := i18n.T(i18n.KeyChatSystemPrompt)
	if msg.EventType == "system-prompt-update" {
		title = i18n.T(i18n.KeyChatSystemPromptUpdate)
	}
	arrow := "▸ "
	if open {
		arrow = "▾ "
	}
	if ui.Button(c, arrow+title).FontSize(12).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
		toggleThinkingExpanded(m.ThinkingExpanded, key)
	}
	if open {
		ui.Scroll(c.Key(key)).MaxHeight(141).Children(func() { ui.Text(c, msg.Body).FontSize(12).TextColor(fg) })
	}
}
