package frame

import "github.com/egoist/mygo/ui"

// SidebarResponsive mirrors AppFrame's desktop 1024px breakpoint. It is
// shared across copied frame models but is never a saved layout preference.
type SidebarResponsive struct {
	narrow, expanded, rightOpen bool
}

func (sizes *LayoutSizes) observeSidebarViewport(width float32) {
	if sizes.Responsive == nil {
		sizes.Responsive = &SidebarResponsive{}
	}
	narrow := width < 1024
	if sizes.Responsive.narrow != narrow {
		sizes.Responsive.expanded = false
	}
	sizes.Responsive.narrow = narrow
}

func (sizes *LayoutSizes) sidebarPreference() float32 {
	if r := sizes.Responsive; r != nil && r.narrow {
		if !r.expanded {
			return 0
		}
		if sizes.Sidebar <= 0 {
			return SidebarDefault
		}
	}
	return sizes.Sidebar
}

// ToggleSidebar changes the preference, never the resolved viewport width.
func (sizes *LayoutSizes) ToggleSidebar() {
	if r := sizes.Responsive; r != nil && r.narrow {
		r.expanded = !r.expanded
		return
	}
	if sizes.Sidebar > 0 {
		sizes.SidebarExpanded = clampSidebarWidth(sizes.Sidebar)
		sizes.Sidebar = 0
	} else {
		width := sizes.SidebarExpanded
		if width < SidebarMin {
			width = SidebarDefault
		}
		sizes.Sidebar = clampSidebarWidth(width)
	}
}

// sidebarColumns preserves the center's identity across collapse/expand. A
// compact rail has no resize target, so dragging cannot overwrite saved width.
func sidebarColumns(c *ui.Context, sizes *LayoutSizes, sidebar, center func()) {
	viewport, _ := c.Size()
	sizes.observeSidebarViewport(viewport)
	preference := sizes.sidebarPreference()
	open := preference > 0
	width := SidebarCollapsedWidth()
	if open {
		width = clampSidebarWidth(preference)
	}
	row := ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	drag := ui.Local(row, "sidebar-drag", func() paneDrag { return paneDrag{} })
	row.Children(func() {
		sizes.dragging = drag.active
		defer func() { sizes.dragging = false }()
		rail := ui.Box(c).Key("sidebar-column").Width(width).Shrink(0).Clip()
		if !drag.active {
			rail.Transition(shellTrackTransition)
		}
		rail.Children(sidebar)
		dividerWidth := SplitterWidth
		if width == 0 {
			dividerWidth = 0
		}
		div := ui.Box(c).Key("sidebar-divider").Width(dividerWidth).Shrink(0)
		if !drag.active {
			div.Transition(shellTrackTransition)
		}
		if open {
			div.Focusable().Role(ui.RoleSplitter).Label("Sidebar divider").Range(float64(SidebarMin), float64(SidebarMax), float64(width)).Step(10)
			if div.Shortcut(0, ui.KeyLeft) {
				sizes.Sidebar = clampSidebarWidth(width - 10)
			}
			if div.Shortcut(0, ui.KeyRight) {
				sizes.Sidebar = clampSidebarWidth(width + 10)
			}
		}
		main := ui.Box(c).Key("main-column").Grow(1).MinWidth(0).Clip()
		if !drag.active {
			main.Transition(shellTrackTransition)
		}
		main.Children(center)
		div.Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, c.Theme().Border, 0) })
		if !open {
			drag.active = false
			return
		}
		const grip = float32(6)
		handle := ui.Box(c).Absolute().Top(0).Bottom(0).Left(width - (grip-SplitterWidth)/2).Width(grip).Cursor(ui.CursorResizeEW)
		if !drag.active {
			handle.Transition(shellTrackTransition)
		}
		dx, _, held := handle.Dragged()
		if held {
			div.Focus()
			if !drag.active {
				drag.active = true
				drag.base = width
				drag.delta = 0
			}
			if dx != 0 {
				drag.delta += dx
				sizes.Sidebar = clampSidebarWidth(drag.base + drag.delta)
			}
		} else {
			drag.active = false
		}
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if div.FocusVisible() {
				p.FocusRing(r, [4]float32{})
			}
		})
	})
}
