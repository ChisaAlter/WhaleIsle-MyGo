package ui

import (
	"fmt"
	"math"
	"time"
)

// The geometry of menus, select popups and popovers, in DIPs, as the
// installed product's menus: a panel padded by 4 and rounded by 16, at
// least 144 wide (163 for a submenu) and at most 360, of rows at least 34
// high rounded by 12, 4 from what it opens from and 12 from the window's
// edges. A submenu opens 10 beside its row, its bottom 4 below the row's.
const (
	menuPadding    = 4
	menuRadius     = 16
	menuMinWidth   = 144
	submenuMin     = 163
	menuMaxWidth   = 360
	menuMargin     = 12
	menuGap        = 4
	submenuGap     = 10
	menuRowHeight  = 34
	menuRowRadius  = 12
	menuRise       = 4
	menuDisabled   = 0.4  // the opacity of a disabled item
	disabledFactor = 0.48 // what the painter dims a disabled element by
)

// menuRiseTime is how long a menu takes to rise into place as it opens. Its
// opacity does not fade.
const menuRiseTime = 160 * time.Millisecond

// menuCheck is the check mark by a checked item, the product's 16×16 check
// drawn with a stroke of 1.
var menuCheck = MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M2.25 8.5L5.49732 11.7473C5.90519 12.1552 6.57263 12.1344 6.95426 11.7018L13.75 4" stroke="currentColor" stroke-width="1"/></svg>`))

// hairline returns the width in DIPs of a line half a DIP thick drawn as
// whole device pixels at scale s, at least one, and the alpha that keeps
// its ink that of half a DIP: one pixel at half alpha at scale 1, one
// pixel at full alpha at scale 2.
func hairline(s float32) (width, alpha float32) {
	if s <= 0 {
		s = 1
	}
	px := max(1, float32(math.Round(float64(s/2))))
	return px / s, min(1, s/2/px)
}

// menuMaterial fills a menu's panel: under, a backdrop blur, if set, and
// fill over it, which lets the blur show through as far as it is
// translucent.
type menuMaterial struct {
	under Material
	fill  Color
}

func (m menuMaterial) PaintMaterial(p *Painter, box Rect, radii [4]float32) {
	if m.under != nil {
		m.under.PaintMaterial(p, box, radii)
	}
	if m.fill.A > 0 {
		p.fill(box, radii, m.fill, 0, Color{})
	}
}

// String describes the material, as the inspector shows it.
func (m menuMaterial) String() string {
	s := fmt.Sprintf("menu rgba(%d %d %d / %.2f)", m.fill.R, m.fill.G, m.fill.B, float32(m.fill.A)/255)
	if u, ok := m.under.(fmt.Stringer); ok {
		s += " over " + u.String()
	}
	return s
}

// styleMenuPanel gives the panel of a menu, a select's popup or a popover
// the theme's menu look (Theme.Menu): its padding and corners, its fill
// over its material, and its elevation, a hairline around it over two soft
// shadows.
func styleMenuPanel(c *context, panel *node) {
	st := c.theme.menuStyle()
	panel.Padding(menuPadding).Gap(0).Radius(menuRadius).BorderWidth(0)
	if st.Material == nil {
		// An opaque fill keeps text over it sharp, as subpixel text needs.
		panel.Background(st.Fill)
	} else {
		under := st.Material
		if b, ok := under.(nodeMaterialBuilder); ok {
			under = b.BuildMaterial(panel)
		} else if b, ok := under.(MaterialBuilder); ok {
			under = b.BuildMaterial(wrapElement(panel))
		}
		panel.Material(menuMaterial{under: under, fill: st.Fill})
	}
	// Shadows paint in order, so the hairline goes over the soft ones.
	panel.Shadow(0, 0, 20, 0, RGBA(0, 0, 0, 0.05))
	panel.Shadow(0, 3, 8, 0, RGBA(0, 0, 0, 0.04))
	if w, a := hairline(c.DeviceScale()); st.Stroke.A > 0 {
		panel.Shadow(0, 0, 0, w, st.Stroke.Alpha(a))
	}
}

// MenuPanel gives the element the look of the panels of menus, select
// popups and popovers, from the theme's Menu: padded by 4 and rounded by
// 16, filled over the menu's material, with a hairline around it and a
// soft shadow under it. Set Radius or Padding after it to change those.
func (e Element) MenuPanel() Element {
	if n := e.node(); n != nil {
		styleMenuPanel(n.c, n)
	}
	return e
}

// placeMenu returns where a menu panel w×h goes in a window vw×vh, opening
// from anchor, the box of its button or, without a size, the point it was
// asked at: menuGap below it, or above with above, flipping to the other
// side when it does not fit there and has more room on it; from the
// anchor's left, or with end its right; and menuMargin inside the window.
func placeMenu(anchor Rect, w, h, vw, vh float32, above, end bool) (x, y float32) {
	below := vh - menuMargin - (anchor.Y + anchor.H + menuGap)
	over := anchor.Y - menuGap - menuMargin
	if above && h > over && below > over || !above && h > below && over > below {
		above = !above
	}
	y = anchor.Y + anchor.H + menuGap
	if above {
		y = anchor.Y - menuGap - h
	}
	x = anchor.X
	if end {
		x = anchor.X + anchor.W - w
	}
	return keepIn(x, w, vw), keepIn(y, h, vh)
}

// placeSubmenu returns where the panel w×h of the submenu of the row box
// goes in a window vw×vh: submenuGap right of the row, or left where it
// does not fit there and fits better on the left, with its bottom
// menuPadding below the row's, so that its last item lines up with the
// row, and menuMargin inside the window.
func placeSubmenu(row Rect, w, h, vw, vh float32) (x, y float32) {
	x = row.X + row.W + submenuGap
	if x+w > vw-menuMargin && row.X-submenuGap-w >= menuMargin {
		x = row.X - submenuGap - w
	}
	return keepIn(x, w, vw), keepIn(row.Y+row.H+menuPadding-h, h, vh)
}

// keepIn moves a span of size from v to keep it menuMargin inside a window
// edge to edge limit wide, the start edge winning where it cannot fit.
func keepIn(v, size, limit float32) float32 {
	return max(menuMargin, min(v, limit-menuMargin-size))
}

// menuRiseAt returns how far below its place a menu shown for age is, as
// it rises into place over menuRiseTime, and whether it still moves.
func menuRiseAt(age time.Duration) (float32, bool) {
	if age >= menuRiseTime {
		return 0, false
	}
	progress := EaseOut(max(float32(age), 0) / float32(menuRiseTime))
	return menuRise * (1 - progress), true
}
