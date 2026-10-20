package frame

import "github.com/egoist/mygo/ui"

type paneDrag struct {
	active      bool
	base, delta float32
}

// centerRightColumns keeps the center mounted while the right track concedes.
// Its divider uses a frozen rendered-width base for the whole pointer gesture:
// the solved center floor cannot feed a new base into the next pointer delta.
func centerRightColumns(c *ui.Context, m Model, center, right func(), header ...func()) {
	if len(header) > 0 && header[0] != nil {
		ui.Column(c).Key("workspace-shared-rows").Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			workspaceHeaderBand(c, header[0])
			centerRightBodyColumns(c, m, center, right)
		})
		return
	}
	centerRightBodyColumns(c, m, center, right)
}

// workspaceHeaderBand is AppFrame row 1 shared by the center's header and the
// right titlebar seat. Its auto height follows the center header, with the 48-DIP
// caption floor even for a sessionless or blank center. Nothing starts below it
// in a second row.
func workspaceHeaderBand(c *ui.Context, header func()) {
	ui.Box(c).Key("workspace-shared-titlebar-row").FillWidth().MinHeight(workspaceCaptionHeight).Shrink(0).Children(header)
}

func centerRightBodyColumns(c *ui.Context, m Model, center, right func()) {
	viewport, _ := viewportDIP(c)
	details := float32(0)
	if m.FilePaneWidth != nil {
		details = *m.FilePaneWidth
	}
	row := ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	drag := ui.Local(row, "right-drag", func() paneDrag { return paneDrag{} })
	row.Children(func() {
		parentDragging := m.Sizes.dragging
		m.Sizes.dragging = parentDragging || drag.active
		defer func() { m.Sizes.dragging = parentDragging }()
		// The conversation column takes the room the other tracks leave instead of
		// an absolute resolved width: a Center that lags a frame (a resize, a
		// dragged sidebar, the drawer band) would otherwise push the column past
		// the window edge and clip the hero headline with it.
		conversation := ui.Box(c).Key("conversation-column").Grow(1).MinWidth(0).Clip()
		if !drag.active && !m.Sizes.dragging {
			conversation.Transition(shellTrackTransition)
		}
		conversation.Children(center)
		open := details >= DetailsMin
		dividerWidth := SplitterWidth
		if !open {
			dividerWidth = 0
		}
		div := ui.Box(c).Key("right-divider").Width(dividerWidth).Shrink(0)
		if !drag.active && !m.Sizes.dragging {
			div.Transition(shellTrackTransition)
		}
		resource := ui.Box(c).Key("right-column").Width(details).Shrink(0).MinWidth(0).Clip()
		if !drag.active && !m.Sizes.dragging {
			resource.Transition(shellTrackTransition)
		}
		// Closed resources stop building immediately: no hidden keyboard owner
		// or native child survives under an animated canvas/exit ghost.
		if open {
			resource.Children(right)
		}
		if !open {
			drag.active = false
			return
		}
		maxDetails := maxf(DetailsMin, viewport*DetailsMaxRatio)
		div.Focusable().Role(ui.RoleSplitter).
			Label("Right pane divider").Range(float64(DetailsMin), float64(maxDetails), float64(details)).Step(10)
		if div.Shortcut(0, ui.KeyLeft) {
			m.Sizes.Right = clampDetailsWidth(details+10, viewport)
		}
		if div.Shortcut(0, ui.KeyRight) {
			m.Sizes.Right = clampDetailsWidth(details-10, viewport)
		}
		grip := float32(6)
		handle := ui.Box(c).Absolute().Top(0).Bottom(0).Left(m.Sizes.Center - (grip-SplitterWidth)/2).
			Width(grip).Cursor(ui.CursorResizeEW)
		if !drag.active && !m.Sizes.dragging {
			handle.Transition(shellTrackTransition)
		}
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
	_, viewportHeight := viewportDIP(c)
	previousViewportHeight := ui.Local(col, "terminal-viewport-height", func() float32 { return 0 })
	// Bounds belongs to the previous layout. The shell's resolved allocation
	// already reflects a resized viewport and its shared caption; an old small
	// Bounds.H must not keep conceding the saved split after the window grows.
	height := m.centerHeight
	if height <= 0 {
		height = col.Bounds().H
		if height <= 0 || *previousViewportHeight != viewportHeight {
			height = maxf(0, viewportHeight-col.Bounds().Y)
		}
	}
	// A consumed action can rebuild several passes before any layout commits.
	// Keep every pass on the resized allocation, then publish the viewport only
	// after painting; publishing here would let pass two reuse the old Bounds.H.
	col.DrawOver(func(_ *ui.Painter, _ ui.Rect) { *previousViewportHeight = viewportHeight })
	open := m.Sizes.Bottom > 40
	main := height
	maxMain := maxf(0, height-81)
	if open {
		main = m.Sizes.Main
		if main < 80 {
			main = DefaultLayout().Main
		}
		main = min(main, maxMain)
		if m.Terminal != nil && m.Terminal.Maximized {
			main = min(maxMain, maxf(min(float32(80), maxMain), height*0.25))
		}
	}
	drag := ui.Local(col, "terminal-drag", func() paneDrag { return paneDrag{} })
	col.Children(func() {
		conversation := ui.Box(c).Key("chat-column").Clip()
		if open {
			conversation.Height(main).Shrink(1).MinHeight(0)
		} else {
			// With no terminal split, inherit the actual parent allocation.
			// A first-frame viewport height has not yet subtracted the caption;
			// fixing and transitioning that stale height clips the footer.
			conversation.Grow(1).MinHeight(0)
		}
		if !drag.active && !m.Sizes.dragging {
			conversation.Transition(shellTrackTransition)
		}
		conversation.Children(chat)
		dividerHeight := SplitterWidth
		if !open {
			dividerHeight = 0
		}
		div := ui.Box(c).Key("terminal-divider").Height(dividerHeight).Shrink(0)
		if !drag.active && !m.Sizes.dragging {
			div.Transition(shellTrackTransition)
		}
		drawer := ui.Box(c).Key("terminal-column").Height(maxf(0, height-main-dividerHeight)).Shrink(1).MinHeight(0).Clip()
		if !drag.active && !m.Sizes.dragging {
			drawer.Transition(shellTrackTransition)
		}
		if open {
			drawer.Children(terminal)
		}
		if !open {
			drag.active = false
			return
		}
		minMain := min(float32(80), maxMain)
		div.Focusable().Role(ui.RoleSplitter).
			Label("Terminal pane divider").Range(float64(minMain), float64(maxMain), float64(main)).Step(10)
		setMain := func(value float32) {
			if m.Terminal != nil {
				m.Terminal.Maximized = false
			}
			m.Sizes.Main = min(maxf(minMain, value), maxMain)
		}
		if div.Shortcut(0, ui.KeyUp) {
			setMain(main - 10)
		}
		if div.Shortcut(0, ui.KeyDown) {
			setMain(main + 10)
		}
		grip := float32(6)
		handle := ui.Box(c).Absolute().Left(0).Right(0).Top(main - (grip-SplitterWidth)/2).
			Height(grip).Cursor(ui.CursorResizeNS)
		if !drag.active && !m.Sizes.dragging {
			handle.Transition(shellTrackTransition)
		}
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
