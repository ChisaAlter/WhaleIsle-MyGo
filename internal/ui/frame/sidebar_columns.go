package frame

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// SidebarResponsive mirrors AppFrame's desktop 1024px breakpoint and the phone
// drawer band below PHONE_MAX. It is shared across copied frame models but is
// never a saved layout preference.
type SidebarResponsive struct {
	viewport                    float32
	narrow, expanded, rightOpen bool
	landscape, orientationKnown bool
}

// ui-layout columns.ts: the sidebar leaves the grid for an overlay drawer below
// PHONE_MAX and the drawer never exceeds PHONE_DRAWER.
const (
	phoneMax    = float32(768)
	phoneDrawer = float32(320)
	// The installed AppFrame collapses desktop navigation below 1024 even on
	// landscape screens. Its narrower phone drawer is a separate decision.
	autoCollapseWidth = float32(1024)
)

func (sizes *LayoutSizes) observeSidebarViewport(width float32) {
	if sizes.Responsive == nil {
		sizes.Responsive = &SidebarResponsive{}
	}
	// AppFrame's desktop branch is narrow below 1024 regardless of screen
	// orientation. The manual override is transient and resets only when that
	// band changes; the persisted Sidebar width remains untouched.
	narrow := width < autoCollapseWidth
	if sizes.Responsive.narrow != narrow {
		sizes.Responsive.expanded = false
	}
	sizes.Responsive.narrow = narrow
	sizes.Responsive.viewport = width
}

// observeSidebarGeometry follows screen orientation when a native owned window
// exists. The reference falls back to viewport aspect only without screen data;
// headless tests take that same fallback. Width-only solvers keep the most recent
// observation instead of inventing a screen orientation.
func (sizes *LayoutSizes) observeSidebarGeometry(width, height float32) {
	landscape, known := displayLandscape()
	sizes.observeSidebarOrientation(width, height, landscape, known)
}

func (sizes *LayoutSizes) observeSidebarOrientation(width, height float32, landscape, known bool) {
	sizes.observeSidebarViewport(width)
	if !known && width > 0 && height > 0 {
		landscape, known = width > height, true
	}
	sizes.Responsive.landscape, sizes.Responsive.orientationKnown = landscape, known
}

// drawerMode is AppFrame's phone band: below 768 and not landscape. A caller
// that supplies only a width retains the historical unknown-screen contract.
func (sizes *LayoutSizes) drawerMode() bool {
	r := sizes.Responsive
	return r != nil && r.viewport > 0 && r.viewport < phoneMax && (!r.orientationKnown || !r.landscape)
}

// drawerWidth is the installed min(320, max(264, viewport-48)). Below 312
// the navigation keeps its 264 DIP minimum; the shell clips it at tiny widths.
func (sizes *LayoutSizes) drawerWidth() float32 {
	viewport := float32(0)
	if sizes.Responsive != nil {
		viewport = sizes.Responsive.viewport
	}
	return min(phoneDrawer, max(SidebarMin, viewport-48))
}

// sidebarCollapsed reports whether the navigation paints its compact rail. In
// the drawer band the open drawer shows the expanded content.
func (sizes *LayoutSizes) sidebarCollapsed() bool {
	if r := sizes.Responsive; r != nil && r.narrow {
		return !r.expanded
	}
	return sizes.sidebarPreference() <= 0
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
		// The source wide toggle reopens at its default, independently of the
		// former width retained for the closing animation and legacy recovery.
		sizes.Sidebar = SidebarDefault
	}
}

// sidebarColumns preserves the center's identity across collapse/expand. A
// compact rail has no resize target, so dragging cannot overwrite saved width.
func sidebarColumns(c *ui.Context, sizes *LayoutSizes, sidebar, center func()) {
	viewport, height := viewportDIP(c)
	sizes.observeSidebarGeometry(viewport, height)
	if sizes.drawerMode() {
		sidebarDrawer(c, sizes, sidebar, center, viewport)
		return
	}
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
		animate := !c.Preferences().ReduceMotion && !drag.active
		rail := ui.Box(c).Key("sidebar-column").Width(width).Shrink(0).Clip()
		if animate {
			rail.Transition(shellTrackTransition)
		}
		rail.Children(sidebar)
		dividerWidth := SplitterWidth
		if width == 0 {
			dividerWidth = 0
		}
		div := ui.Box(c).Key("sidebar-divider").Width(dividerWidth).Shrink(0)
		if animate {
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
		if animate {
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
		if animate {
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

// sidebarDrawer is the phone band's overlay navigation (AppFrame data-phone):
// the centre column keeps the full width, the sidebar slides in over it from the
// left at min(320, max(264, 100%-48)); a floating button reopens it once closed.
func sidebarDrawer(c *ui.Context, sizes *LayoutSizes, sidebar, center func(), viewport float32) {
	open := sizes.Responsive != nil && sizes.Responsive.expanded
	width := sizes.drawerWidth()
	animate := !c.Preferences().ReduceMotion
	shell := ui.Box(c).Key("sidebar-drawer-shell").Grow(1).MinWidth(0).MinHeight(0).Clip()
	shell.Children(func() {
		main := ui.Box(c).Key("main-column").Fill().MinWidth(0).MinHeight(0).Clip()
		main.Children(center)

		if open {
			backdrop := ui.Box(c).Key("sidebar-drawer-backdrop").Absolute().Top(0).Bottom(0).Left(0).Right(0)
			backdrop.Background(ui.RGBA(0, 0, 0, .28)).Label(i18n.T(i18n.KeySidebarCollapse))
			if backdrop.Clicked() {
				sizes.ToggleSidebar()
			}
		}

		left := float32(0)
		if !open {
			left = -width
		}
		drawer := ui.Box(c).Key("sidebar-drawer").Absolute().Top(0).Bottom(0).Left(left).Width(width).Clip()
		if animate {
			drawer.Transition(shellTrackTransition)
		}
		// Closed drawers stop building immediately: no off-canvas keyboard owner
		// or focusable control survives behind the conversation.
		if open {
			drawer.Children(sidebar)
		}

		if !open {
			// AppFrame .phoneMenu: the only way back into navigation once the
			// drawer has left the grid.
			open := ui.ButtonBase(c.Key("sidebar-drawer-open")).Absolute().Left(10).Top(10).Size(36, 36).
				Radius(10).Padding(0).BorderWidth(0).Background(c.Theme().Surface).
				Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, .14)).Center().
				Label(i18n.T(i18n.KeySidebarExpand)).Tooltip(i18n.T(i18n.KeySidebarExpand))
			open.Children(func() { ui.Icon(c, drawerMenuArtwork).Size(18, 18).TextColor(c.Theme().Text) })
			if open.Clicked() {
				sizes.ToggleSidebar()
			}
		}
	})
}

// drawerMenuArtwork is the phone menu glyph: three 1px rules.
var drawerMenuArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"><path d="M3 5h12M3 9h12M3 13h12"/></svg>`))
