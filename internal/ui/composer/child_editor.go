package composer

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// ChildEditor exposes editing primitives only. No parent session snapshot,
// model/permission picker, command catalog, or ordinary prompt transport enters.
type ChildEditorModel struct {
	ThemeID           string
	SystemDark        bool
	Snapshot          session.ViewState
	OnSendGesture     func(string, bool)
	OnStop            func()
	StopBlocked       bool
	OnQueryReferences func(context.Context, string, bool) ([]session.ReferenceCandidate, error)
	Draft             *DraftState
	Chip              *ChipEdit
	Images            []string
	Disabled          bool
	OnAddImages       func([]string)
	OnRemoveImage     func(string)
	OnPickImages      func()
	OnPasteImage      func(bool)
	OnSend            func(string)
	OnOpenReference   func(string)
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
		if !v.Disabled && (strings.TrimSpace(d.Text) != "" || len(v.Images) > 0) {
			if v.OnSendGesture != nil {
				v.OnSendGesture(d.Text, false)
			} else if v.OnSend != nil {
				v.OnSend(d.Text)
			}
		}
	}
	// Editing suppresses main-only local /model and /permission commands. Text is
	// sent literally to the dedicated child callback, never interpreted as root UI.
	m := Model{Draft: &d.Text, State: d, TextInput: d.Client, UseTextInputClient: true, Editing: true, ChipEdit: v.Chip, OnSubmit: send, OnAttachFiles: v.OnAddImages, OnPasteImage: v.OnPasteImage, OnOpenReference: v.OnOpenReference}
	m.Child = true
	m.Snapshot = v.Snapshot
	m.OnStop = v.OnStop
	m.StopBlocked = v.StopBlocked
	m.OnQueryReferences = v.OnQueryReferences
	m.referenceInvalidate = c.Services().Invalidate
	m.OnSubmitGesture = func(accelerated bool) {
		if v.Disabled {
			return
		}
		if v.OnSendGesture != nil {
			v.OnSendGesture(d.Text, accelerated)
		} else {
			send()
		}
	}
	colors := theme.Resolve(v.ThemeID).Active(v.SystemDark)
	radius := theme.ResolvedSkinLayout(v.ThemeID, v.SystemDark).ComposerRadius
	root := ui.Column(c.Key("child-composer")).FillWidth().MinWidth(0).Padding(8, 0, 0, 0).Gap(12).Radius(radius).Draw(chrome.Glass(chrome.Color(colors.InputMajor), chrome.MaterialPanel, radius))
	selected := ui.Local(root, "reference-selection", func() int { return 0 })
	m.SuggestList = ui.Local(root, "references", func() ui.ListState { return ui.ListState{Selected: selected} })
	BindStopRegion(root, d.Client, m, false)
	root.Children(func() {
		paintChildAttachments(c, v, muted)
		start, end := -1, -1
		if v.Chip != nil && v.Chip.Open {
			paintChipEditPopup(c, m, c.Theme().Background, c.Theme().Border, muted)
		}
		area := paintDraftTextInput(c, m, fg, muted, start, end).Disabled(v.Disabled)
		kind, q, tok, quoted, open := activeSuggestion(m, d.Client)
		if open && kind == SuggestMention {
			items := buildSuggestItemsWithQuote(m, kind, q, quoted)
			paintSuggestPopup(c, m, area, items, kind, tok, theme.Resolve(v.ThemeID).Active(v.SystemDark))
		} else {
			d.references.cancelQuery()
		}
		if !v.Disabled && !c.Root().FocusWithin() {
			area.AutoFocus()
		}
		if !v.Disabled && d.EditFocus {
			area.Focus()
			d.EditFocus = false
		}
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
