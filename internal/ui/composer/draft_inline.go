package composer

import (
	"math"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// Inline controls use the same object rectangles as native selection and hit
// testing. Pointer capture belongs to the draft, including a drag across chips.
func paintInlineDraftChips(c *ui.Context, client *DraftClient, m Model, fg, muted, accent ui.Color) {
	left, _, top := client.ins()
	for _, chip := range client.inlineChips {
		chip := chip
		b := client.chipRect(chip)
		b.X += left
		b.Y += top - client.scrollY
		if b.Y+b.H <= 0 || b.Y >= client.viewportHeight {
			continue
		}
		mention, valid := client.mention(chip.reference)
		if !valid {
			continue
		}
		ink := accent
		if mention.Kind == DraftMentionSkill {
			ink = fg
		}
		body := ui.Box(c.Key(struct {
			Name  string
			Start int
		}{"composer-inline-chip", mention.Start})).Absolute().Left(b.X).Top(b.Y + 1).
			Width(b.W).Height(max(1, b.H-2)).Radius(5).Role(ui.RoleButton).Label(ChipLabel(mention)).
			Tooltip(mention.Text).KeepFocus().Disabled(m.Inert)
		fill := c.Theme().Surface
		selection := client.selection.Range
		if selection.Start < chip.reference.End && selection.End > chip.reference.Start {
			fill = c.Theme().Selection
		} else if body.Hovered() {
			fill = fill.Mix(accent, .12)
		}
		if client.pointerDragging && client.pointerChip != nil && *client.pointerChip == chip.reference {
			body.Opacity(.5)
		}
		body.Background(fill).Border(1, accent.Alpha(.22))
		body.HandleInput(func(ev ui.InputEvent) bool {
			ev.X, ev.Y = ev.X+b.X, ev.Y+b.Y+1
			return draftInputEvent(client, m, ev)
		})
		body.Children(func() {
			ui.Text(c, chip.label).Absolute().Left(6).Right(23).Top(0).Height(b.H - 2).
				Font(client.font.Family).FontSize(client.chipFont().Size).FixedLineHeight(b.H - 2).
				SingleLine().Ellipsis("…").TextColor(ink).PassThrough()
			remove := ui.Box(c).Absolute().Right(2).Top(0).Width(20).Height(b.H - 2).
				Role(ui.RoleButton).Label("Remove reference " + mention.Text).KeepFocus().
				Tooltip(i18n.T(i18n.KeyComposerRemoveReference)).Disabled(m.Inert || m.State != nil && m.State.Sending)
			remove.HandleInput(func(ev ui.InputEvent) bool {
				ev.X, ev.Y = ev.X+b.X+b.W-22, ev.Y+b.Y+1
				return draftInputEvent(client, m, ev)
			})
			remove.Children(func() {
				ui.Text(c, "×").FontSize(client.chipFont().Size).FixedLineHeight(b.H - 2).TextColor(muted).
					Fill().TextAlign(ui.Center).PassThrough()
			})
		})
	}
}

func (c *DraftClient) chipAt(p ui.Point) (*draftInlineChip, bool) {
	left, _, top := c.ins()
	p.X, p.Y = p.X-left, p.Y-top+c.scrollY
	for i := range c.inlineChips {
		b := c.chipRect(c.inlineChips[i])
		if b.Contains(p.X, p.Y) {
			return &c.inlineChips[i], p.X >= b.X+b.W-22
		}
	}
	return nil, false
}

func (c *DraftClient) pointerEvent(m Model, ev ui.InputEvent) bool {
	point := ui.Point{X: ev.X, Y: ev.Y}
	switch ev.Kind {
	case ui.InputBlur:
		c.pointerDown, c.pointerDragging, c.pointerChip = false, false, nil
		return false
	case ui.InputPointerDown:
		if c.marked != nil {
			return false
		}
		chip, remove := c.chipAt(point)
		if ev.Button == 1 {
			if chip == nil {
				return false
			}
			if mention, valid := c.mention(chip.reference); valid {
				openChipEdit(m, mention)
				c.SetSelectionUTF16(chip.reference.Start, chip.reference.End)
			}
			return true
		}
		if ev.Button != 0 {
			return false
		}
		c.pointerDown, c.pointerDragging, c.pointerChip = true, false, nil
		c.pointerStart, c.pointerRemove = point, remove
		if chip != nil && ev.Mods&ui.Shift == 0 {
			ref := chip.reference
			c.pointerChip = &ref
			// Preview activation must preserve a prior ranged selection.
			return true
		}
		to, valid := c.IndexForPoint(point)
		if valid {
			c.move(to, ev.Mods&ui.Shift != 0)
		}
		return valid
	case ui.InputPointerMove:
		if !c.pointerDown || ev.Button != 0 {
			return false
		}
		if math.Abs(float64(point.X-c.pointerStart.X))+math.Abs(float64(point.Y-c.pointerStart.Y)) > 4 {
			c.pointerDragging = true
		}
		if c.pointerChip == nil {
			to, valid := c.IndexForPoint(point)
			if valid {
				c.move(to, true)
			}
			return valid
		}
		return true
	case ui.InputPointerUp:
		if ev.Button != 0 || !c.pointerDown {
			return false
		}
		ref, dragging, remove := c.pointerChip, c.pointerDragging, c.pointerRemove
		c.pointerDown, c.pointerDragging, c.pointerChip = false, false, nil
		if ref == nil {
			return true
		}
		mention, valid := c.mention(*ref)
		if !valid {
			return true
		}
		if dragging {
			to, valid := c.IndexForPoint(point)
			if valid {
				c.moveReference(*ref, to)
			}
			return true
		}
		chip, hitRemove := c.chipAt(point)
		if chip == nil || chip.reference != *ref || hitRemove != remove {
			return true
		}
		if remove {
			removeDraftReference(m, mention)
		} else {
			activateDraftReference(m, mention)
		}
		return true
	}
	return false
}

// Moving an object is one native undo operation. Prose and all other object
// identities keep their relative order, unlike swapping labels across slots.
func (c *DraftClient) moveReference(ref draftReference, to int) bool {
	if c.marked != nil || to >= ref.Start && to <= ref.End {
		return false
	}
	m, valid := c.mention(ref)
	if !valid {
		return false
	}
	to = c.normalizeRange(ui.TextInputRange{Start: to, End: to}).Start
	c.pushUndo()
	c.typingUndoEnd = -1 // a pointer move starts its own edit, outside a typing burst
	c.replace(ui.TextInputRange{Start: ref.Start, End: ref.End}, "", false)
	if to > ref.End {
		to -= ref.End - ref.Start
	}
	remaining := c.text()
	byteAt := ui.UTF16ByteOffset(remaining, to)
	prefix, suffix := "", ""
	if byteAt > 0 {
		previous, _ := utf8.DecodeLastRuneInString(remaining[:byteAt])
		if !unicode.IsSpace(previous) {
			prefix = " "
		}
	}
	if byteAt < len(remaining) {
		next, _ := utf8.DecodeRuneInString(remaining[byteAt:])
		if !unicode.IsSpace(next) {
			suffix = " "
		}
	}
	c.replace(ui.TextInputRange{Start: to, End: to}, prefix+m.Text+suffix, false)
	moved := ref
	moved.Start = to + ui.UTF16Len(prefix)
	moved.End = moved.Start + ui.UTF16Len(m.Text)
	c.references = append(c.references, moved)
	c.sortReferences()
	c.shape()
	return true
}
