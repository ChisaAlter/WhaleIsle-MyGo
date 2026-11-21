package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// dialogExitTester builds a page with an "Open" button, a "Behind" button
// where the dialog's panel lands, and a DialogBase showing "Panel body"
// with a focusable button in it. The clock is the test's.
func dialogExitTester(options *DialogOptions) (*Tester, *time.Time, *bool, *int) {
	open, behind := false, 0
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			if coreButton(c, "Open").Clicked() {
				open = true
			}
		})
		if b := coreButton(c, "Behind").Absolute().Left(150).Top(120).Right(150).Bottom(120); b.Clicked() {
			behind++
		}
		paint := func(backdrop, panel *node) {
			panel.Children(func() {
				coreText(c, "Panel body")
				coreButton(c, "Inside")
			})
		}
		if options != nil {
			coreDialogBaseOptions(c, &open, *options, paint)
		} else {
			coreDialogBase(c, &open, paint)
		}
	}, 400, 300)
	return tt, now, &open, &behind
}

// exitGhost returns the exit copy still showing under the overlay layer,
// or nil.
func exitGhost(tt *Tester) *node {
	for e := tt.rt.c.root.first; e != nil; e = e.next {
		if e.id != overlayID {
			continue
		}
		for g := e.first; g != nil; g = g.next {
			if g.leaving != 0 {
				return g
			}
		}
	}
	return nil
}

// TestDialogExitFadesAndPassesThrough closes a dialog mid-motion: the
// overlay stays as a fading copy that queries, clicks and focus pass by.
func TestDialogExitFadesAndPassesThrough(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	tt, now, open, behind := dialogExitTester(nil)
	tt.Key(0, KeyTab) // the keyboard focus is the opener's to give back
	if !tt.Focused("Open") {
		t.Fatal("the opener did not take the focus")
	}
	*open = true
	tt.Frame()
	tt.Key(0, KeyTab) // the modal takes the focus in
	if !tt.Focused("Inside") {
		t.Fatal("the dialog did not take the focus")
	}
	tt.Key(0, KeyEscape)
	if *open {
		t.Fatal("Escape did not close the dialog")
	}
	*now = now.Add(40 * time.Millisecond)
	tt.Frame()
	g := exitGhost(tt)
	if g == nil {
		t.Fatal("the closed dialog left no exit ghost")
	}
	if g.opacity <= 0 || g.opacity >= 1 {
		t.Fatalf("the exit ghost is not fading: opacity %v", g.opacity)
	}
	if tt.HasText("Panel body") {
		t.Fatal("queries still find the exit ghost's content")
	}
	if !tt.Focused("Open") {
		t.Fatal("the focus did not return to the opener")
	}
	tt.ClickAt(200, 150)
	if *behind != 1 {
		t.Fatalf("a click under the exit ghost did not reach the page: %d", *behind)
	}
}

// TestDialogExitCompletesAndIdles sees the ghost gone once the exit ran,
// and the runtime asking for no more frames.
func TestDialogExitCompletesAndIdles(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	tt, now, open, _ := dialogExitTester(nil)
	*open = true
	tt.Frame()
	*open = false
	tt.Frame()
	*now = now.Add(150 * time.Millisecond)
	tt.Frame()
	if g := exitGhost(tt); g != nil {
		t.Fatalf("the exit ghost is still there after the exit: opacity %v", g.opacity)
	}
	if tt.h.requested.Load() {
		t.Fatal("the runtime still asks for frames after the exit")
	}
	if tt.HasText("Panel body") {
		t.Fatal("the closed dialog still shows")
	}
}

// TestDialogExitReduceMotionIsImmediate checks that less motion closes a
// dialog at once, exit copy and all.
func TestDialogExitReduceMotionIsImmediate(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	tt, _, open, _ := dialogExitTester(nil)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	*open = true
	tt.Frame()
	*open = false
	tt.Frame()
	if g := exitGhost(tt); g != nil {
		t.Fatal("ReduceMotion kept an exit ghost")
	}
	if tt.HasText("Panel body") {
		t.Fatal("the closed dialog still shows under ReduceMotion")
	}
}

// TestDialogExitNoExitIsImmediate checks that DialogOptions.NoExit removes
// its dialog at once whatever SetDialogExit configured.
func TestDialogExitNoExitIsImmediate(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	tt, _, open, _ := dialogExitTester(&DialogOptions{NoExit: true})
	*open = true
	tt.Frame()
	*open = false
	tt.Frame()
	if g := exitGhost(tt); g != nil {
		t.Fatal("NoExit kept an exit ghost")
	}
	if tt.HasText("Panel body") {
		t.Fatal("the NoExit dialog still shows")
	}
}

