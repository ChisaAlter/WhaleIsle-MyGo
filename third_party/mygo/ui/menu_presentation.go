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

func menuItemContent(c *context, shown *shownMenu, item *platform.MenuItem, siblings []*platform.MenuItem) {
	t := c.theme
	style := shown.presentation[item.ID]
	ink, secondary := t.Text, t.TextMuted
	if style.danger {
		ink, secondary = t.Danger, t.Danger
	}
	if !item.Enabled {
		ink, secondary = t.TextMuted, t.TextMuted
	}
	checks, icons := false, false
	for _, sibling := range siblings {
		if !sibling.Visible || sibling.Type == platform.MenuItemSeparator {
			continue
		}
		checks = checks || sibling.Type == platform.MenuItemCheckbox || sibling.Type == platform.MenuItemRadio || sibling.Checked
		icons = icons || shown.presentation[sibling.ID].icon != nil
	}
	if checks {
		mark := ""
		if item.Checked {
			mark = "✓"
		}
		coreText(c, mark).Width(16).Shrink(0).FontSize(12).TextColor(ink)
	}
	if icons {
		if style.icon != nil {
			coreIcon(c, style.icon).Size(16, 16).Shrink(0).TextColor(ink)
		} else {
			coreBox(c).Size(16, 16).Shrink(0)
		}
	}
	coreText(c, item.Label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(ink)
	shortcut := item.Accelerator
	if style.shortcutSet {
		shortcut = style.shortcutLabel
	}
	if shortcut != "" {
		coreText(c, shortcut).Shrink(0).FontSize(11).TextColor(secondary)
	}
	if item.Submenu != nil {
		coreText(c, "›").Shrink(0).FontSize(14).TextColor(secondary)
	}
}
