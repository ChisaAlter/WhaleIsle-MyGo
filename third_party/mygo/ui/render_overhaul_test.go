package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestPointerMoveSameElementKeepsRedrawAndMemo verifies that moving the pointer
// within the same element (no hover transition, no TrackPointer, no drag) does
// not break paint-only redraw or bust Memo regions.
func TestPointerMoveSameElementKeepsRedrawAndMemo(t *testing.T) {
	memoBuilds := 0
	tt := newRepaintTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreBox(c).Size(100, 40).Draw(func(p *Painter, r Rect) {
				p.AnimationFrame()
				p.Fill(r, RGB(10, 20, 30), 0)
			})
			coreColumn(c).Size(100, 40).Memo(1).Children(func() {
				memoBuilds++
				coreText(c, "Static memo")
			})
		})
	})

	// First move enters the static memo box (hover transitions from nil -> box).
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 20, Y: 50})
	tt.frame(16 * time.Millisecond)
	memoBuildsAfterEnter := memoBuilds

	// Subsequent moves inside the exact same element must NOT break rt.redraw
	// and must NOT bust the memo region.
	built, more := tt.frame(16 * time.Millisecond)
	if built || !more {
		t.Fatalf("steady repaint before move: built=%v more=%v", built, more)
	}
	for i := range 5 {
		tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(21 + i), Y: 52})
		if !tt.rt.redraw {
			t.Fatalf("step %d: same-element PointerMove broke rt.redraw", i)
		}
		built, more = tt.frame(16 * time.Millisecond)
		if built {
			t.Fatalf("step %d: same-element PointerMove triggered full view rebuild", i)
		}
		if !more {
			t.Fatalf("step %d: animation stopped unexpectedly", i)
		}
	}
	if memoBuilds != memoBuildsAfterEnter {
		t.Fatalf("same-element PointerMove busted Memo: builds went %d -> %d", memoBuildsAfterEnter, memoBuilds)
	}
}

// TestFocusedTypingDoesNotBustUnrelatedMemos verifies that typing into a
// focused input does not bust unrelated sibling Memo regions, even when those
// sibling regions hold shortcuts, FocusGroups, or resting Transitions.
func TestFocusedTypingDoesNotBustUnrelatedMemos(t *testing.T) {
	textVal := ""
	sidebarBuilds, chatBuilds := 0, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreColumn(c).Height(60).Memo(1).Children(func() {
				sidebarBuilds++
				b := coreButton(c, "Sidebar Action").Transition(ElementTransition{Colors: true})
				b.Shortcut(Cmd, KeyK)
			})
			coreColumn(c).Height(60).FocusGroup(Vertical).Memo(1).Children(func() {
				chatBuilds++
				coreText(c, "Chat history")
			})
			coreColumn(c).Height(40).Memo(1).Children(func() {
				coreTextInput(c, &textVal)
			})
		})
	}, 300, 200)
	tt.Frame()
	tt.ClickAt(20, 130)
	tt.Frame()
	sidebarBefore, chatBefore := sidebarBuilds, chatBuilds

	for _, ch := range "hello" {
		tt.Type(string(ch))
		tt.Frame()
	}
	if textVal != "hello" {
		t.Fatalf("typed text %q, want %q", textVal, "hello")
	}
	if sidebarBuilds != sidebarBefore {
		t.Fatalf("unrelated sidebar Memo with shortcut+Transition rebuilt %d times while typing", sidebarBuilds-sidebarBefore)
	}
	if chatBuilds != chatBefore {
		t.Fatalf("unrelated chat Memo with FocusGroup rebuilt %d times while typing", chatBuilds-chatBefore)
	}
}

// TestCaretBlinkUsesPaintOnlyRefresh verifies that cursor blinking in both
// TextInput and TextArea schedules Painter.After (paint-only repaintFrame)
// instead of rebuilding the view tree every 530ms.
func TestCaretBlinkUsesPaintOnlyRefresh(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		name := "TextInput"
		if multiline {
			name = "TextArea"
		}
		t.Run(name, func(t *testing.T) {
			val := "hello"
			tt := newRepaintTester(func(c *context) {
				if multiline {
					coreTextArea(c, &val).Size(180, 80)
				} else {
					coreTextInput(c, &val).Size(180, 32)
				}
			})
			// Focus the editor.
			tt.Press(20, 16)
			tt.Release(20, 16)
			tt.frame(0)

			if !tt.rt.redraw || tt.rt.repaintDue.IsZero() {
				t.Fatalf("expected paint-only blink timer (redraw=%v repaintDue=%v animating=%v)",
					tt.rt.redraw, tt.rt.repaintDue, tt.rt.animating)
			}
			if tt.rt.animating {
				t.Fatal("caret blink must not set rt.animating")
			}

			// Advancing past multiple 530ms blink intervals must repaint without rebuilding.
			for i := range 3 {
				built, _ := tt.frame(530 * time.Millisecond)
				if built {
					t.Fatalf("blink %d rebuilt the view tree instead of paint-only repaint", i)
				}
				if !tt.rt.redraw || tt.rt.repaintDue.IsZero() {
					t.Fatalf("blink %d lost paint-only schedule", i)
				}
			}
		})
	}
}

