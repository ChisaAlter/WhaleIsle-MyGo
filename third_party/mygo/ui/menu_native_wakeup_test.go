package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

// Drive only frames the host requested, as a native surface does. Tester.settle
// deliberately runs a frame even when none is requested, so it cannot expose a
// menu promoted to overlay only after the current scene was committed.
func TestMenuPublicContextMenuRightReleaseRequestsVisibleOverlay(t *testing.T) {
	// Less motion: a menu rising into place asks for a frame each frame, more
	// than pump allows.
	h := &headless{w: 400, h: 300, scale: 1, prefs: platform.Preferences{ReduceMotion: true}}
	var open bool
	rt := newRuntime(func(c *context) {
		public := makeContext(c)
		anchor := Box(public).Size(120, 40).Label("Anchor").ContextMenu(func(m *Menu) { m.CloseOnPointerLeave(true); m.Item("Pin").Icon(menuFixtureIcon) })
		open = anchor.MenuOpen()
	}, menuOverlayFixtureHost{h})
	rt.runFrame()
	pump := func() {
		for count := 0; h.requested.Swap(false); count++ {
			if count > 5 {
				t.Fatal("menu requested an unbounded frame loop")
			}
			rt.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame})
		}
	}
	rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, Button: 1, X: 20, Y: 20})
	pump()
	rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, Button: 1, X: 20, Y: 20})
	pump()
	if !open || rt.menu.overlay == nil {
		t.Fatal("public menu owner did not report the right-clicked menu")
	}
	found := false
	for _, label := range rt.labels {
		if label.text == "Pin" {
			found = true
		}
	}
	// Native runtimes need no test label collector; inspect the committed
	// menu role as well so the assertion follows the actual presentation tree.
	var findItem func(*node)
	findItem = func(root *node) {
		if root == nil {
			return
		}
		if root.role == RoleMenuItem {
			found = true
		}
		for child := root.first; child != nil; child = child.next {
			findItem(child)
		}
	}
	findItem(rt.c.root)
	if !found {
		t.Fatal("public right-click reported an open menu but requested frames never presented a menu item")
	}
}

func TestMenuOverlayRequestsPresentationFrameWithInactivePointer(t *testing.T) {
	h := &headless{w: 400, h: 300, scale: 1}
	var owner *state
	rt := newRuntime(func(c *context) {
		anchor := coreBox(c).Size(120, 40).Label("Anchor").ContextMenu(func(m *Menu) {
			m.CloseOnPointerLeave(true)
			m.Item("Pin").Icon(menuFixtureIcon)
			m.Item("Delete").Icon(menuFixtureIcon).Danger(true)
		})
		owner = anchor.st
	}, menuOverlayFixtureHost{h})
	rt.runFrame()
	h.requested.Store(false)
	rt.pointerIn = false
	rt.askMenu(owner, 20, 40, false)
	if !h.requested.Load() {
		t.Fatal("opening event requested no native frame")
	}
	// Consume exactly the requested frame, without an extra Tester event.
	h.requested.Store(false)
	rt.runFrame()
	countMenus := func(root *node) int { return 0 }
	countMenus = func(root *node) int {
		if root == nil {
			return 0
		}
		n := 0
		if root.role == RoleMenu || root.role == RoleMenuItem {
			n++
		}
		for child := root.first; child != nil; child = child.next {
			n += countMenus(child)
		}
		return n
	}
	p := rt.menu.overlay
	t.Logf("after requested frame: overlay=%v menuNodes=%d requested=%v pointerIn=%v", p != nil, countMenus(rt.c.root), h.requested.Load(), rt.pointerIn)
	if p == nil {
		t.Fatal("menu was closed rather than waiting for presentation")
	}
	if !h.requested.Load() && countMenus(rt.c.root) == 0 {
		t.Error("native menu is open but no menu scene was built and no presentation frame was requested")
	}
	if !p.leaveAt.IsZero() || p.pointerInside {
		t.Fatal("inactive pointer armed pointer-leave closure")
	}
	// Consume the requested presentation frame only. No Frame()/settle(),
	// pointer nudge or unrelated animation may fill in the missing wakeup.
	if !h.requested.Swap(false) {
		t.Fatal("no host request permits the overlay presentation frame")
	}
	rt.runFrame()
	t.Logf("after host-requested presentation frame: sameOverlay=%v menuNodes=%d pointerIn=%v", rt.menu.overlay == p, countMenus(rt.c.root), rt.pointerIn)
	if rt.menu.overlay != p || countMenus(rt.c.root) < 3 {
		t.Fatal("host-requested presentation frame did not build the retained menu")
	}
	var pin *node
	var findPin func(*node)
	findPin = func(root *node) {
		if root == nil {
			return
		}
		if root.kind == kindText && root.text == "Pin" {
			pin = root
		}
		for child := root.first; child != nil; child = child.next {
			findPin(child)
		}
	}
	findPin(rt.c.root)
	if pin == nil || pin.Bounds().W <= 0 || pin.Bounds().H <= 0 {
		t.Fatal("menu text was not laid out in the presented scene")
	}
	icon := pin.parent.first
	if icon.kind != kindIcon {
		t.Fatal("menu lost its visible icon")
	}
	box := icon.Bounds()
	pixel := h.image().RGBAAt(int(box.X+box.W/2), int(box.Y+box.H/2))
	if pixel.A == 0 || pixel.R > 70 || pixel.G > 70 || pixel.B > 70 {
		t.Fatalf("host-requested scene did not actually paint its menu icon: %v", pixel)
	}
}
