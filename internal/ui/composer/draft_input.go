package composer

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"math"
	"time"
)

// readablePlaceholder keeps the reference caption token when it is legible on
// the input fill, and falls back to muted ink when the light card washes it
// out (the default light caption sits around 2.2:1 on white).
func readablePlaceholder(caption, muted, fill ui.Color) ui.Color {
	if contrastRatio(caption, fill) >= 3 {
		return caption
	}
	return muted
}

func contrastRatio(a, b ui.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + .05) / (lb + .05)
}

func relativeLuminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= .04045 {
			return s / 12.92
		}
		return math.Pow((s+.055)/1.055, 2.4)
	}
	return .2126*channel(c.R) + .7152*channel(c.G) + .0722*channel(c.B)
}

func paintDraftTextInput(c *ui.Context, m Model, fg, placeholderInk ui.Color, selectUTF16Start, selectUTF16End int) ui.Element {
	client := m.TextInput
	client.Sync()
	client.configureTyping(m, c.Preferences().ReduceMotion)
	_, _, textInk := client.typing.colors(fg, c.Theme().Accent)
	if selectUTF16Start >= 0 {
		client.SetSelectionUTF16(selectUTF16Start, selectUTF16End)
	}
	placeholder := composerPlaceholder(m)
	left, right, top := draftInsets(m.Hero)
	client.setInsets(left, right, top)
	minH := float32(36)
	if m.Hero {
		// `.hero .input` is 52 DIP tall in the reference; the card's own
		// 20 DIP padding frames it.
		minH = 52
	}
	box := ui.Box(c.Key("composer-draft-input")).FillWidth().Role(ui.RoleTextField).Label("Composer draft").Disabled(m.Inert)
	client.syncStop(m, box.Focused())
	// Previous-frame width is available before sizing. On a first mount/resize,
	// the draw pass updates wrapping once and requests the matching height frame.
	if width := box.Bounds().W; width > 0 {
		client.setWrapWidth(max(0, width-left-right))
	}
	height := min(draftInputMaxHeight, max(minH, client.visualHeight()+top))
	height = resizeInputHeight(c, m, height)
	client.setViewport(height)
	box.Height(height).Clip()
	if m.Inert {
		box.PassThrough()
	} else {
		box.HandleTextInput(&client.input).HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEscape && !c.EscapeSequenceAvailable() {
				client.stop.reset()
				return false
			}
			if ev.Kind == ui.InputScroll {
				return client.scrollBy(ev.DY)
			}
			return draftInputEvent(client, m, ev)
		})
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		width := max(0, r.W-left-right)
		if width != client.wrapWidth {
			client.setWrapWidth(width)
			c.Services().Invalidate()
		}
		if m.Inert {
			return
		}
		p.Clip(r, 0, func() {
			for _, b := range client.visualSelectionRects(client.selection.Range) {
				b.X += r.X + left
				b.Y += r.Y + top - client.scrollY
				p.Fill(b, c.Theme().Selection, 0)
			}
		})
	}).DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !m.Inert {
			client.paintTyping(p, r, fg, c.Theme().Accent, box.Focused())
		}
	})
	box.Children(func() {
		text := client.text()
		if text == "" {
			ui.Box(c).Label("Composer placeholder").Absolute().Left(left).Right(right).
				Top(top).Children(func() {
				if paintNextPrompt(c, m, placeholderInk) {
					return
				}
				display := ui.Text(c, placeholder).FillWidth().Font(client.font.Family).FontSize(client.font.Size).
					FixedLineHeight(client.lineHeight()).TextColor(placeholderInk).PassThrough()
				if m.Hero {
					display.MaxLines(2)
				} else {
					display.SingleLine()
				}
			})
			return
		}
		ui.Text(c, text).Absolute().Left(left).Right(right).
			Top(top - client.scrollY).Font(client.font.Family).FontSize(client.font.Size).
			FixedLineHeight(client.lineHeight()).TextColor(textInk).PassThrough()
	})

	return box
}

