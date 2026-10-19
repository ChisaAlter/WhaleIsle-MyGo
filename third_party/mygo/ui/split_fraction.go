package ui

import "math"

// SplitFraction lays out two panes separated by a one-DIP divider. The first
// pane receives *fraction of the available width after the divider. Pointer
// dragging and arrow keys update the supplied persistent binding directly,
// clamped to [minFraction, 1-minFraction]. minFraction must be between 0 and .5.
// Unlike Split, this control has no additional 40-DIP pane minimum. Changed
// reports a user move; the split remains proportional when its container resizes.
func SplitFraction(c *Context, fraction *float32, minFraction float32, first, second func()) Element {
	return splitFractionElement(c, fraction, minFraction, first, second, false)
}

// SplitVerticalFraction is SplitFraction with the first pane above the second.
func SplitVerticalFraction(c *Context, fraction *float32, minFraction float32, first, second func()) Element {
	return splitFractionElement(c, fraction, minFraction, first, second, true)
}

func splitFractionElement(c *Context, fraction *float32, minFraction float32, first, second func(), vertical bool) Element {
	if fraction == nil {
		panic("ui.SplitFraction: nil fraction binding")
	}
	if math.IsNaN(float64(minFraction)) || minFraction < 0 || minFraction > .5 {
		panic("ui.SplitFraction: minFraction must be between 0 and .5")
	}
	raw := c.build()
	if raw == nil {
		return Element{}
	}
	return wrapElement(buildSplitFraction(raw, fraction, minFraction, first, second, vertical))
}

func buildSplitFraction(c *context, fraction *float32, minimum float32, first, second func(), vertical bool) *node {
	clamp := func(value float32) float32 {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			value = .5
		}
		return min(1-minimum, max(minimum, value))
	}
	*fraction = clamp(*fraction)
	value := *fraction
	t := c.theme
	grip := t.Space(1.5)
	var e *node
	if vertical {
		e = coreColumn(c).AlignItems(Stretch)
	} else {
		e = coreRow(c).AlignItems(Stretch)
	}
	e.widget = "Split"
	e.Children(func() {
		var in *node
		if vertical {
			in = coreColumn(c).Grow(1).MinHeight(0).AlignItems(Stretch)
		} else {
			in = coreRow(c).Grow(1).MinWidth(0).AlignItems(Stretch)
		}
		// Input refers to the geometry at which the event was received. Layout
		// itself uses flex weights, so a new container size needs no old pixel size.
		total := in.st.w
		if vertical {
			total = in.st.h
		}
		span := max(float32(0), total-1)
		move := func(delta float32) {
			if span <= 0 {
				return
			}
			next := clamp(*fraction + delta/span)
			if next != *fraction {
				*fraction = next
				e.st.markChanged()
				c.rt.consumed = true
			}
		}
		in.Children(func() {
			coreBox(c).Grow(value).Basis(0).Shrink(1).MinWidth(0).MinHeight(0).Clip().Children(first)
			div := coreBox(c).Shrink(0).Focusable().Role(RoleSplitter).Label("Divider")
			div.widget = "Divider"
			div.flags |= flagOwnRing
			back, forth := KeyLeft, KeyRight
			if vertical {
				div.Height(1)
				back, forth = KeyUp, KeyDown
			} else {
				div.Width(1)
			}
			div.afterInput(func() {
				if div.Shortcut(0, back) {
					move(-t.Space(2.5))
				}
				if div.Shortcut(0, forth) {
					move(t.Space(2.5))
				}
			})
			div.hasRange, div.accRange = true, [3]float64{float64(minimum * 100), float64((1 - minimum) * 100), float64(value * 100)}
			if span > 0 {
				div.accStep = float64(t.Space(2.5) / span * 100)
			}
			coreBox(c).Grow(1 - value).Basis(0).Shrink(1).MinWidth(0).MinHeight(0).Clip().Children(second)
			handle := coreBox(c).Absolute()
			handle.flags |= flagDraggable | flagHover
			// Percent placement plus the divider correction tracks the actual
			// flex layout even on the first measured frame or a container resize.
			offset := -value - (grip-1)/2
			if vertical {
				handle.Left(0).Right(0).TopPercent(value*100).Margin(offset, 0, 0, 0).Height(grip).Cursor(CursorResizeNS)
			} else {
				handle.Top(0).Bottom(0).LeftPercent(value*100).Margin(0, 0, 0, offset).Width(grip).Cursor(CursorResizeEW)
			}
			dx, dy, held := handle.Dragged()
			handle.afterInput(func() {
				if held {
					div.Focus()
					if vertical {
						move(dy)
					} else {
						move(dx)
					}
				}
			})
			div.Draw(func(p *Painter, r Rect) {
				line := t.Border
				if held || handle.Hovered() {
					line = t.Accent
				}
				p.Fill(r, line, 0)
			})
			handle.Draw(func(p *Painter, r Rect) {
				if div.FocusVisible() {
					p.FocusRing(r, [4]float32{})
				}
			})
		})
	})
	return e
}
