package ui

import (
	"slices"
	"testing"
	"time"
)

// menuNear reports whether two lengths in DIPs are the same, give or take
// what float32 arithmetic loses.
func menuNear(a, b float32) bool { return abs32(a-b) < 0.01 }

// menuNodes returns the nodes under root with role, in tree order.
func menuNodes(root *node, role Role) []*node {
	var out []*node
	var walk func(*node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if n.role == role {
			out = append(out, n)
		}
		for k := n.first; k != nil; k = k.next {
			walk(k)
		}
	}
	walk(root)
	return out
}

// menuChildren returns the children of panel with role, its rows.
func menuChildren(panel *node, role Role) []*node {
	var out []*node
	for k := panel.first; k != nil; k = k.next {
		if k.role == role {
			out = append(out, k)
		}
	}
	return out
}

// menuText returns the first text node under root showing s.
func menuText(root *node, s string) *node {
	if root == nil {
		return nil
	}
	if root.kind == kindText && root.text == s {
		return root
	}
	for k := root.first; k != nil; k = k.next {
		if n := menuText(k, s); n != nil {
			return n
		}
	}
	return nil
}

// TestHairline checks that a hairline is whole device pixels, at least
// one, with the alpha that keeps its ink that of half a DIP.
func TestHairline(t *testing.T) {
	for _, c := range []struct{ scale, width, alpha float32 }{
		{1, 1, 0.5},
		{2, 0.5, 1},
		{1.5, 2.0 / 3, 0.75},
		{3, 2.0 / 3, 0.75},
		{0, 1, 0.5},
	} {
		w, a := hairline(c.scale)
		if !menuNear(w, c.width) || !menuNear(a, c.alpha) {
			t.Errorf("at scale %v: a hairline %v wide at %v, want %v at %v", c.scale, w, a, c.width, c.alpha)
		}
	}
}

// TestPlaceMenu checks where a menu opens from what it opens from, in an
// 800×600 window: 4 below or above it, flipping where the other side has
// the room, from its left or right edge, and 12 inside the window.
func TestPlaceMenu(t *testing.T) {
	for _, c := range []struct {
		name       string
		anchor     Rect
		w, h       float32
		above, end bool
		x, y       float32
	}{
		{"below", Rect{100, 200, 80, 30}, 150, 100, false, false, 100, 234},
		{"above", Rect{100, 200, 80, 30}, 150, 100, true, false, 100, 96},
		{"flips up", Rect{100, 500, 80, 30}, 150, 100, false, false, 100, 396},
		{"flips down", Rect{100, 60, 80, 30}, 150, 100, true, false, 100, 94},
		{"from the end", Rect{100, 200, 80, 30}, 150, 100, false, true, 30, 234},
		{"kept from the right", Rect{700, 200, 80, 30}, 150, 100, false, false, 638, 234},
		{"kept from the left", Rect{20, 200, 40, 30}, 150, 100, false, true, 12, 234},
		{"from a point", Rect{100, 500, 0, 0}, 150, 300, false, false, 100, 196},
		{"taller than the window", Rect{100, 500, 0, 0}, 150, 700, false, false, 100, 12},
	} {
		x, y := placeMenu(c.anchor, c.w, c.h, 800, 600, c.above, c.end)
		if !menuNear(x, c.x) || !menuNear(y, c.y) {
			t.Errorf("%s: at (%v, %v), want (%v, %v)", c.name, x, y, c.x, c.y)
		}
	}
}

// TestPlaceSubmenu checks where a submenu opens beside its row, in an
// 800×600 window: 10 right of it, or left where only that fits, its bottom
// 4 below the row's, and 12 inside the window.
func TestPlaceSubmenu(t *testing.T) {
	for _, c := range []struct {
		name string
		row  Rect
		w, h float32
		x, y float32
	}{
		{"right", Rect{100, 300, 200, 34}, 180, 180, 310, 158},
		{"left", Rect{600, 300, 200, 34}, 180, 180, 410, 158},
		{"neither fits", Rect{20, 300, 700, 34}, 180, 180, 608, 158},
		{"kept from the top", Rect{100, 20, 200, 34}, 180, 180, 310, 12},
	} {
		x, y := placeSubmenu(c.row, c.w, c.h, 800, 600)
		if !menuNear(x, c.x) || !menuNear(y, c.y) {
			t.Errorf("%s: at (%v, %v), want (%v, %v)", c.name, x, y, c.x, c.y)
		}
	}
}