func draftInputEvent(client *DraftClient, m Model, ev ui.InputEvent) bool {
	if m.Inert {
		return false
	}
	// Key-up separates independent presses without discarding the first. All
	// other intervening input, repeated/modifier Esc and local UI ownership reset.
	if ev.Kind != ui.InputKeyUp && (ev.Kind != ui.InputKeyDown || ev.Key != ui.KeyEscape || ev.Mods != 0 || ev.Repeat) {
		client.stop.reset()
	}

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
		if client.marked == nil && (ev.Mods == ui.Cmd || ev.Mods == ui.Cmd|ui.Shift) {
			switch ev.Key {
			case ui.KeyZ:
				if ev.Mods&ui.Shift != 0 {
					return client.Redo()
				}
				return client.Undo()
			case ui.KeyY:
				if ev.Mods == ui.Cmd {
					return client.Redo()
				}
			}
		}
		if claimLeadingSpace(client, m, ev) {
			return true
		}
		if handleSuggestKey(client, m, ev) {
			client.stop.reset()
			return true
		}
		if client.navigate(ev) {
			return true
		}
		switch ev.Key {
		case ui.KeyTab:
			if ev.Mods == 0 && client.marked == nil {
				return acceptNextPrompt(m)
			}
			return false
		case ui.KeyEscape:
			target, eligible := composerStopTarget(m)
			if !eligible || client.marked != nil || ev.Mods != 0 || ev.Repeat {
				client.stop.reset()
			}
			if m.State != nil && m.State.PermissionCommand.Visible(m.Snapshot.SelectedID) && client.marked == nil {
				m.State.PermissionCommand.Open = false
				return true
			}
			if m.State != nil && m.State.ModelCommand.Visible(m.Snapshot.SelectedID) && client.marked == nil {
				m.State.ModelCommand.Open = false
				return true
			}
			if cancelVoiceOnEscape(client, m, ev) {
				return true
			}
			if m.Editing && client.marked == nil && m.OnCancelEdit != nil {
				m.OnCancelEdit()
				return true
			}
			if eligible && client.marked == nil && ev.Mods == 0 && !ev.Repeat {
				if client.stop.press(target, time.Now(), time.Duration(m.Snapshot.StopSequenceMS)*time.Millisecond) {
					m.OnStop()
				}
				return true
			}
			return false
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
			if client.composing() {
				return false
			}
			// Only plain Enter and exactly Ctrl/Cmd+Enter submit. Other
			// chords remain with application shortcuts, never send a draft.
			if ev.Mods != 0 && ev.Mods != ui.Shift && ev.Mods != ui.Ctrl && ev.Mods != ui.Super {
				return false
			}
			if ev.Mods == ui.Shift {
				client.ReplaceText(nil, "\n")
				return true
			}
			if ev.Repeat || m.Draft == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
				return true
			}
			if handleModelCommand(m) {
				return true
			}
			if m.OnSubmitGesture != nil {
				m.OnSubmitGesture(ev.Mods == ui.Ctrl || ev.Mods == ui.Super)
				return true
			}
			if m.OnSubmit != nil {
				m.OnSubmit()
				return true
			}
			text := *m.Draft
			if handleSlashSend(m, text) {
				*m.Draft = ""
				client.ClearHistory()
				return true
			}
			if m.OnSend != nil {
				*m.Draft = ""
				client.ClearHistory()
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
			if pasteNativeFiles(m, mygo.Clipboard.ReadFiles) {
				return true
			}
			if m.OnPasteImage != nil {
				m.OnPasteImage(false)
			}
			if text := mygo.Clipboard.ReadText(); text != "" {
				r := client.target(nil)
				client.ReplaceText(&r, text)
			}
		case "delete":
			client.ReplaceText(nil, "")
		default:
			return false
		}
		return true
	}
	return false
}
