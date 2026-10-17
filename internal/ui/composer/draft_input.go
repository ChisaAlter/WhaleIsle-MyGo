package composer

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func paintDraftTextInput(c *ui.Context, m Model, fg, muted ui.Color, selectUTF16Start, selectUTF16End int) ui.Element {
	client := m.TextInput
	client.Sync()
	if selectUTF16Start >= 0 {
		client.SetSelectionUTF16(selectUTF16Start, selectUTF16End)
	}
	placeholder := i18n.T(i18n.KeyComposerPlaceholder)
	if m.Hero {
		placeholder = i18n.T(i18n.KeyComposerPlaceholderHero)
	}
	// InputBar.input: docked min-height 36; hero .input 52.
	minH, maxH := float32(36), float32(120)
	if m.Hero {
		minH, maxH = 52, 160
	}
	box := ui.Box(c).Fill().MinHeight(minH).MaxHeight(maxH).Role(ui.RoleTextField).Label("Composer draft").
		HandleTextInput(client).
		HandleInput(func(ev ui.InputEvent) bool { return draftInputEvent(client, m, ev) }).
		Draw(func(p *ui.Painter, r ui.Rect) {
			client.setWrapWidth(max(0, r.W-2*draftInputPad))
			t := c.Theme()
			text := client.text()
			if text == "" && placeholder != "" {
				p.Text(r.X+draftInputPad, r.Y+draftInputPad, placeholder, client.font.Size, muted)
				return
			}
			for _, b := range client.layout.SelectionRects(client.selection.Range) {
				b.X += r.X + draftInputPad
				b.Y += r.Y + draftInputPad
				p.Fill(b, t.Selection, 0)
			}
			p.TextLayout(client.layout, r.X+draftInputPad, r.Y+draftInputPad, fg)
			caret := client.layout.Caret(client.selection.Caret())
			caret.X += r.X + draftInputPad
			caret.Y += r.Y + draftInputPad
			p.Fill(caret, t.Accent, 0)
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
			// Shift+Enter → newline; Enter → send (chat parity with Electron).
			if ev.Mods&ui.Shift != 0 {
				client.ReplaceText(nil, "\n")
				return true
			}
			if m.Draft == nil || m.Snapshot.PromptBusy {
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
