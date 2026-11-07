package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// NextPromptHint returns the localized Tab adoption hint (`input.nextPrompt.hint`).
func NextPromptHint() string {
	return composerText("Tab 采纳，不会自动发送", "Tab to accept; not sent automatically")
}

// NextPromptAcceptLabel returns the localized accessible label for adopting the
// next-prompt suggestion (`input.nextPrompt.accept`).
func NextPromptAcceptLabel(text string) string {
	return composerText("采纳下一句提示", "Accept next-prompt suggestion") + ": " + text
}

// NextPromptTooltip returns the localized tooltip combining the suggestion text
// and the Tab adoption hint.
func NextPromptTooltip(text string) string {
	return text + " · " + NextPromptHint()
}

// NextPromptEligible mirrors the original plain, empty, idle composer surface.
func NextPromptEligible(snapshot session.ViewState, draft *DraftState, enabled, hero bool) bool {
	if !enabled || hero || !snapshot.ClientReady || snapshot.SelectedID == "" || !snapshot.HistoryReady || snapshot.HistoryLoading || snapshot.HistoryError != "" || snapshot.PromptBusy || snapshot.MessageEditBusy || snapshot.HistoryBranchBusy || draft == nil {
		return false
	}
	if strings.TrimSpace(draft.Text) != "" || len(draft.Attachments) != 0 || draft.Sending || draft.Preparing != 0 || draft.EditSeq != 0 || draft.ModelCommand.Visible(snapshot.SelectedID) || draft.PermissionCommand.Visible(snapshot.SelectedID) {
		return false
	}
	for _, item := range snapshot.Items {
		if item.SessionID == snapshot.SelectedID {
			return !item.Running && !item.Blank && item.Origin != "subagent" && item.SubagentIdentity == nil && !composerItemParentOffline(snapshot, item) && (item.Presentation == nil || item.Presentation.Composer != "managed")
		}
	}
	return false
}

func nextPromptVisible(m Model) bool {
	if m.NextPrompt.Text == "" || m.Inert || m.Disabled || composerParentOffline(m) || m.Editing || m.StopBlocked || m.stopSuggestionsOpen || !NextPromptEligible(m.Snapshot, m.State, true, m.Hero) {
		return false
	}
	if m.TextInput != nil {
		if _, composing := m.TextInput.MarkedRange(); composing {
			return false
		}
	}
	return true
}

func acceptNextPrompt(m Model) bool {
	if m.OnAcceptNextPrompt == nil || m.Draft == nil || strings.TrimSpace(*m.Draft) != "" || !nextPromptVisible(m) {
		return false
	}
	if m.State != nil && (m.State.Sending || m.State.Preparing != 0 || len(m.State.Attachments) != 0) {
		return false
	}
	text := m.OnAcceptNextPrompt()
	if text == "" {
		return false
	}
	*m.Draft = text
	if m.TextInput != nil {
		m.TextInput.Sync()
		end := ui.UTF16Len(text)
		m.TextInput.SetSelectionUTF16(end, end)
	}
	if m.State != nil {
		m.State.EditFocus = true
	}
	return true
}

func paintNextPrompt(c *ui.Context, m Model, ink ui.Color) bool {
	if !nextPromptVisible(m) {
		return false
	}
	hint := NextPromptHint()
	button := ui.Button(c, "").Label(NextPromptAcceptLabel(m.NextPrompt.Text)).
		Tooltip(NextPromptTooltip(m.NextPrompt.Text)).FillWidth().Padding(0).Background(ui.Transparent).BorderWidth(0)
	button.Children(func() {
		ui.Row(c).Label("采纳下一条消息建议 / Accept next prompt").FillWidth().Gap(6).AlignItems(ui.Center).PassThrough().Children(func() {
			ui.Text(c, m.NextPrompt.Text).Grow(1).MinWidth(0).SingleLine().FontSize(14).FixedLineHeight(22).TextColor(ink).PassThrough()
			ui.Box(c).Label(hint).Tooltip(hint).Shrink(0).Padding(0, 4).Radius(4).Border(0.5, c.Theme().Border).PassThrough().Children(func() {
				ui.Text(c, "Tab").FontSize(12).FixedLineHeight(18).TextColor(ink).PassThrough()
			})
		})
	})
	if button.Clicked() {
		acceptNextPrompt(m)
	}
	return true
}
