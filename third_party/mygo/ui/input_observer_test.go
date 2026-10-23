package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestInputObserverIsGlobalPassiveAndPrecedesConsumingWidgets(t *testing.T) {
	var order []string
	var observed []InputEvent
	tt := NewTester(func(c *Context) {
		c.ObserveInput(func(ev InputEvent) { order = append(order, "observer"); observed = append(observed, ev) })
		Box(c).Fill().Children(func() {
			Box(c).Width(100).Height(100).Focusable().HandleInput(func(ev InputEvent) bool {
				if ev.Kind != InputPointerMove {
					order = append(order, "widget")
				}
				return true
			})
		})
	}, 300, 200)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: 40, Y: 30, Button: 1, Clicks: 2})
	if len(order) != 2 || order[0] != "observer" || order[1] != "widget" {
		t.Fatalf("delivery order %v", order)
	}
	if observed[0].X != 40 || observed[0].Y != 30 || observed[0].Button != 1 || observed[0].Clicks != 2 {
		t.Fatal(observed)
	}
	order = nil
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 250, Y: 150, DX: 0, DY: 0, Precise: true, Mods: platform.ModCtrl})
	if len(order) != 1 || observed[1].Kind != InputScroll || !observed[1].Precise || observed[1].Mods != Ctrl {
		t.Fatalf("global exhausted wheel %v %v", order, observed)
	}
	order = nil
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyPageDown, Mods: platform.ModAlt, Repeat: true, ShortcutBlocked: true})
	if len(order) == 0 || order[0] != "observer" || observed[2].Key != KeyPageDown || !observed[2].Repeat || !observed[2].ShortcutBlocked {
		t.Fatal(observed)
	}
}

func TestKeyCapturePrecedesFocusedHandlerAndWindowShortcuts(t *testing.T) {
	show := true
	var order []string
	shortcuts := 0
	tt := NewTester(func(c *Context) {
		if show {
			c.ObserveInput(func(ev InputEvent) {
				if ev.Kind == InputKeyDown {
					order = append(order, "observe")
				}
			})
			c.CaptureKeyDown(func(ev InputEvent) bool { order = append(order, "capture"); return ev.Key == KeyEscape })
		}
		if c.Shortcut(0, KeyEscape) {
			shortcuts++
		}
		Box(c).Fill().Focusable().Focus().HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown {
				order = append(order, "handler")
			}
			return true
		})
	}, 100, 100)
	tt.Key(0, KeyEscape)
	if len(order) != 2 || order[0] != "observe" || order[1] != "capture" || shortcuts != 0 {
		t.Fatalf("captured key escaped: %v shortcuts=%d", order, shortcuts)
	}
	order = nil
	tt.Key(0, KeyA)
	if len(order) != 3 || order[2] != "handler" {
		t.Fatalf("uncaptured key changed: %v", order)
	}
	show = false
	tt.Frame()
	tt.Key(0, KeyEscape)
	if shortcuts != 1 || len(tt.rt.keyCaptures) != 0 {
		t.Fatal("hidden key capture survived")
	}
	show = true
	tt.Frame()
	tt.rt.close()
	if len(tt.rt.keyCaptures) != 0 {
		t.Fatal("closed capture retained")
	}
}

func TestKeyCaptureRetiresPhysicalChordAndUpdatesModifiers(t *testing.T) {
	tt := NewTester(func(c *Context) {
		c.CaptureKeyDown(func(ev InputEvent) bool { return ev.Key == KeyEscape })
		Text(c, "body")
	}, 100, 100)
	tt.rt.physical.held = map[string]bool{"KeyQ": true}
	tt.rt.physical.consumed = map[string]bool{"KeyQ": true}
	if !tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyEscape, Mods: platform.ModAlt}) {
		t.Fatal("capture did not consume Alt+Escape")
	}
	if tt.rt.mods != Alt || len(tt.rt.physical.held) != 0 || len(tt.rt.physical.consumed) != 0 {
		t.Fatal("captured key retained stale modifiers or a physical chord")
	}
}

func TestBlurObserverRetiresPressBeforeSyntheticRelease(t *testing.T) {
	var order []string
	tt := NewTester(func(c *Context) {
		c.ObserveInput(func(ev InputEvent) {
			if ev.Kind == InputBlur {
				order = append(order, "observe-blur")
			}
		})
		Box(c).Fill().Focusable().HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputBlur:
				order = append(order, "blur")
			case InputPointerUp:
				order = append(order, "release")
			}
			return true
		})
	}, 100, 100)
	tt.Press(20, 20)
	tt.SetFocused(false)
	if len(order) != 3 || order[0] != "observe-blur" || order[1] != "blur" || order[2] != "release" {
		t.Fatalf("blur/release order %v", order)
	}
}

func TestKeepFocusMaskDoesNotStealKeyboardFocus(t *testing.T) {
	open := false
	tt := NewTester(func(c *Context) {
		Button(c, "owner").AutoFocus()
		if open {
			Overlay(c, func() {
				Box(c).Absolute().Left(0).Top(50).Size(100, 100).KeepFocus().HandleInput(func(InputEvent) bool { return true })
			})
		}
	}, 200, 200)
	open = true
	tt.Frame()
	tt.ClickAt(40, 80)
	if !tt.Focused("owner") {
		t.Fatal("mask stole owner focus")
	}
}

func TestInputObserverReplacedByLastPassAndReleasedWhenAbsentOrClosed(t *testing.T) {
	show := true
	count := 0
	tt := NewTester(func(c *Context) {
		if show {
			c.ObserveInput(func(InputEvent) { count++ })
		}
		Text(c, "body")
	}, 100, 100)
	for range 5 {
		tt.Frame()
	}
	tt.Key(0, KeyA)
	if count != 1 {
		t.Fatalf("observers accumulated:%d", count)
	}
	show = false
	tt.Frame()
	tt.Press(20, 20)
	if count != 1 || len(tt.rt.inputObservers) != 0 {
		t.Fatal("hidden observer survived")
	}
	show = true
	tt.Frame()
	tt.rt.close()
	if len(tt.rt.inputObservers) != 0 {
		t.Fatal("closed observer retained")
	}
}
