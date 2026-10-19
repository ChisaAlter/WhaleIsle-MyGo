package ui

import (
	"math"
	"testing"
)

func assertSplitFraction(t *testing.T, got, want float32) {
	t.Helper()
	if math.Abs(float64(got-want)) > .00001 {
		t.Fatalf("fraction=%v want=%v", got, want)
	}
}

func TestSplitFractionPersistentPointerKeyboardBoundsAndResize(t *testing.T) {
	fraction := float32(.5)
	tt := NewTester(func(c *Context) {
		SplitFraction(c, &fraction, .2, func() { Box(c).Label("fraction-first").Fill() }, func() { Box(c).Label("fraction-second").Fill() }).Fill()
	}, 401, 200)
	assertPosition := func(want float32) {
		t.Helper()
		r, ok := tt.Find("fraction-second")
		if !ok || math.Abs(float64(r.X-want)) > .1 {
			t.Fatalf("second pane=%+v found=%v wantX=%v", r, ok, want)
		}
	}
	assertPosition(201)
	// Actual pointer input updates the caller's persistent fraction after input,
	// rather than a pixel-sized local that disappears when build returns.
	tt.Press(200.5, 100)
	tt.Move(500, 100)
	tt.Release(500, 100)
	assertSplitFraction(t, fraction, .8)
	assertPosition(321)
	tt.Key(0, KeyRight)
	assertSplitFraction(t, fraction, .8)
	for range 40 {
		tt.Key(0, KeyLeft)
	}
	assertSplitFraction(t, fraction, .2)
	assertPosition(81)
	tt.Key(0, KeyLeft)
	assertSplitFraction(t, fraction, .2)
	tt.Key(0, KeyRight)
	assertSplitFraction(t, fraction, .225)
	assertPosition(91)
	tt.SetSize(1001, 200)
	assertSplitFraction(t, fraction, .225)
	assertPosition(226)
	// The handle follows proportional allocation immediately after resize.
	tt.Press(225.5, 100)
	tt.Move(-100, 100)
	tt.Release(-100, 100)
	assertSplitFraction(t, fraction, .2)
	assertPosition(201)
}

func TestSplitVerticalFractionBindsLiveKeyboardAndPointerLimits(t *testing.T) {
	fraction := float32(.5)
	tt := NewTester(func(c *Context) {
		SplitVerticalFraction(c, &fraction, .2, func() { Box(c).Label("fraction-top").Fill() }, func() { Box(c).Label("fraction-bottom").Fill() }).Fill()
	}, 200, 401)
	tt.Press(100, 200.5)
	tt.Move(100, -100)
	tt.Release(100, -100)
	assertSplitFraction(t, fraction, .2)
	tt.Key(0, KeyDown)
	assertSplitFraction(t, fraction, .225)
	for range 40 {
		tt.Key(0, KeyDown)
	}
	assertSplitFraction(t, fraction, .8)
	if r, ok := tt.Find("fraction-bottom"); !ok || math.Abs(float64(r.Y-321)) > .1 {
		t.Fatalf("bottom=%+v found=%v", r, ok)
	}
	tt.SetSize(200, 1001)
	if r, ok := tt.Find("fraction-bottom"); !ok || math.Abs(float64(r.Y-801)) > .1 {
		t.Fatalf("resized bottom=%+v found=%v", r, ok)
	}
}