// TestMenuRiseAt checks that a menu rises 4 DIPs into place over 160ms,
// easing out.
func TestMenuRiseAt(t *testing.T) {
	for _, c := range []struct {
		age    time.Duration
		rise   float32
		moving bool
	}{
		{-time.Millisecond, 4, true},
		{0, 4, true},
		{80 * time.Millisecond, 0.5, true},
		{160 * time.Millisecond, 0, false},
		{time.Second, 0, false},
	} {
		rise, moving := menuRiseAt(c.age)
		if !menuNear(rise, c.rise) || moving != c.moving {
			t.Errorf("at %v: %v below, moving %v; want %v, %v", c.age, rise, moving, c.rise, c.moving)
		}
	}
}

// TestMenuHoverFill checks the fill of the row the pointer or the arrows
// are on: the menu's Hover, or for a dangerous item the danger color at
// 5%, or 15% in the dark.
func TestMenuHoverFill(t *testing.T) {
	light, dark := LightTheme(), DarkTheme()
	if got := menuHoverFill(&context{theme: light}, false); got != light.SurfaceHover {
		t.Errorf("hover: %v, want %v", got, light.SurfaceHover)
	}
	// The products' danger hovers: #ec13130d in the light, #f25a5a26 in the
	// dark, the danger color at 5% and 15%.
	if got := menuHoverFill(&context{theme: light}, true); got != light.Danger.Alpha(0.05) || got.A != 13 {
		t.Errorf("danger hover: %v, want %v at alpha 13", got, light.Danger.Alpha(0.05))
	}
	if got := menuHoverFill(&context{theme: dark}, true); got != dark.Danger.Alpha(0.15) || got.A != 38 {
		t.Errorf("dark danger hover: %v, want %v at alpha 38", got, dark.Danger.Alpha(0.15))
	}
	themed := *light
	themed.Menu.Hover = RGB(1, 2, 3)
	if got := menuHoverFill(&context{theme: &themed}, false); got != RGB(1, 2, 3) {
		t.Errorf("a theme's menu hover: %v", got)
	}
}

// menuButtonTester returns a tester of a window 600×400 with the menu
// button Actions in its middle, its menu built by build, and a function
// returning the button's box.
func menuButtonTester(build func(m *Menu)) (*Tester, func() Rect) {
	var button *node
	tt := menuOverlayTester(func(c *context) {
		coreBox(c).Fill().Center().Children(func() {
			button = coreMenuButton(c, "Actions", build)
		})
	}, 600, 400)
	return tt, func() Rect { return button.Bounds() }
}

// TestMenuOverlayGeometry checks a button's menu against the product's: 4
// below the button, from its left, 144 wide for short labels, padded by 4
// and rounded by 16, with a hairline over two soft shadows, of rows 34
// high rounded by 12.
func TestMenuOverlayGeometry(t *testing.T) {
	tt, button := menuButtonTester(func(m *Menu) {
		m.Item("Rename")
		m.Item("Delete")
	})
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	panels := menuNodes(tt.rt.c.root, RoleMenu)
	if len(panels) != 1 {
		t.Fatalf("%d menu panels", len(panels))
	}
	panel, b := panels[0], button()
	r := panel.Bounds()
	if !near(r.X, b.X) || !near(r.Y, b.Y+b.H+menuGap) || !menuNear(r.W, menuMinWidth) {
		t.Errorf("the menu at %v, under the button at %v: want it 4 below, from its left, 144 wide", r, b)
	}
	if panel.radius != [4]float32{16, 16, 16, 16} || panel.pad != [4]float32{4, 4, 4, 4} {
		t.Errorf("the panel is rounded by %v and padded by %v", panel.radius, panel.pad)
	}
	// At scale 1 the hairline is a pixel at half the border's alpha.
	want := []shadow{
		{0, 0, 20, 0, RGBA(0, 0, 0, 0.05)},
		{0, 3, 8, 0, RGBA(0, 0, 0, 0.04)},
		{0, 0, 0, 1, tt.rt.c.theme.Border.Alpha(0.5)},
	}
	if !slices.Equal(panel.shadows, want) {
		t.Errorf("the panel's shadows: %v, want %v", panel.shadows, want)
	}
	rows := menuChildren(panel, RoleMenuItem)
	if len(rows) != 2 {
		t.Fatalf("%d rows", len(rows))
	}
	for i, row := range rows {
		rr := row.Bounds()
		if !menuNear(rr.H, menuRowHeight) || !menuNear(rr.X, r.X+menuPadding) || !menuNear(rr.W, r.W-2*menuPadding) {
			t.Errorf("row %d at %v, in the panel at %v: want it 34 high, padded by 4", i, rr, r)
		}
		if !menuNear(rr.Y, r.Y+menuPadding+float32(i)*menuRowHeight) {
			t.Errorf("row %d at y %v, want %v", i, rr.Y, r.Y+menuPadding+float32(i)*menuRowHeight)
		}
		if row.radius != [4]float32{12, 12, 12, 12} {
			t.Errorf("row %d is rounded by %v", i, row.radius)
		}
	}
	if !menuNear(r.H, 2*menuRowHeight+2*menuPadding) {
		t.Errorf("the panel is %v high, want its rows and padding", r.H)
	}
}

