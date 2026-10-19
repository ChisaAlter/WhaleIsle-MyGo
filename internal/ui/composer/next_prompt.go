package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

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
			return !item.Running && !item.Blank && item.Origin != "subagent" && (item.Presentation == nil || item.Presentation.Composer != "managed")
		}
	}
	return false
}

func acceptNextPrompt(m Model) bool {
	if m.NextPrompt.Text == "" || m.OnAcceptNextPrompt == nil || m.Draft == nil || strings.TrimSpace(*m.Draft) != "" || m.Editing || m.Inert || m.StopBlocked || !NextPromptEligible(m.Snapshot, m.State, true, m.Hero) {
		return false
	}
	if m.State != nil && (m.State.Sending || m.State.Preparing != 0 || len(m.State.Attachments) != 0) {
		return false
	}
	if m.TextInput != nil {
		if _, composing := m.TextInput.MarkedRange(); composing {
			return false
		}
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
	if m.NextPrompt.Text == "" || m.Inert || m.StopBlocked || !NextPromptEligible(m.Snapshot, m.State, true, m.Hero) {
		return false
	}
	button := ui.Button(c, "").Label("采纳下一条消息建议 / Accept next prompt").
		Tooltip(m.NextPrompt.Text + " · Tab").FillWidth().Padding(0).Background(ui.Transparent).BorderWidth(0)
	button.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, m.NextPrompt.Text).Grow(1).MinWidth(0).SingleLine().FontSize(14).TextColor(ink)
			ui.Text(c, "Tab").FontSize(11).TextColor(ink)
		})
	})
	if button.Clicked() {
		acceptNextPrompt(m)
	}
	return true
}
