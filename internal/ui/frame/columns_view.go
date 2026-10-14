package frame

import "github.com/egoist/mygo/ui"

type paneDrag struct {
	active      bool
	base, delta float32
}

// centerRightColumns keeps the center mounted while the right track concedes.
// Its divider uses a frozen rendered-width base for the whole pointer gesture:
// the solved center floor cannot feed a new base into the next pointer delta.
func centerRightColumns(c *ui.Context, m Model, center, right func()) {
	viewport, _ := c.Size()
	details := float32(0)
	if m.FilePaneWidth != nil {
		details = *m.FilePaneWidth
	}
	row := ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	drag := ui.Local(row, "right-drag", func() paneDrag { return paneDrag{} })
	row.Children(func() {
		ui.Box(c).Key("conversation-column").Width(m.Sizes.Center).Shrink(0).Clip().Children(center)
		if details < DetailsMin {
			drag.active = false
			return
		}
		maxDetails := maxf(DetailsMin, viewport*DetailsMaxRatio)
		div := ui.Box(c).Width(SplitterWidth).Shrink(0).Focusable().Role(ui.RoleSplitter).
			Label("Right pane divider").Range(float64(DetailsMin), float64(maxDetails), float64(details)).Step(10)
		if div.Shortcut(0, ui.KeyLeft) {
			m.Sizes.Right = clampDetailsWidth(details+10, viewport)
		}
		if div.Shortcut(0, ui.KeyRight) {
			m.Sizes.Right = clampDetailsWidth(details-10, viewport)
		}
		ui.Box(c).Key("right-column").Grow(1).MinWidth(0).Clip().Children(right)
		grip := float32(6)
		handle := ui.Box(c).Absolute().Top(0).Bottom(0).Left(m.Sizes.Center - (grip-SplitterWidth)/2).
			Width(grip).Cursor(ui.CursorResizeEW)
		dx, _, held := handle.Dragged()
		if held {
			div.Focus()
			if !drag.active {
				drag.active = true
				drag.base = details
				drag.delta = 0
			}
			if dx != 0 {
				drag.delta += dx
				m.Sizes.Right = clampDetailsWidth(drag.base-drag.delta, viewport)
			}
		} else {
			drag.active = false
		}
		div.Draw(func(p *ui.Painter, r ui.Rect) {
			color := c.Theme().Border
			if held || handle.Hovered() {
				color = c.Theme().Accent
			}
			p.Fill(r, color, 0)
		})
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if div.FocusVisible() {
				p.FocusRing(r, [4]float32{})
			}
		})
	})
}

// centerTerminalColumn keeps chat at one stable parent/identity whether the
// terminal is mounted or hidden. It never requests focus unless its divider
// itself is being dragged, so an existing input selection survives toggles.
func centerTerminalColumn(c *ui.Context, m Model, chat, terminal func()) {
	col := ui.Column(c).Fill().MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	height := m.centerHeight
	if height <= 0 {
		_, viewportHeight := c.Size()
		height = maxf(0, viewportHeight-col.Bounds().Y)
	}
	open := m.Sizes.Bottom > 40
	main := height
	maxMain := maxf(0, height-81)
	if open {
		main = m.Sizes.Main
		if main < 80 {
			main = DefaultLayout().Main
		}
		main = min(main, maxMain)
	}
	drag := ui.Local(col, "terminal-drag", func() paneDrag { return paneDrag{} })
	col.Children(func() {
		ui.Box(c).Key("chat-column").Height(main).Shrink(0).Clip().Children(chat)
		if !open {
			drag.active = false
			return
		}
		minMain := min(float32(80), maxMain)
		div := ui.Box(c).Height(SplitterWidth).Shrink(0).Focusable().Role(ui.RoleSplitter).
			Label("Terminal pane divider").Range(float64(minMain), float64(maxMain), float64(main)).Step(10)
		setMain := func(value float32) { m.Sizes.Main = min(maxf(minMain, value), maxMain) }
		if div.Shortcut(0, ui.KeyUp) {
			setMain(main - 10)
		}
		if div.Shortcut(0, ui.KeyDown) {
			setMain(main + 10)
		}
		ui.Box(c).Key("terminal-column").Grow(1).MinHeight(0).Clip().Children(terminal)
		grip := float32(6)
		handle := ui.Box(c).Absolute().Left(0).Right(0).Top(main - (grip-SplitterWidth)/2).
			Height(grip).Cursor(ui.CursorResizeNS)
		_, dy, held := handle.Dragged()
		if held {
			div.Focus()
			if !drag.active {
				drag.active = true
				drag.base = main
				drag.delta = 0
			}
			if dy != 0 {
				drag.delta += dy
				setMain(drag.base + drag.delta)
			}
		} else {
			drag.active = false
		}
		div.Draw(func(p *ui.Painter, r ui.Rect) {
			color := c.Theme().Border
			if held || handle.Hovered() {
				color = c.Theme().Accent
			}
			p.Fill(r, color, 0)
		})
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if div.FocusVisible() {
				p.FocusRing(r, [4]float32{})
			}
		})
	})
}
