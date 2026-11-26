package sidebar

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
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
	sessionRenameDialogPtr(c, &m)
}

func sessionRenameDialogPtr(c *ui.Context, m *Model) {
	if m.RenameSessionID == "" || m.RenameDraft == nil {
		return
	}
	open := true
	closeDialog := func() {
		if !m.RenameBusy && m.OnCancelRename != nil {
			m.OnCancelRename()
		}
	}
	title := i18n.T(i18n.KeyWSRenameSessionTitle)
	submit := false

	coss.Dialog(c, &open, coss.DialogOptions{
		Title:           title,
		MaxWidth:        380,
		NoClose:         m.RenameBusy,
		DismissDisabled: m.RenameBusy,
		Key:             "session-rename-dialog:" + m.RenameSessionID,
		Footer: func() {
			trimmed := strings.TrimSpace(*m.RenameDraft)
			blocked := m.RenameBusy || trimmed == "" || m.OnCommitRename == nil
			if coss.Button(c, i18n.T(i18n.KeySidebarRenameCancel), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonMD,
				Disabled: m.RenameBusy,
				Key:      "session-rename-cancel",
			}).MinWidth(72).Clicked() {
				closeDialog()
			}
			save := coss.Button(c, i18n.T(i18n.KeySidebarRename), coss.ButtonOptions{
				Variant:  coss.ButtonDefault,
				Size:     coss.ButtonMD,
				Disabled: blocked,
				Loading:  m.RenameBusy,
				Key:      "session-rename-save",
			}).MinWidth(72)
			if !blocked && (save.Clicked() || submit) {
				m.OnCommitRename(m.RenameSessionID, trimmed)
			}
		},
	}, func() {
		coss.Field(c, coss.FieldOptions{
			Error:    m.RenameError,
			Disabled: m.RenameBusy,
		}, func() {
			input := coss.Input(c, m.RenameDraft, coss.InputOptions{
				Size:     coss.InputMD,
				Label:    i18n.T(i18n.KeyWSFieldSessionName),
				Disabled: m.RenameBusy,
				Invalid:  m.RenameError != "",
				Error:    m.RenameError,
				Key:      "session-rename-name",
			})
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
			submit = input.Submitted() && !keys.composing && !keys.ended
		})
	}).Label(title)

	if !open {
		closeDialog()
	}
}
