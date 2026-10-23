package ui

import (
	"math"
	"testing"
	"time"
)

func TestListAnimatedNewOwnerOffsetTakesPriority(t *testing.T) {
	var list ListState
	scroll := &ScrollState{}
	key := "old"
	tt, now := clockTester(func(c *context) {
		List(makeContext(c).Key(key), &list, 40).TrackScroll(scroll).Rows(func(r ListRow) { Box(r.Context).Height(60) }).Width(300).Height(260)
	}, 300, 260)
	list.ScrollToOffset(100)
	tt.Frame()
	list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	*now = now.Add(200 * time.Millisecond)
	tt.Frame()
	key = "new"
	scroll = &ScrollState{Y: 200}
	tt.Frame()
	if list.ScrollAnimating() || list.ScrollOffset() != 200 {
		t.Fatalf("remounted external offset lost: %v", list.ScrollOffset())
	}
}

func TestListAnimatedTargetBuilderCanReplacePendingMovement(t *testing.T) {
	var list ListState
	replace := false
	tt, now := clockTester(func(c *context) {
		coreList(c, &list, 100, func(i int) {
			if i == 80 && replace {
				replace = false
				list.ScrollToOffset(200)
			}
			coreBox(c).Height(60)
		}).Width(300).Height(260)
	}, 300, 260)
	list.ScrollToOffset(100)
	tt.Frame()
	list.ScrollToAnimated(80, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	replace = true
	*now = now.Add(time.Millisecond)
	tt.rt.runFrame()
	if list.ScrollOffset() != 200 || list.ScrollAnimating() {
		t.Fatalf("late target builder replacement lost: %v moving%v", list.ScrollOffset(), list.ScrollAnimating())
	}
}

func TestListAnimatedCoveredRowBuilderCanReplaceOrStopMovement(t *testing.T) {
	for _, stop := range []bool{false, true} {
		var list ListState
		replace := false
		tt, now := clockTester(func(c *context) {
			coreList(c, &list, 100, func(i int) {
				if i == 41 && replace {
					replace = false
					if stop {
						list.StopScrollAnimation()
					} else {
						list.ScrollToOffset(200)
					}
				}
				coreBox(c).Height(60)
			}).Width(300).Height(260)
		}, 300, 260)
		for i := range 100 {
			list.heights.set(i, 60)
		}
		list.ScrollToOffset(100)
		tt.Frame()
		list.ScrollToAnimated(80, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
		tt.Frame()
		replace = true
		*now = now.Add(500 * time.Millisecond)
		tt.rt.runFrame()
		want := float32(200)
		if stop {
			want = 100
		}
		if list.ScrollOffset() != want || list.ScrollAnimating() {
			t.Fatalf("covered builder request lost: stop%v offset%v want%v", stop, list.ScrollOffset(), want)
		}
	}
}

func TestListAnimatedLateMeasuredTallRowDoesNotReverseAtSameTime(t *testing.T) {
	var list ListState
	tt, now := clockTester(func(c *context) {
		coreList(c, &list, 100, func(i int) {
			h := float32(60)
			if i == 38 {
				h = 600
			}
			coreBox(c).Height(h)
		}).Width(300).Height(260)
	}, 300, 260)
	// A previously visited transcript has known ordinary rows and one new tall
	// row just outside the current virtual window; overscan discovers it later.
	for i := range 100 {
		if i != 38 {
			list.heights.set(i, 60)
		}
	}
	list.ScrollToAnimated(80, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	*now = now.Add(500 * time.Millisecond)
	tt.rt.runFrame()
	first := list.ScrollOffset()
	tt.rt.runFrame()
	second := list.ScrollOffset()
	if math.Abs(float64(first-second)) > .01 {
		t.Fatalf("measurement caused reverse motion with unchanged clock: %v -> %v", first, second)
	}
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	r, ok := rowBox(tt, &list, 80)
	if !ok || r.Y != 100 {
		t.Fatalf("final measured target:%+v", r)
	}
}
