package ui

import "testing"

func TestScrollbarsVisibleIsLocalAndHiddenTracksDoNotTakeInput(t *testing.T) {
	visible, override := true, true
	clicks := 0
	var state *state
	tt := coreNewTester(func(c *context) {
		scrollTheme := *c.Theme()
		scrollTheme.ScrollbarAlways = false
		c.SetTheme(&scrollTheme)
		coreRow(c).Children(func() {
			scroll := coreScroll(c).Size(100, 100).Label("owned")
			if override {
				scroll.ScrollbarsVisible(visible)
			}
			scroll.Children(func() {
				if coreButtonBase(c).FillWidth().Height(400).Clicked() {
					clicks++
				}
			})
			state = scroll.st
			coreScroll(c).Size(100, 100).Label("other").Children(func() { coreBox(c).Height(400) })
		})
	}, 220, 120)
	countBars := func() int {
		n := 0
		for _, op := range tt.rt.scene.Ops {
			if op.Color == tt.rt.c.theme.Scrollbar.scene() {
				n++
			}
		}
		return n
	}
	if countBars() != 1 {
		t.Fatalf("forced visible leaked to another scroll or not shown: %d", countBars())
	}
	visible = false
	tt.Frame()
	tt.Move(95, 10)
	if countBars() != 0 {
		t.Fatalf("hover revealed forced hidden scrollbar: %d", countBars())
	}
	before := state.scrollY
	tt.ClickAt(95, 90)
	if state.scrollY != before || clicks != 1 {
		t.Fatalf("hidden track took click: offset=%v clicks=%d", state.scrollY, clicks)
	}
	tt.Scroll(30, 50, 0, 50)
	if state.scrollY <= before {
		t.Fatal("hiding bars disabled wheel scrolling")
	}
	visible = true
	tt.Frame()
	tt.Move(210, 110)
	if countBars() != 1 {
		t.Fatal("forced shown bar disappeared away from pointer")
	}
	override = false
	tt.Frame()
	if countBars() != 0 {
		t.Fatal("omitted override did not restore theme automatic visibility")
	}
}

func TestScrollbarsVisibleOnlyWhenContentOverflows(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		scrollTheme := *c.Theme()
		scrollTheme.ScrollbarAlways = false
		c.SetTheme(&scrollTheme)
		coreScroll(c).Size(100, 100).ScrollbarsVisible(true).Children(func() { coreBox(c).Height(20) })
	}, 120, 120)
	for _, op := range tt.rt.scene.Ops {
		if op.Color == tt.rt.c.theme.Scrollbar.scene() {
			t.Fatal("forced shown invented a scrollbar for fitting content")
		}
	}
}
