package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var systemPromptIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><rect x="2" y="2.5" width="12" height="11" rx="2"/><path d="M2 5.5h12M5.5 5.5v8"/></svg>`))

// System prompts use the shared 141px opaque inline code well. The list
// measures its flow height just as it does other context disclosures.
func SystemPromptView(c *ui.Context, m Model, msg session.Message, fg, muted ui.Color) {
	key := "system-prompt:" + m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID + ":" + msg.ID
	box := ui.Column(c.Key(key)).FillWidth().MinWidth(0)
	open := ui.Local(box, "open", func() bool { return false })
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := i18n.T(i18n.KeyChatSystemPrompt)
	if msg.EventType == "system-prompt-update" {
		title = i18n.T(i18n.KeyChatSystemPromptUpdate)
	}
	box.Children(func() {
		paintContextDisclosure(c, box, open, contextDisclosure{Icon: systemPromptIcon, Title: title, Body: func() {
			paintContextBodyRuns(c, []contextContentRun{{Text: msg.Body}}, contextBodyPalette(palette), false)
		}}, palette)
	})
}
