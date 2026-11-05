package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// ConversationSessionHeader hides actions/utilities only for the authoritative
// blank phase. A presentation owns its header even when its composer is not
// managed; running and attempted prompts are independent of the hero layout.
func conversationHeaderHidden(snap session.ViewState, activeTarget bool) bool {
	if snap.SelectedID == "" || !snap.SelectedSummaryBlank || activeTarget || snap.PromptBusy || snap.PromptAttempted || conversationHasTurnContent(snap) {
		return false
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			return item.Presentation == nil && !item.Running
		}
	}
	return true
}

// Blank headers preserve the two slots that the original header still renders.
// The explicit native plugin operation also remains in the selected row menu.
func conversationBlankHeader(c *ui.Context, m Model, snap session.ViewState) {
	ui.Row(c.Key("conversation-blank-caption")).Label("Blank conversation caption").FillWidth().Height(workspaceCaptionContent).
		Padding(0, m.captionRightInset, 0, 20).AlignItems(ui.Center).Children(func() {
		if m.ConversationContributions != nil {
			m.ConversationContributions(c, snap, "conversation.session.header.leading")
		}
		ui.Box(c).Grow(1).MinWidth(0)
		if m.ConversationContributions != nil {
			m.ConversationContributions(c, snap, "conversation.session.header.corner")
		}
	})
}
