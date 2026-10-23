package ui

import (
	"math"
	"testing"
	"time"
)

func TestListAnimatedNaturalFitDoesNotCompleteBeforeShownLayout(t *testing.T) {
	for _, start := range []float32{0, 100} {
		var list ListState
		height := float32(260)
		tt, now := clockTester(func(c *context) {
			coreBox(c).Width(300).Height(height).Transition(ElementTransition{Size: true, Duration: time.Second, Ease: Linear}).Children(func() { coreList(c, &list, 40, func(i int) { coreBox(c).Height(60) }).Fill() })
		}, 300, 2500)
		list.ScrollToOffset(start)
		tt.Frame()
		height = 2400
		list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(d float32) time.Duration { return time.Duration(d) * time.Millisecond }})
		tt.rt.runFrame()
		if !list.ScrollAnimating() || list.ScrollOffset() != start {
			t.Fatalf("natural fit prematurely completed: start%v offset%v moving%v", start, list.ScrollOffset(), list.ScrollAnimating())
		}
		*now = now.Add(100 * time.Millisecond)
		tt.rt.runFrame()
		if !list.ScrollAnimating() {
			t.Fatal("provisional completion discarded motion next frame")
		}
	}
}

func TestListAnimatedInitialDurationUsesDisplayedTransitionSize(t *testing.T) {
	var list ListState
	height := float32(260)
	var distance float32
	tt, _ := clockTester(func(c *context) {
		coreBox(c).Width(300).Height(height).Transition(ElementTransition{Size: true, Duration: time.Second, Ease: Linear}).Children(func() { coreList(c, &list, 40, func(i int) { coreBox(c).Height(60) }).Fill() })
	}, 300, 500)
	list.ScrollToOffset(100)
	tt.Frame()
	height = 400
	list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(d float32) time.Duration { distance = d; return time.Second }})
	tt.rt.runFrame()
	if distance != 1300 || list.ScrollOffset() != 100 {
		t.Fatalf("initial motion used natural size: distance%v offset%v", distance, list.ScrollOffset())
	}
}

func TestListAnimatedFinalCenterUsesActualTransitionSize(t *testing.T) {
	var list ListState
	height := float32(260)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Width(300).Height(height).Transition(ElementTransition{Size: true, Duration: time.Second, Ease: Linear}).Children(func() { coreList(c, &list, 40, func(i int) { coreBox(c).Height(60) }).Fill() })
	}, 300, 500)
	list.ScrollToOffset(100)
	tt.Frame()
	list.ScrollToAnimated(25, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	height = 400
	*now = now.Add(time.Second)
	tt.rt.runFrame()
	r, ok := rowBox(tt, &list, 25)
	if !ok || math.Abs(float64(r.Y-100)) > .01 || list.ScrollAnimating() {
		t.Fatalf("final alignment used natural instead of displayed viewport: %+v offset%v", r, list.ScrollOffset())
	}
}