// TestMenuOverlayCapsItsWidth checks that a menu with a long label is at
// most 360 wide, the label cut to fit.
func TestMenuOverlayCapsItsWidth(t *testing.T) {
	long := "A label much longer than a menu is ever wide, which the menu cuts short rather than grow without end"
	tt, _ := menuButtonTester(func(m *Menu) { m.Item(long) })
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	panels := menuNodes(tt.rt.c.root, RoleMenu)
	if len(panels) != 1 {
		t.Fatalf("%d menu panels", len(panels))
	}
	r := panels[0].Bounds()
	if !menuNear(r.W, menuMaxWidth) {
		t.Errorf("the menu is %v wide, want 360", r.W)
	}
	label := menuText(panels[0], long)
	if label == nil {
		t.Fatal("no label")
	}
	if w := label.Bounds().W; w > menuMaxWidth-2*menuPadding-16+0.01 {
		t.Errorf("the label is %v wide, more than its row holds", w)
	}
}

// TestMenuOverlayOpenAboveAndAlignEnd checks that a menu asked to open
// above its button and from its right edge does, where it fits.
func TestMenuOverlayOpenAboveAndAlignEnd(t *testing.T) {
	tt, button := menuButtonTester(func(m *Menu) {
		m.OpenAbove(true)
		m.AlignEnd(true)
		m.Item("Rename")
		m.Item("Delete")
	})
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	panels := menuNodes(tt.rt.c.root, RoleMenu)
	if len(panels) != 1 {
		t.Fatalf("%d menu panels", len(panels))
	}
	r, b := panels[0].Bounds(), button()
	if !near(r.Y+r.H, b.Y-menuGap) || !near(r.X+r.W, b.X+b.W) {
		t.Errorf("the menu at %v, over the button at %v: want it 4 above, ending at its right", r, b)
	}
}

// TestMenuRisesIntoPlace checks that a menu opens 4 below its place and
// rises into it over 160ms, easing out, unless the user asks for less
// motion.
func TestMenuRisesIntoPlace(t *testing.T) {
	var button *node
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Center().Children(func() {
			button = coreMenuButton(c, "Actions", func(m *Menu) { m.Item("Rename") })
		})
	}, 600, 400)
	tt.rt.host = menuOverlayFixtureHost{tt.h}
	base := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	tt.SetTime(base)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	b := button.Bounds()
	place := round(b.Y + b.H + menuGap)
	at := func() float32 {
		panels := menuNodes(tt.rt.c.root, RoleMenu)
		if len(panels) != 1 {
			t.Fatalf("%d menu panels", len(panels))
		}
		return panels[0].Bounds().Y
	}
	if y := at(); !menuNear(y, place+menuRise) {
		t.Errorf("as it opens, the menu is at y %v, want %v", y, place+menuRise)
	}
	tt.SetTime(base.Add(80 * time.Millisecond))
	if y := at(); !menuNear(y, place+0.5) {
		t.Errorf("halfway, the menu is at y %v, want %v", y, place+0.5)
	}
	tt.SetTime(base.Add(menuRiseTime))
	if y := at(); !menuNear(y, place) {
		t.Errorf("risen, the menu is at y %v, want %v", y, place)
	}
	tt.Key(0, KeyEscape)
	if tt.rt.menu.overlay != nil {
		t.Fatal("Escape left the menu open")
	}
	tt.SetPreferences(Preferences{ReduceMotion: true})
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	if y := at(); !menuNear(y, place) {
		t.Errorf("with less motion, the menu opens at y %v, want %v", y, place)
	}
}