// TestDialogExitUnsetIsImmediate keeps the behavior of before: with no
// SetDialogExit a closing dialog is gone at once.
func TestDialogExitUnsetIsImmediate(t *testing.T) {
	tt, _, open, _ := dialogExitTester(nil)
	*open = true
	tt.Frame()
	*open = false
	tt.Frame()
	if g := exitGhost(tt); g != nil {
		t.Fatal("an unset exit kept a ghost")
	}
	if tt.HasText("Panel body") {
		t.Fatal("the closed dialog still shows")
	}
}

// TestDialogExitReopenDropsGhost reopens a dialog while its exit copy still
// fades: the copy goes at once and exactly one panel shows.
func TestDialogExitReopenDropsGhost(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	tt, now, open, _ := dialogExitTester(nil)
	*open = true
	tt.Frame()
	*open = false
	tt.Frame()
	*now = now.Add(40 * time.Millisecond)
	tt.Frame()
	if exitGhost(tt) == nil {
		t.Fatal("no exit ghost to reopen over")
	}
	*open = true
	tt.Frame()
	if g := exitGhost(tt); g != nil {
		t.Fatal("reopening did not drop the exit ghost")
	}
	panels := 0
	for _, s := range tt.Texts() {
		if s == "Panel body" {
			panels++
		}
	}
	if panels != 1 {
		t.Fatalf("reopened dialog shows %d panels", panels)
	}
}

// TestDialogExitGhostPaintsPastExpiredElements guards the crash an
// application met at settings close: the exit ghost repaints the Draw
// and DrawOver closures built a frame before, whose captured elements
// expired with that frame. On a ghost those queries must read absent —
// the copy refers to no other element — so the paint does not panic.
func TestDialogExitGhostPaintsPastExpiredElements(t *testing.T) {
	SetDialogExit(DialogExit{Duration: 100 * time.Millisecond, Ease: Linear})
	t.Cleanup(func() { SetDialogExit(DialogExit{}) })
	open := true
	ghostPainted := 0
	now := time.Unix(1_000_000, 0)
	tt := NewTester(func(c *Context) {
		DialogBase(c, &open, func(backdrop, panel Element) {
			panel.Children(func() {
				el := Box(c).Size(40, 20)
				el.Draw(func(p *Painter, r Rect) {
					if !el.Valid() {
						ghostPainted++
					}
					_ = el.FocusVisible()
					_ = el.Hovered()
					_ = el.Pressed()
					_ = el.Focused()
					_ = el.Bounds()
					p.Fill(r, RGBA(0, 0, 0, .5), 0)
				})
				el.DrawOver(func(p *Painter, r Rect) {
					_ = el.FocusVisible()
					_ = el.Hovered()
				})
			})
		})
	}, 400, 300)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	open = false
	now = now.Add(40 * time.Millisecond)
	tt.Frame()
	g := exitGhost(tt)
	if g == nil {
		t.Fatal("the closed dialog left no exit ghost")
	}
	if !g.ghost {
		t.Fatal("the exit copy is not marked as a ghost")
	}
	if ghostPainted == 0 {
		t.Fatal("the ghost's Draw did not run")
	}
	now = now.Add(150 * time.Millisecond)
	tt.Frame()
	if exitGhost(tt) != nil {
		t.Fatal("the exit ghost is still there after the exit")
	}
	if tt.h.requested.Load() {
		t.Fatal("the runtime still asks for frames after the exit")
	}
}

// TestExpiredElementQueryStillPanicsOnLiveElements keeps the diagnostic
// the ghost suspends: a stale element queried from a live paint callback
// still panics, so the misuse stays loud outside exit painting.
func TestExpiredElementQueryStillPanicsOnLiveElements(t *testing.T) {
	var stale Element
	first := true
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(fmt.Sprint(r), "element from pass") {
			t.Fatalf("expected the expired-element panic, got %v", r)
		}
	}()
	tt := NewTester(func(c *Context) {
		cur := Box(c).Size(10, 10)
		if first {
			stale, first = cur, false
		}
		cur.Draw(func(p *Painter, r Rect) { _ = stale.FocusVisible() })
	}, 100, 100)
	tt.Frame()
}
