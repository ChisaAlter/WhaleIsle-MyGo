package composer

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

// ChildEditor exposes editing primitives only. No parent session snapshot,
// model/permission picker, command catalog, or ordinary prompt transport enters.
type ChildEditorModel struct {
	Draft           *DraftState
	Chip            *ChipEdit
	Images          []string
	Disabled        bool
	OnAddImages     func([]string)
	OnRemoveImage   func(string)
	OnPickImages    func()
	OnPasteImage    func(bool)
	OnSend          func(string)
	OnOpenReference func(string)
}

func ChildEditor(c *ui.Context, v ChildEditorModel) {
	d := v.Draft
	if d == nil {
		return
	}
	if d.Client == nil {
		d.Client = NewDraftClient(&d.Text)
	}
	fg, muted := c.Theme().Text, c.Theme().TextMuted
	send := func() {
		if !v.Disabled && v.OnSend != nil && (strings.TrimSpace(d.Text) != "" || len(v.Images) > 0) {
			v.OnSend(d.Text)
		}
	}
	// Editing suppresses main-only local /model and /permission commands. Text is
	// sent literally to the dedicated child callback, never interpreted as root UI.
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, Editing: true, ChipEdit: v.Chip, OnSubmit: send, OnAttachFiles: v.OnAddImages, OnPasteImage: v.OnPasteImage, OnOpenReference: v.OnOpenReference}
	ui.Column(c.Key("child-composer")).FillWidth().Gap(0).Children(func() {
		paintChildAttachments(c, v, muted)
		start, end := -1, -1
		if mentions := ParseDraftMentions(d.Text); len(mentions) > 0 {
			paintMentionChips(c, m, mentions, muted, c.Theme().Accent, true, &start, &end)
		}
		if v.Chip != nil && v.Chip.Open {
			paintChipEditPopup(c, m, c.Theme().Background, c.Theme().Border, muted)
		}
		area := paintDraftTextInput(c, m, fg, muted, start, end).Disabled(v.Disabled)
		if !v.Disabled {
			if paths := area.DroppedFiles(); len(paths) > 0 {
				intakeTransferredFiles(m, paths)
			}
		}
		paintChildToolbar(c, v, send)
		if d.Error != "" {
			ui.Text(c, d.Error).FontSize(11).TextColor(muted)
		}
	})
}