// TestMenuOverlayHighlight checks that a menu the pointer opens has no
// item highlighted until the arrows or the pointer go to one, that the
// arrows skip what is disabled, and that Up from none goes to the last.
func TestMenuOverlayHighlight(t *testing.T) {
	tt, _ := menuButtonTester(func(m *Menu) {
		m.Item("Rename")
		m.Item("Locked").Disabled(true)
		m.Item("Delete")
	})
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	if tt.rt.menu.overlay == nil {
		t.Fatal("the menu did not open")
	}
	if i := tt.rt.menu.index; i != -1 {
		t.Fatalf("the pointer opened the menu on item %d", i)
	}
	tt.Key(0, KeyDown)
	if i := tt.rt.menu.index; i != 0 {
		t.Errorf("Down went to item %d, want 0", i)
	}
	tt.Key(0, KeyDown)
	if i := tt.rt.menu.index; i != 2 {
		t.Errorf("Down went to item %d, want 2, past the disabled one", i)
	}
	tt.Key(0, KeyEscape)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, KeyUp)
	if i := tt.rt.menu.index; i != 2 {
		t.Errorf("Up from no item went to item %d, want the last, 2", i)
	}
	rows := menuChildren(menuNodes(tt.rt.c.root, RoleMenu)[0], RoleMenuItem)
	if len(rows) != 3 {
		t.Fatalf("%d rows", len(rows))
	}
	if hover := tt.rt.c.theme.menuStyle().Hover; rows[2].bg != hover || rows[0].bg != (Color{}) {
		t.Errorf("rows filled %v, %v, want only the highlighted one, by %v", rows[0].bg, rows[2].bg, hover)
	}
	rename := rows[0].Bounds()
	tt.Move(rename.X+rename.W/2, rename.Y+rename.H/2)
	if i := tt.rt.menu.index; i != 0 {
		t.Errorf("the pointer on Rename highlights item %d", i)
	}
	tt.Move(5, 5)
	if i := tt.rt.menu.index; i != -1 {
		t.Errorf("the pointer off the items leaves item %d highlighted", i)
	}
}

// TestMenuOverlayKeyboardOpensOnFirstItem checks that a menu the keyboard
// opens highlights its first item that can be chosen, and that one the
// pointer opens from the same focused button highlights none.
func TestMenuOverlayKeyboardOpensOnFirstItem(t *testing.T) {
	tt, button := menuButtonTester(func(m *Menu) {
		m.Item("Locked").Disabled(true)
		m.Item("Rename")
		m.Item("Delete")
	})
	tt.Key(0, KeyTab)
	tt.Key(0, KeyDown)
	if tt.rt.menu.overlay == nil {
		t.Fatal("the keyboard did not open the menu")
	}
	if i := tt.rt.menu.index; i != 1 {
		t.Errorf("the keyboard opened the menu on item %d, want 1, Rename", i)
	}
	tt.Key(0, KeyEscape)
	if tt.rt.menu.overlay != nil {
		t.Fatal("Escape left the menu open")
	}
	// The button keeps the focus, and its ring, as the pointer presses it.
	if !tt.rt.focusVisible {
		t.Fatal("the button lost its focus ring")
	}
	b := button()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	if tt.rt.menu.overlay == nil {
		t.Fatal("the pointer did not open the menu")
	}
	if i := tt.rt.menu.index; i != -1 {
		t.Errorf("the pointer opened the menu on item %d", i)
	}
}

// TestMenuOverlaySeparators checks that a menu draws one line between
// groups of items, none before the first, and one for several in a row.
func TestMenuOverlaySeparators(t *testing.T) {
	tt, _ := menuButtonTester(func(m *Menu) {
		m.Separator()
		m.Item("A")
		m.Separator()
		m.Separator()
		m.Item("B")
	})
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	panels := menuNodes(tt.rt.c.root, RoleMenu)
	if len(panels) != 1 {
		t.Fatalf("%d menu panels", len(panels))
	}
	var got []string
	for k := panels[0].first; k != nil; k = k.next {
		switch {
		case k.role == RoleMenuItem:
			got = append(got, "item")
		case menuNear(k.Bounds().H, 0.5):
			got = append(got, "line")
		default:
			got = append(got, "other")
		}
	}
	if want := []string{"item", "line", "item"}; !slices.Equal(got, want) {
		t.Errorf("the menu holds %v, want %v", got, want)
	}
}