// TestColorAndOpacityTransitionsUsePaintOnly verifies that pure color and pure
// opacity ElementTransitions advance through repaintFrame with zero view builds.
func TestColorAndOpacityTransitionsUsePaintOnly(t *testing.T) {
	t.Run("color", func(t *testing.T) {
		hot := false
		tt := newRepaintTester(func(c *context) {
			bg := RGB(20, 20, 20)
			if hot {
				bg = RGB(220, 40, 40)
			}
			coreBox(c).Size(60, 40).Background(bg).Transition(ElementTransition{
				Duration: 200 * time.Millisecond,
				Ease:     Linear,
				Colors:   true,
			})
		})
		hot = true
		tt.rt.changed()
		built, more := tt.frame(0)
		if !built || !more {
			t.Fatalf("trigger frame: built=%v more=%v; want true, true", built, more)
		}
		if tt.rt.animating || !tt.rt.redraw {
			t.Fatalf("color transition should be paint-only: animating=%v redraw=%v", tt.rt.animating, tt.rt.redraw)
		}

		// Midpoint must advance color without rebuilding the view.
		built, more = tt.frame(100 * time.Millisecond)
		if built || !more {
			t.Fatalf("midpoint: built=%v more=%v; want false, true", built, more)
		}
		mid := tt.Image().RGBAAt(10, 10)
		if mid.R < 100 || mid.R > 140 {
			t.Fatalf("midpoint color not interpolated: %+v", mid)
		}

		// Completion finishes at target color without rebuilding.
		built, more = tt.frame(150 * time.Millisecond)
		if built || more {
			t.Fatalf("completion: built=%v more=%v; want false, false", built, more)
		}
		end := tt.Image().RGBAAt(10, 10)
		if end.R != 220 {
			t.Fatalf("final color R=%d, want 220", end.R)
		}
	})

	t.Run("opacity-enter", func(t *testing.T) {
		show := false
		tt := newRepaintTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				if show {
					coreBox(c).Key("fade").Size(60, 40).Background(RGB(200, 40, 40)).Transition(ElementTransition{
						Duration: 200 * time.Millisecond,
						Ease:     Linear,
						Enter:    &Motion{Opacity: 0},
					})
				}
			})
		})
		show = true
		tt.rt.changed()
		built, more := tt.frame(0)
		if !built || !more {
			t.Fatalf("enter frame: built=%v more=%v; want true, true", built, more)
		}
		if tt.rt.animating || !tt.rt.redraw {
			t.Fatalf("pure opacity enter should be paint-only: animating=%v redraw=%v", tt.rt.animating, tt.rt.redraw)
		}
		built, more = tt.frame(100 * time.Millisecond)
		if built || !more {
			t.Fatalf("opacity midpoint: built=%v more=%v; want false, true", built, more)
		}
		built, more = tt.frame(150 * time.Millisecond)
		if built || more {
			t.Fatalf("opacity end: built=%v more=%v; want false, false", built, more)
		}
	})
}

// TestSmoothWheelScrollFastPathSkipsViewRebuild verifies that discrete smooth
// wheel scroll animation steps shift and repaint the existing scroll tree
// without rebuilding the outer view on every frame.
func TestSmoothWheelScrollFastPathSkipsViewRebuild(t *testing.T) {
	tt := newRepaintTester(func(c *context) {
		coreScroll(c).Size(200, 100).Children(func() {
			for i := range 30 {
				coreBox(c).Height(24).Shrink(0).Children(func() {
					coreText(c, fmt.Sprintf("Line %d", i))
				})
			}
		})
	})
	tt.rt.smoothWheelInTest = true
	// Establish pointer inside the scroll container before scrolling.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 50, Y: 50})
	tt.frame(16 * time.Millisecond)

	beforeRow, ok := tt.Find("Line 5")
	if !ok {
		t.Fatal("Line 5 not found")
	}

	// Send one discrete wheel tick (Precise: false).
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 50, Y: 50, DY: 99, Precise: false})

	steps := 0
	rebuilds := 0
	for range 40 {
		built, more := tt.frame(16 * time.Millisecond)
		steps++
		if built {
			rebuilds++
		}
		if !more {
			break
		}
	}
	if steps < 4 {
		t.Fatalf("expected multi-frame smooth scroll animation, got %d steps", steps)
	}
	if rebuilds != 0 {
		t.Fatalf("smooth wheel scroll rebuilt the view %d times across %d steps; want 0", rebuilds, steps)
	}
	afterRow, ok := tt.Find("Line 5")
	if !ok || afterRow.Y >= beforeRow.Y-50 {
		t.Fatalf("Line 5 did not scroll smoothly: before=%v after=%v", beforeRow, afterRow)
	}
}

// TestSmoothWheelScrollListFastPath verifies that discrete smooth wheel
// scrolling on a virtual List skips outer view rebuilds for steps within
// already-built overscan rows and safely falls back when new rows enter.
func TestSmoothWheelScrollListFastPath(t *testing.T) {
	var listState ListState
	tt := newRepaintTester(func(c *context) {
		coreList(c, &listState, 200, func(i int) {
			coreBox(c).Height(24).Children(func() {
				coreText(c, fmt.Sprintf("Item %d", i))
			})
		}).Size(200, 120)
	})
	tt.rt.smoothWheelInTest = true
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 50, Y: 50})
	tt.frame(16 * time.Millisecond)

	// Scroll by 120 DIPs: spans multiple smooth steps, most within overscan.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 50, Y: 50, DY: 120, Precise: false})
	steps := 0
	rebuilds := 0
	for range 40 {
		built, more := tt.frame(16 * time.Millisecond)
		steps++
		if built {
			rebuilds++
		}
		if !more {
			break
		}
	}
	if steps < 4 {
		t.Fatalf("expected multi-frame smooth list scroll, got %d steps", steps)
	}
	if rebuilds >= steps/2 {
		t.Fatalf("virtual List smooth scroll rebuilt too often: %d rebuilds in %d steps", rebuilds, steps)
	}
	if !tt.HasText("Item 7") {
		t.Fatalf("virtual List did not reveal newly scrolled rows: %q", tt.Texts())
	}
}
