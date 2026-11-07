package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"time"
)

// menuItemPresentation stays in the UI snapshot, not the platform menu model.
// Item IDs are the menu builder's existing sequence, shared by submenus.
type menuItemPresentation struct {
	icon          *SVG
	danger        bool
	shortcutLabel string
	shortcutSet   bool
}

// menuPointerInside uses the last committed panel box, before this pass lays
// out its overlay. Hover flags make entering/leaving padding request a frame.
func menuPointerInside(rt *engine, panel *node) bool {
	if panel == nil || !rt.pointerIn || panel.st == nil {
		return false
	}
	s := panel.st
	return (Rect{s.vx, s.vy, s.vw, s.vh}).Contains(rt.pointerX, rt.pointerY)
}

func (shown *shownMenu) pointerLeaveDue(c *context, inside bool) bool {
	if inside {
		shown.pointerInside = true
		shown.leaveAt = time.Time{}
		return false
	}
	if shown.pointerInside {
		shown.pointerInside = false
		shown.leaveAt = c.Now().Add(200 * time.Millisecond)
	}
	if shown.leaveAt.IsZero() {
		return false
	}
	if !c.Now().Before(shown.leaveAt) {
		return true
	}
	c.After(shown.leaveAt.Sub(c.Now()))
	return false
}

// menuRow creates the row of a menu item: at least menuRowHeight high,
// padded by 6 and 8 and rounded by menuRowRadius, its content 6 apart, and
// a disabled item's at menuDisabled opacity.
func menuRow(c *context, item *platform.MenuItem) *node {
	row := coreButtonBase(c).Justify(Start).MinHeight(menuRowHeight).Padding(6, 8).Gap(6).Radius(menuRowRadius).
		BorderWidth(0).Background(Color{}).TextColor(c.theme.Text).Role(RoleMenuItem)
	if !item.Enabled {
		// The painter dims a disabled element by disabledFactor; this makes up
		// the rest of menuDisabled.
		row.Disabled(true).Opacity(menuDisabled / disabledFactor)
	}
	return row
}

// menuSeparator creates the line between groups of items: half a DIP high,
// 3 from the items around it and 2 in from the panel's padding, drawn as
// whole device pixels (hairline) in col.
func menuSeparator(c *context, col Color) {
	coreBox(c).Height(.5).Margin(3, 2).Shrink(0).Draw(func(p *Painter, r Rect) {
		s := p.Scale()
		h, a := hairline(s)
		y := round(r.Y*s) / s
		p.Fill(Rect{X: r.X, Y: y, W: r.W, H: h}, col.Alpha(a), 0)
	})
}

// menuHoverFill returns the fill of the row the pointer or the arrow keys
// are on: the theme's menu Hover, or for a danger item a tint of Danger, as
// the installed product's (#ec13130d light, #f25a5a26 dark).
func menuHoverFill(c *context, danger bool) Color {
	t := c.theme
	switch {
	case !danger:
		return t.menuStyle().Hover
	case t.Dark:
		return t.Danger.Alpha(0.15)
	default:
		return t.Danger.Alpha(0.05)
	}
}

// menuItemContent creates what a menu item's row holds: its icon, 14 DIPs,
// its label, 13 on lines of 20, its shortcut, 11 on lines of 16, and the
// check mark of a checked item after them. A danger item's label and icon
// take the theme's Danger.
func menuItemContent(c *context, shown *shownMenu, item *platform.MenuItem) {
	t := c.theme
	st := t.menuStyle()
	style := shown.presentation[item.ID]
	ink, iconInk := t.Text, st.Icon
	if style.danger {
		ink, iconInk = t.Danger, t.Danger
	}
	if style.icon != nil {
		coreIcon(c, style.icon).Size(14, 14).Shrink(0).TextColor(iconInk)
	}
	coreText(c, item.Label).Grow(1).MinWidth(0).SingleLine().FontSize(13).FixedLineHeight(20).TextColor(ink)
	shortcut := item.Accelerator
	if style.shortcutSet {
		shortcut = style.shortcutLabel
	}
	if shortcut != "" {
		coreText(c, shortcut).Shrink(0).SingleLine().FontSize(11).FixedLineHeight(16).TextColor(st.Caption)
	}
	if item.Checked {
		coreIcon(c, menuCheck).Size(14, 14).Shrink(0).TextColor(t.Text)
	}
}
