package composer

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
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
	ParentOffline     bool
	Placeholder       string
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
	t := design.For(c)
	fg, muted := t.Foreground, t.MutedForeground
	send := func() {
		if !v.Disabled && !v.ParentOffline && (strings.TrimSpace(d.Text) != "" || len(v.Images) > 0) {
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
	m.Disabled = v.Disabled
	m.ParentOffline = v.ParentOffline || (v.Disabled && !d.Sending && !v.StopBlocked)
	m.Placeholder = v.Placeholder
	m.OnQueryReferences = v.OnQueryReferences
	m.referenceInvalidate = c.Services().Invalidate
	m.OnSubmitGesture = func(accelerated bool) {
		if v.Disabled || v.ParentOffline {
			return
		}
		if v.OnSendGesture != nil {
			v.OnSendGesture(d.Text, accelerated)
		} else {
			send()
		}
	}
	radius := design.RadiusXL
	disabled := v.Disabled || v.ParentOffline
	root := ui.Column(c.Key("child-composer")).FillWidth().MinWidth(0).Padding(design.Sp(2), 0, 0, 0).Gap(design.Sp(3)).Radius(radius)
	focused := !disabled && root.FocusWithin()
	fill := composerCardFill(t)
	borderCol := composerCardBorder(t, focused)
	composerCardElevation(root, t, focused, disabled)
	root.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, fill, radius)
	}).DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Stroke(r, borderCol, radius, composerCardBorderWidth(p))
	})
	selected := ui.Local(root, "reference-selection", func() int { return 0 })
	m.SuggestList = ui.Local(root, "references", func() ui.ListState { return ui.ListState{Selected: selected} })
	BindStopRegion(root, d.Client, m, false)
	root.Children(func() {
		paintChildAttachments(c, v, muted)
		start, end := -1, -1
		if v.Chip != nil && v.Chip.Open {
			paintChipEditPopup(c, m)
		}
		area := paintDraftTextInput(c, m, fg, muted, start, end).Disabled(disabled)
		kind, q, tok, quoted, open := activeSuggestion(m, d.Client)
		if open && kind == SuggestMention {
			items := buildSuggestItemsWithQuote(m, kind, q, quoted)
			paintSuggestPopup(c, m, area, items, kind, tok)
		} else {
			d.references.cancelQuery()
		}
		if !disabled && !c.Root().FocusWithin() {
			area.AutoFocus()
		}
		if !disabled && d.EditFocus {
			area.Focus()
			d.EditFocus = false
		}
		if !disabled {
			if paths := area.DroppedFiles(); len(paths) > 0 {
				intakeTransferredFiles(m, paths)
			}
		}
		vToolbar := v
		vToolbar.Disabled = disabled
		paintChildToolbar(c, vToolbar, send)
		if d.Error != "" {
			ui.Text(c, FormatComposerError(d.Error)).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground)
		}
	})
}
