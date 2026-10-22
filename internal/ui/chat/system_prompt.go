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
	box := ui.Column(c.Key(key)).FillWidth()
	open := ui.Local(box, "open", func() bool { return false })
	title := i18n.T(i18n.KeyChatSystemPrompt)
	if msg.EventType == "system-prompt-update" {
		title = i18n.T(i18n.KeyChatSystemPromptUpdate)
	}
	arrow := "▸ "
	if *open {
		arrow = "▾ "
	}
	trigger := ui.Button(c, arrow+title).Label(i18n.T(i18n.KeyChatSystemPrompt)).Tooltip(title).
		FontSize(12).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
	if trigger.Clicked() {
		*open = !*open
	}
	// A long system prompt opens over the transcript instead of growing its
	// list row: an inline expansion changed row height after layout and painted
	// over the following message.
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel.Width(620).MaxWidthPercent(92).MaxHeight(420).Padding(12).Gap(8).
			Background(c.Theme().Background).Border(1, c.Theme().Border).Radius(12)
		ui.Text(c, title).FontSize(12).TextColor(muted)
		ui.Scroll(c).FillWidth().MaxHeight(360).Children(func() {
			ui.Text(c, msg.Body).FontSize(12).TextColor(fg).Selectable()
		})
	})
}
