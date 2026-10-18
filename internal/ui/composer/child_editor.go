package composer

import (
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"strings"
	"whaleisle/internal/store/session"
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
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		if len(v.Images) > 0 {
			ui.ScrollHorizontal(c).FillWidth().Height(120).Children(func() {
				for _, path := range v.Images {
					a := session.ComposerAttachment{Path: path, Name: filepath.Base(path)}
					card := ui.Column(c).Key("child-image:" + path).Width(160).Height(110).Gap(3)
					seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
					seat.load(a, false, c.Services().Invalidate)
					card.Children(func() {
						ui.Text(c, a.Name).FontSize(11).MaxLines(1)
						if seat.result != nil && seat.result.bitmap != nil {
							preview := ui.Box(c).FillWidth().Height(48).Focusable().Label("Preview attachment " + a.Name)
							preview.Children(func() { ui.Image(c, seat.result.bitmap).Fill().Fit(ui.Contain) })
							if preview.Clicked() || preview.Shortcut(0, ui.KeyEnter) {
								seat.open = true
							}
						}
						if ui.Button(c, "×").Label("Remove attachment "+a.Name).Disabled(d.Sending).Clicked() && v.OnRemoveImage != nil {
							v.OnRemoveImage(path)
						}
					})
					paintAttachmentOriginal(c, seat, a, a.Name)
				}
			})
		}
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
		ui.Row(c).Gap(6).Children(func() {
			if v.OnPickImages != nil && ui.Button(c, "添加图片 / Add images").Disabled(v.Disabled).Clicked() {
				v.OnPickImages()
			}
			if v.OnPasteImage != nil && ui.Button(c, "粘贴图片 / Paste image").Disabled(v.Disabled).Clicked() {
				v.OnPasteImage(true)
			}
			if ui.Button(c, "发送 / Send").Disabled(v.Disabled || (strings.TrimSpace(d.Text) == "" && len(v.Images) == 0)).Clicked() {
				send()
			}
		})
		if d.Error != "" {
			ui.Text(c, d.Error).FontSize(11).TextColor(muted)
		}
	})
}