// menuMaterialProbe is a material that paints red over its box, and
// records the box.
type menuMaterialProbe struct{ boxes *[]Rect }

func (m menuMaterialProbe) PaintMaterial(p *Painter, box Rect, radii [4]float32) {
	*m.boxes = append(*m.boxes, box)
	p.Fill(box, RGB(255, 0, 0), 0)
}

// TestMenuOverlayPaintsTheThemesMaterial checks that a theme's menu
// Material paints under the panel's Fill, which it shows through as far as
// the fill is translucent.
func TestMenuOverlayPaintsTheThemesMaterial(t *testing.T) {
	var boxes []Rect
	tt := menuOverlayTester(func(c *context) {
		th := *c.Theme()
		th.Menu.Material = menuMaterialProbe{&boxes}
		th.Menu.Fill = RGBA(0, 0, 255, 0.5)
		c.SetTheme(&th)
		coreBox(c).Fill().Center().Children(func() {
			coreMenuButton(c, "Actions", func(m *Menu) { m.Item("Rename") })
		})
	}, 600, 400)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	img := tt.Image()
	panels := menuNodes(tt.rt.c.root, RoleMenu)
	if len(panels) != 1 {
		t.Fatalf("%d menu panels", len(panels))
	}
	r := panels[0].Bounds()
	if len(boxes) == 0 || !nearRect(boxes[len(boxes)-1], r) {
		t.Fatalf("the material painted over %v, want the panel's box %v", boxes, r)
	}
	rows := menuChildren(panels[0], RoleMenuItem)
	if len(rows) != 1 {
		t.Fatalf("%d rows", len(rows))
	}
	row := rows[0].Bounds()
	// Half blue over the material's red, in the panel's padding.
	if got := img.RGBAAt(int(r.X+2), int(row.Y+row.H/2)); !nearColor(got, RGB(128, 0, 128)) {
		t.Errorf("the panel shows %v, want half blue over red", got)
	}
}

// TestSelectPopupLooksLikeAMenu checks that a select's popup is a menu's
// panel, 4 below the select, of rows 34 high, the chosen one with its check
// mark.
func TestSelectPopupLooksLikeAMenu(t *testing.T) {
	size := "Medium"
	var trigger *node
	tt := coreNewTester(func(c *context) {
		coreRow(c).Padding(40).Children(func() {
			trigger = coreSelect(c, &size, []string{"Small", "Medium", "Large"})
		})
	}, 600, 400)
	if err := tt.Click("Medium"); err != nil {
		t.Fatal(err)
	}
	panels := menuNodes(tt.rt.c.root, RolePopup)
	if len(panels) != 1 {
		t.Fatalf("%d popups", len(panels))
	}
	panel, b := panels[0], trigger.Bounds()
	r := panel.Bounds()
	if !near(r.X, b.X) || !near(r.Y, b.Y+b.H+menuGap) || r.W < menuMinWidth-0.01 {
		t.Errorf("the popup at %v, under the select at %v: want it 4 below, from its left, at least 144 wide", r, b)
	}
	if panel.radius != [4]float32{16, 16, 16, 16} || panel.pad != [4]float32{4, 4, 4, 4} {
		t.Errorf("the popup is rounded by %v and padded by %v", panel.radius, panel.pad)
	}
	n := 0
	for k := panel.first; k != nil; k = k.next {
		n++
		if rr := k.Bounds(); !menuNear(rr.H, menuRowHeight) || k.radius != [4]float32{12, 12, 12, 12} {
			t.Errorf("an option at %v, rounded by %v: want it 34 high, rounded by 12", rr, k.radius)
		}
	}
	if n != 3 {
		t.Errorf("%d options", n)
	}
	if !menuRowChecked(panel, "Medium") || menuRowChecked(panel, "Small") {
		t.Error("the check mark is not on the chosen option alone")
	}
}
