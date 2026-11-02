package sidebar

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Native TextInput delivers IME edits through its text client. Keep the Enter
// which finishes that composition separate from an ordinary submit press.
type sessionRenameKeys struct{ composing, ended bool }

func (k *sessionRenameKeys) observe(composing bool) {
	if k.composing && !composing {
		k.ended = true
	}
	k.composing = composing
}
func (k *sessionRenameKeys) handle(event ui.InputEvent, composing bool) bool {
	switch event.Kind {
	case ui.InputCompose:
		k.composing = event.Text != ""
		k.ended = !k.composing
	case ui.InputText:
		if k.composing {
			k.composing = false
			k.ended = true
		}
	case ui.InputKeyUp:
		k.ended = false
	case ui.InputKeyDown:
		if event.Key == ui.KeyEnter && (event.ShortcutBlocked || composing || k.composing || k.ended) {
			return true
		}
	}
	return false
}

// The request belongs to the sidebar root rather than the row or its menu:
// search, archive folding and collapsing the rail cannot unmount its editor.
func sessionRenameDialog(c *ui.Context, m Model) {
	if m.RenameSessionID == "" || m.RenameDraft == nil {
		return
	}
	open := true
	close := func() {
		if !m.RenameBusy && m.OnCancelRename != nil {
			m.OnCancelRename()
		}
	}
	ui.DialogBase(c.Key("session-rename-dialog:"+m.RenameSessionID), &open, func(backdrop, panel ui.Element) {
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), chrome.DialogSurfaceBox{Width: 380, Height: 206, Radius: 14, Padding: 20}).Label(workspaceText("重命名会话", "Rename session")).Gap(14).Children(func() {
			ui.Row(c).FillWidth().Shrink(0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, workspaceText("重命名会话", "Rename session")).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(chrome.Color(colors.FG)).Grow(1)
				if ui.Button(c, "×").Label(workspaceText("关闭", "Close")).Size(24, 24).Padding(0).Background(ui.Transparent).BorderWidth(0).Disabled(m.RenameBusy).Clicked() {
					close()
				}
			})
			input := ui.TextInput(c.Key("session-rename-name"), m.RenameDraft).Label(workspaceText("会话名称", "Session name")).FillWidth().MinWidth(0).Height(34).Shrink(0).FontSize(theme.InterfaceSize(14)).Disabled(m.RenameBusy)
			first := ui.Local(input, "initial-focus", func() bool { return true })
			if *first {
				input.Focus().SetTextSelection(0, utf8.RuneCountInString(*m.RenameDraft))
				*first = false
			}
			keys := ui.Local(input, "ime-enter", func() sessionRenameKeys { return sessionRenameKeys{} })
			composing := input.Composing()
			keys.observe(composing)
			input.HandleInput(func(event ui.InputEvent) bool { return keys.handle(event, composing) })
			if input.Changed() && m.OnRenameDraftChanged != nil {
				m.OnRenameDraftChanged(*m.RenameDraft)
			}
			trimmed := strings.TrimSpace(*m.RenameDraft)
			blocked := m.RenameBusy || trimmed == "" || m.OnCommitRename == nil
			submit := input.Submitted() && !keys.composing && !keys.ended
			if m.RenameError != "" {
				ui.Text(c, m.RenameError).Label(m.RenameError).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG))
			} else {
				ui.Box(c).Grow(1).MinHeight(0)
			}
			ui.Row(c).FillWidth().Shrink(0).Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeySidebarRenameCancel)).MinWidth(72).Height(32).Disabled(m.RenameBusy).Clicked() {
					close()
				}
				save := ui.PrimaryButton(c, i18n.T(i18n.KeySidebarRename)).MinWidth(72).Height(32).Disabled(blocked)
				if !blocked && (save.Clicked() || submit) {
					m.OnCommitRename(m.RenameSessionID, trimmed)
				}
			})
		})
	})
	if !open {
		close()
	}
}
