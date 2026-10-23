package ui

import (
	"math"
	"testing"
	"time"
)

func TestListAnimatedScrollMeasuresWithoutJumpAndStopsBeforeWheel(t *testing.T) {
	var list ListState
	var scroll ScrollState
	var distance float32
	tt := coreNewTester(func(c *context) {
		coreList(c, &list, 40, func(i int) { coreBox(c).Height(60) }).Fill().TrackScroll(&scroll)
	}, 300, 260)
	now := time.Unix(9000, 0)
	tt.SetTime(now)
	list.ScrollToOffset(100)
	tt.Frame()
	list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(d float32) time.Duration { distance = d; return time.Second }})
	tt.Frame()
	if scroll.Y != 100 || distance != 1300 || !list.ScrollAnimating() {
		t.Fatalf("initial layout jumped: %+v distance%v", scroll, distance)
	}
	tt.SetTime(now.Add(500 * time.Millisecond))
	if scroll.Y != 750 || list.ScrollOffset() != 750 {
		t.Fatalf("halfway:%+v", scroll)
	}
	list.StopScrollAnimation()
	tt.Scroll(200, 100, 0, 40)
	if scroll.Y != 790 {
		t.Fatalf("cancel swallowed wheel:%+v", scroll)
	}
	tt.SetTime(now.Add(2 * time.Second))
	if scroll.Y != 790 || list.ScrollAnimating() {
		t.Fatalf("cancel resumed:%+v", scroll)
	}
}

func TestListAnimatedScrollFollowsTargetKeyAndMeasuresFinalCenter(t *testing.T) {
	ids := make([]int, 50)
	for i := range ids {
		ids[i] = i
	}
	list := ListState{Key: func(i int) any { return ids[i] }}
	var scroll ScrollState
	tt := coreNewTester(func(c *context) {
		coreList(c, &list, len(ids), func(i int) {
			h := float32(60)
			if ids[i] == 25 {
				h = 180
			}
			coreBox(c).Height(h)
		}).Fill().TrackScroll(&scroll)
	}, 300, 260)
	now := time.Unix(9000, 0)
	tt.SetTime(now)
	list.ScrollToOffset(100)
	tt.Frame()
	list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	if scroll.Y != 100 {
		t.Fatalf("target measurement jumped:%+v", scroll)
	}
	tt.SetTime(now.Add(200 * time.Millisecond))
	ids = append([]int{100, 101, 102}, ids...)
	tt.Frame()
	tt.SetTime(now.Add(time.Second))
	r, ok := rowBox(tt, &list, 28)
	if !ok || math.Abs(float64(r.Y-40)) > .01 || r.H != 180 || list.ScrollAnimating() {
		t.Fatalf("keyed tall target not centered: %+v ok%v scroll%+v", r, ok, scroll)
	}
	checkRows(t, tt, &list, 0, 0, 260)
}

func TestListAnimatedScrollReducedMotionReplacementAndRemoval(t *testing.T) {
	n := 40
	list := ListState{FollowEnd: true, Key: func(i int) any { return i }}
	var scroll ScrollState
	tt := coreNewTester(func(c *context) {
		coreList(c, &list, n, func(i int) { coreBox(c).Height(60) }).Fill().TrackScroll(&scroll)
	}, 300, 260)
	now := time.Unix(9000, 0)
	tt.SetTime(now)
	spec := ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }}
	list.ScrollToAnimated(2, Center, spec)
	tt.Frame()
	tt.SetTime(now.Add(200 * time.Millisecond))
	list.ScrollTo(10, Center)
	tt.Frame()
	if list.ScrollAnimating() || scroll.Y != 500 {
		t.Fatalf("immediate replacement:%+v", scroll)
	}
	list.ScrollToAnimated(30, Center, spec)
	tt.Frame()
	n = 20
	tt.Frame()
	if list.ScrollAnimating() {
		t.Fatal("removed target retained animation")
	}
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	list.ScrollToAnimated(5, Center, spec)
	tt.Frame()
	if scroll.Y != 200 || list.ScrollAnimating() {
		t.Fatalf("reduced did not center immediately:%+v", scroll)
	}
	if !list.FollowEnd {
		t.Fatal("animation changed configured follow policy")
	}
}
