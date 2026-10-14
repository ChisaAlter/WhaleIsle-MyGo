package composer

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func paintDraftTextInput(c *ui.Context, m Model, fg, placeholderInk ui.Color, selectUTF16Start, selectUTF16End int) ui.Element {
	client := m.TextInput
	client.Sync()
	if selectUTF16Start >= 0 {
		client.SetSelectionUTF16(selectUTF16Start, selectUTF16End)
	}
	placeholder := i18n.T(i18n.KeyComposerPlaceholder)
	minH := float32(36)
	if m.Hero {
		placeholder = i18n.T(i18n.KeyComposerPlaceholderHero)
		minH = 52
	}
	box := ui.Box(c).FillWidth().Role(ui.RoleTextField).Label("Composer draft")
	// Previous-frame width is available before sizing. On a first mount/resize,
	// the draw pass updates wrapping once and requests the matching height frame.
	if width := box.Bounds().W; width > 0 {
		client.setWrapWidth(max(0, width-draftInputLeft-draftInputRight))
	}
	height := min(draftInputMaxHeight, max(minH, client.visualHeight()+draftInputTop))
	client.setViewport(height)
	box.Height(height).Clip().
		HandleTextInput(client).
		HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputScroll {
				return client.scrollBy(ev.DY)
			}
			return draftInputEvent(client, m, ev)
		}).Draw(func(p *ui.Painter, r ui.Rect) {
		width := max(0, r.W-draftInputLeft-draftInputRight)
		if width != client.wrapWidth {
			client.setWrapWidth(width)
			c.Services().Invalidate()
		}
		p.Clip(r, 0, func() {
			for _, b := range client.visualSelectionRects(client.selection.Range) {
				b.X += r.X + draftInputLeft
				b.Y += r.Y + draftInputTop - client.scrollY
				p.Fill(b, c.Theme().Selection, 0)
			}
		})
	}).DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !box.Focused() {
			return
		}
		caret := client.visualCaret(client.selection.Caret())
		caret.X += r.X + draftInputLeft
		caret.Y += r.Y + draftInputTop - client.scrollY
		p.Clip(r, 0, func() { p.Fill(caret, c.Theme().Accent, 0) })
	})
	box.Children(func() {
		text := client.text()
		if text == "" {
			ui.Box(c).Label("Composer placeholder").Absolute().Left(draftInputLeft).Right(draftInputRight).
				Top(draftInputTop).Children(func() {
				display := ui.Text(c, placeholder).FillWidth().Font(client.font.Family).FontSize(client.font.Size).
					FixedLineHeight(draftInputLineHeight).TextColor(placeholderInk).PassThrough()
				if m.Hero {
					display.MaxLines(2)
				} else {
					display.SingleLine()
				}
			})
			return
		}
		ui.Text(c, text).Absolute().Left(draftInputLeft).Right(draftInputRight).
			Top(draftInputTop - client.scrollY).Font(client.font.Family).FontSize(client.font.Size).
			FixedLineHeight(draftInputLineHeight).TextColor(fg).PassThrough()
	})

	return box
}

func draftInputEvent(client *DraftClient, m Model, ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputPointerDown:
		to, ok := client.IndexForPoint(ui.Point{X: ev.X, Y: ev.Y})
		if !ok {
			return false
		}
		client.move(to, ev.Mods&ui.Shift != 0)
		return true
	case ui.InputPointerMove:
		if ev.Button == 0 {
			to, ok := client.IndexForPoint(ui.Point{X: ev.X, Y: ev.Y})
			if !ok {
				return false
			}
			client.move(to, true)
			return true
		}
	case ui.InputPointerUp:
		return true
	case ui.InputKeyDown:
		if ev.Mods == ui.Cmd && ev.Key == ui.KeyA {
			client.selection = ui.TextInputSelection{Range: ui.TextInputRange{End: ui.UTF16Len(client.text())}}
			return true
		}
		if ev.Mods == ui.Cmd {
			switch ev.Key {
			case ui.KeyZ:
				if ev.Mods&ui.Shift != 0 {
					return client.Redo()
				}
				return client.Undo()
			case ui.KeyY:
				return client.Redo()
			}
		}
		switch ev.Key {
		case ui.KeyEscape:
			if m.Editing && client.marked == nil && m.OnCancelEdit != nil {
				m.OnCancelEdit()
				return true
			}
			return false
		case ui.KeyHome:
			client.move(0, ev.Mods&ui.Shift != 0)
			return true
		case ui.KeyEnd:
			client.move(ui.UTF16Len(client.text()), ev.Mods&ui.Shift != 0)
			return true
		case ui.KeyBackspace, ui.KeyDelete:
			r := client.selection.Range
			if r.Start == r.End && client.layout != nil {
				next := client.layout.PreviousBoundary(r.Start)
				if ev.Key == ui.KeyDelete {
					next = client.layout.NextBoundary(r.End)
				}
				r = ui.TextInputRange{Start: min(next, r.Start), End: max(next, r.End)}
			}
			client.ReplaceText(&r, "")
			return true
		case ui.KeyEnter:
			if client.marked != nil {
				return false
			}
			// Shift+Enter → newline; Enter → send (chat parity with Electron).
			if ev.Mods&ui.Shift != 0 {
				client.ReplaceText(nil, "\n")
				return true
			}
			if m.Draft == nil || m.Snapshot.PromptBusy {
				return true
			}
			if m.OnSubmit != nil {
				m.OnSubmit()
				return true
			}
			text := *m.Draft
			if handleSlashSend(m, text) {
				*m.Draft = ""
				client.Sync()
				return true
			}
			if m.OnSend != nil {
				*m.Draft = ""
				client.Sync()
				m.OnSend(text)
			}
			return true
		}
	case ui.InputCommand:
		switch ev.Text {
		case "selectAll":
			client.selection = ui.TextInputSelection{Range: ui.TextInputRange{End: ui.UTF16Len(client.text())}}
		case "copy", "cut":
			s, _ := client.TextForRange(client.selection.Range)
			mygo.Clipboard.WriteText(s)
			if ev.Text == "cut" {
				client.ReplaceText(nil, "")
			}
		case "paste":
			client.ReplaceText(nil, mygo.Clipboard.ReadText())
		case "delete":
			client.ReplaceText(nil, "")
		default:
			return false
		}
		return true
	}
	return false
}
