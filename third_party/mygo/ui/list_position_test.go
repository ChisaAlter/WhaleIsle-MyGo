package ui

import (
	"math"
	"testing"
)

func TestListPositionPreservesKeyedRowAcrossPrependAndResize(t *testing.T) {
	ids := make([]int, 80)
	for i := range ids {
		ids[i] = i
	}
	var state ListState
	var scroll ScrollState
	state.Key = func(i int) any { return ids[i] }
	tt := coreNewTester(func(c *context) {
		coreList(c, &state, len(ids), func(i int) { coreBox(c).Height(varied(ids[i])) }).Grow(1).TrackScroll(&scroll)
	}, 300, 300)
	state.ScrollToPosition(20, -9.5)
	tt.Frame()
	before, _ := rowBox(tt, &state, 20)
	if math.Abs(float64(before.Y+9.5)) > .01 {
		t.Fatalf("partial position=%+v", before)
	}
	ids = append([]int{1000, 1001, 1002, 1003}, ids...)
	state.ScrollToPosition(24, before.Y)
	tt.Frame()
	if after, _ := rowBox(tt, &state, 24); after.Y != before.Y {
		t.Fatalf("prepend position=%+v want Y %v", after, before.Y)
	}
	// The target is measured again after a width change clears cached heights.
	tt.SetSize(420, 360)
	state.ScrollToPosition(24, 38)
	tt.Frame()
	if after, _ := rowBox(tt, &state, 24); after.Y != 38 {
		t.Fatalf("resized position=%+v", after)
	}
	state.ScrollTo(24, Start)
	tt.Frame()
	if after, _ := rowBox(tt, &state, 24); after.Y != 0 {
		t.Fatalf("ordinary ScrollTo changed=%+v", after)
	}
	state.ScrollToPosition(0, 100)
	tt.Frame()
	if after, _ := rowBox(tt, &state, 0); after.Y != 0 || scroll.Y != 0 {
		t.Fatalf("start not clamped row=%+v scroll=%+v", after, scroll)
	}
	state.ScrollToPosition(len(ids)-1, -100)
	tt.Frame()
	if scroll.Y != scroll.MaxY {
		t.Fatalf("end not clamped=%+v", scroll)
	}
}

func TestListPositionRejectsNonfiniteAndWorksWithoutKeys(t *testing.T) {
	var state ListState
	var scroll ScrollState
	tt := coreNewTester(func(c *context) {
		coreList(c, &state, 100, func(int) { coreBox(c).Height(30) }).Grow(1).TrackScroll(&scroll)
	}, 300, 300)
	state.ScrollToPosition(30, 42)
	state.ScrollToPosition(0, float32(math.NaN()))
	state.ScrollToPosition(0, float32(math.Inf(1)))
	tt.Frame()
	if box, _ := rowBox(tt, &state, 30); box.Y != 42 {
		t.Fatalf("invalid positions replaced good request=%+v", box)
	}
}
