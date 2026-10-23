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
