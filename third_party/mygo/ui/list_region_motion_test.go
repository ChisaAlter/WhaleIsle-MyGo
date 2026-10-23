package ui

import (
	"math"
	"testing"
	"time"
)

func TestListRegionAnimatedCentersTallOwnerChildrenWithoutEndpointSnap(t *testing.T) {
	for _, childTop := range []float32{60, 880} {
		t.Run(map[float32]string{60: "near-top", 880: "near-bottom"}[childTop], func(t *testing.T) {
			var list ListState
			var scroll ScrollState
			tt := NewTester(func(c *Context) {
				List(c, &list, 40, func(i int) {
					if i != 20 {
						Box(c).Height(60)
						return
					}
					Box(c).Height(1000).Children(func() {
						Box(c).Height(childTop).Shrink(0)
						Box(c).Height(40).Shrink(0).Label("child tool")
						Box(c).Height(1000 - childTop - 40).Shrink(0)
					})
				}).Fill().TrackScroll(&scroll)
			}, 300, 260)
			now := time.Unix(9000, 0)
			tt.SetTime(now)
			// Ordinary rows were visited earlier; only the tall owner is cold.
			for i := range 40 {
				if i != 20 {
					list.heights.set(i, 60)
				}
			}
			list.ScrollToOffset(100)
			tt.Frame()
			if _, ok := rowBox(tt, &list, 20); ok {
				t.Fatal("target owner was already in the virtual window")
			}
			var distance float32
			list.ScrollToRegionAnimated(20, childTop, 40, Center, ScrollAnimation{
				Duration: func(d float32) time.Duration { distance = d; return time.Second },
				Ease:     func(p float32) float32 { return p * p },
			})
			tt.Frame()
			goal := 20*float32(60) + childTop - 110
			if scroll.Y != 100 || distance != goal-100 || !list.ScrollAnimating() {
				t.Fatalf("measurement jumped or used the whole owner: offset%v distance%v goal%v moving%v", scroll.Y, distance, goal, list.ScrollAnimating())
			}
			tt.SetTime(now.Add(500 * time.Millisecond))
			if want := 100 + (goal-100)*.25; scroll.Y != want || list.ScrollOffset() != want || !list.ScrollAnimating() {
				t.Fatalf("intermediate easing: offset%v want%v moving%v", scroll.Y, want, list.ScrollAnimating())
			}
			tt.SetTime(now.Add(time.Second))
			child, ok := tt.Find("child tool")
			if !ok || math.Abs(float64(child.Y-110)) > .01 || child.H != 40 || list.ScrollAnimating() {
				t.Fatalf("child did not stay centered at completion: %+v found%v offset%v moving%v", child, ok, scroll.Y, list.ScrollAnimating())
			}
			tt.Frame()
			if scroll.Y != goal {
				t.Fatalf("completed region snapped to the whole owner: offset%v want%v", scroll.Y, goal)
			}
			checkRows(t, tt, &list, 0, 0, 260)

			list.ScrollToAnimated(20, Center, ScrollAnimation{})
			tt.Frame()
			owner, ok := rowBox(tt, &list, 20)
			if !ok || owner.Y != -370 || owner.H != 1000 {
				t.Fatalf("whole-item scrolling changed after a region request: %+v found%v", owner, ok)
			}
		})
	}
}

func TestListRegionAnimatedColdFirstFrameMeasuresBeforeMoving(t *testing.T) {
	var list ListState
	show := false
	tt, now := clockTester(func(c *context) {
		if show {
			coreList(c, &list, 40, func(i int) {
				h := float32(60)
				if i == 25 {
					h = 1000
				}
				coreBox(c).Height(h)
			}).Width(300).Height(260)
		}
	}, 300, 260)
	var measured float64
	list.ScrollToRegionAnimated(25, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration {
		measured = list.heights.height(25)
		return time.Second
	}})
	show = true
	tt.rt.runFrame()
	if measured != 1000 || list.ScrollOffset() != 0 || !list.ScrollAnimating() {
		t.Fatalf("cold initial frame: measured%v offset%v moving%v", measured, list.ScrollOffset(), list.ScrollAnimating())
	}
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	if list.ScrollOffset() <= 0 || !list.ScrollAnimating() {
		t.Fatalf("cold request did not advance: offset%v moving%v", list.ScrollOffset(), list.ScrollAnimating())
	}
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	r, ok := rowBox(tt, &list, 25)
	if !ok || math.Abs(float64(r.Y+880-110)) > .01 || list.ScrollAnimating() {
		t.Fatalf("cold measured endpoint lost the region: %+v found%v moving%v", r, ok, list.ScrollAnimating())
	}
}

func TestListRegionAnimatedImmediateAndReducedMotionUseRegion(t *testing.T) {
	for _, mode := range []string{"nil-duration", "zero-duration", "negative-duration", "reduced-motion"} {
		t.Run(mode, func(t *testing.T) {
			var list ListState
			show := false
			spec := ScrollAnimation{}
			switch mode {
			case "zero-duration":
				spec.Duration = func(float32) time.Duration { return 0 }
			case "negative-duration":
				spec.Duration = func(float32) time.Duration { return -time.Second }
			case "reduced-motion":
				spec.Duration = func(float32) time.Duration { return time.Second }
			}
			tt := coreNewTester(func(c *context) {
				if show {
					coreList(c, &list, 40, func(i int) {
						h := float32(60)
						if i == 25 {
							h = 1000
						}
						coreBox(c).Height(h)
					}).Fill()
				}
			}, 300, 260)
			if mode == "reduced-motion" {
				tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
			}
			list.ScrollToRegionAnimated(25, 880, 40, Center, spec)
			show = true
			tt.Frame()
			r, ok := rowBox(tt, &list, 25)
			if !ok || math.Abs(float64(r.Y+880-110)) > .01 || list.ScrollAnimating() {
				t.Fatalf("immediate region alignment: %+v found%v moving%v", r, ok, list.ScrollAnimating())
			}
		})
	}
}

func TestListRegionAnimatedTargetKeySurvivesPrepend(t *testing.T) {
	for _, beforeLayout := range []bool{true, false} {
		name := "during-motion"
		if beforeLayout {
			name = "before-first-layout"
		}
		t.Run(name, func(t *testing.T) {
			ids := make([]int, 40)
			for i := range ids {
				ids[i] = i
			}
			list := ListState{Key: func(i int) any { return ids[i] }}
			tt, now := clockTester(func(c *context) {
				coreList(c, &list, len(ids), func(i int) {
					h := float32(60)
					if ids[i] == 20 {
						h = 1000
					}
					coreBox(c).Height(h)
				}).Fill()
			}, 300, 260)
			for i := range ids {
				if i != 20 {
					list.heights.set(i, 60)
				}
			}
			list.ScrollToOffset(100)
			tt.Frame()
			list.ScrollToRegionAnimated(20, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
			if !beforeLayout {
				tt.Frame()
				*now = now.Add(250 * time.Millisecond)
				tt.Frame()
			}
			ids = append([]int{100, 101, 102}, ids...)
			tt.Frame()
			if beforeLayout {
				r, ok := rowBox(tt, &list, 4)
				if !ok || r.Y != -40 {
					t.Fatalf("prepend moved the starting visual anchor: %+v found%v", r, ok)
				}
			}
			*now = now.Add(time.Second)
			tt.Frame()
			r, ok := rowBox(tt, &list, 23)
			if !ok || r.H != 1000 || math.Abs(float64(r.Y+880-110)) > .01 || list.ScrollAnimating() {
				t.Fatalf("region did not follow its owner key: %+v found%v moving%v", r, ok, list.ScrollAnimating())
			}
			checkRows(t, tt, &list, 0, 0, 260)
		})
	}
}

func TestListRegionAnimatedRemovedKeyStopsMotion(t *testing.T) {
	ids := make([]int, 40)
	for i := range ids {
		ids[i] = i
	}
	list := ListState{Key: func(i int) any { return ids[i] }}
	tt, now := clockTester(func(c *context) {
		coreList(c, &list, len(ids), func(i int) {
			h := float32(60)
			if ids[i] == 20 {
				h = 1000
			}
			coreBox(c).Height(h)
		}).Fill()
	}, 300, 260)
	list.ScrollToRegionAnimated(20, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	*now = now.Add(200 * time.Millisecond)
	tt.Frame()
	// Keep the row count unchanged: the old index still exists, but its key is gone.
	ids[20] = 100
	tt.Frame()
	if list.ScrollAnimating() {
		t.Fatal("removed owner key retained its region motion")
	}
	stopped := list.ScrollOffset()
	*now = now.Add(time.Second)
	tt.Frame()
	if list.ScrollOffset() != stopped {
		t.Fatalf("removed region resumed: %v -> %v", stopped, list.ScrollOffset())
	}
}

func TestListRegionAnimatedManualInputAndReplacementCancel(t *testing.T) {
	for _, mode := range []string{"wheel", "stop-before-wheel", "external-offset", "whole-row-request"} {
		t.Run(mode, func(t *testing.T) {
			var list ListState
			var scroll ScrollState
			tt, now := clockTester(func(c *context) {
				coreList(c, &list, 40, func(i int) {
					h := float32(60)
					if i == 20 {
						h = 1000
					}
					coreBox(c).Height(h)
				}).Fill().TrackScroll(&scroll)
			}, 300, 260)
			for i := range 40 {
				if i != 20 {
					list.heights.set(i, 60)
				}
			}
			list.ScrollToRegionAnimated(20, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
			tt.Frame()
			*now = now.Add(500 * time.Millisecond)
			tt.Frame()
			want := scroll.Y + 40
			switch mode {
			case "wheel":
				tt.Scroll(150, 100, 0, 40)
			case "stop-before-wheel":
				list.StopScrollAnimation()
				tt.Scroll(150, 100, 0, 40)
			case "external-offset":
				scroll.Y = want
				tt.Frame()
			case "whole-row-request":
				list.ScrollTo(10, Center)
				want = 500
				tt.Frame()
			}
			if scroll.Y != want || list.ScrollAnimating() {
				t.Fatalf("interrupting position lost: offset%v want%v moving%v", scroll.Y, want, list.ScrollAnimating())
			}
			*now = now.Add(2 * time.Second)
			tt.Frame()
			if scroll.Y != want || list.ScrollAnimating() {
				t.Fatalf("canceled region resumed: offset%v want%v moving%v", scroll.Y, want, list.ScrollAnimating())
			}
		})
	}
}

func TestListRegionAnimatedClampsToMeasuredOwner(t *testing.T) {
	cases := []struct {
		name                        string
		top, height, wantTop, wantH float32
	}{
		{"negative-top", -20, 40, 0, 40},
		{"negative-height", 880, -40, 880, 0},
		{"crosses-bottom", 980, 200, 980, 20},
		{"past-bottom-point", 1200, 80, 1000, 0},
		{"height-exceeds-owner", 0, 2000, 0, 1000},
		{"zero-height-point", 350, 0, 350, 0},
		{"negative-point", -20, -40, 0, 0},
	}
	for _, tc := range cases {
		for _, align := range []Align{Start, Center, End} {
			name := tc.name + "/" + map[Align]string{Start: "start", Center: "center", End: "end"}[align]
			t.Run(name, func(t *testing.T) {
				var list ListState
				tt := coreNewTester(func(c *context) {
					coreList(c, &list, 40, func(i int) {
						h := float32(60)
						if i == 20 {
							h = 1000
						}
						coreBox(c).Height(h)
					}).Fill()
				}, 300, 260)
				list.ScrollToRegionAnimated(20, tc.top, tc.height, align, ScrollAnimation{})
				tt.Frame()
				y := float32(0)
				switch align {
				case Center:
					y = (260 - tc.wantH) / 2
				case End:
					y = 260 - tc.wantH
				}
				r, ok := rowBox(tt, &list, 20)
				if !ok || r.H != 1000 || math.Abs(float64(r.Y+tc.wantTop-y)) > .01 || list.ScrollAnimating() {
					t.Fatalf("measured clamp: owner%+v found%v region%v/%v wantY%v moving%v", r, ok, tc.wantTop, tc.wantH, y, list.ScrollAnimating())
				}
			})
		}
	}
}

func TestListRegionAnimatedClampsToContentBounds(t *testing.T) {
	for _, tc := range []struct {
		name        string
		top, height float32
		align       Align
		want        float32
	}{
		{"start-edge", -10, 40, Center, 0},
		{"end-edge-point", 1200, 80, Center, 740},
		{"whole-owner", 0, 2000, Center, 370},
		{"point-at-end", 1000, 0, Start, 740},
		{"point-at-start", 0, 0, End, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var list ListState
			tt := coreNewTester(func(c *context) {
				coreList(c, &list, 1, func(int) { coreBox(c).Height(1000) }).Fill()
			}, 300, 260)
			list.ScrollToRegionAnimated(0, tc.top, tc.height, tc.align, ScrollAnimation{})
			tt.Frame()
			if list.ScrollOffset() != tc.want || list.ScrollAnimating() {
				t.Fatalf("content bound: offset%v want%v moving%v", list.ScrollOffset(), tc.want, list.ScrollAnimating())
			}
		})
	}
}

func TestListRegionAnimatedReclampsAfterOwnerShrinks(t *testing.T) {
	var list ListState
	height := float32(1000)
	tt, now := clockTester(func(c *context) {
		coreList(c, &list, 40, func(i int) {
			h := float32(60)
			if i == 20 {
				h = height
			}
			coreBox(c).Height(h)
		}).Fill()
	}, 300, 260)
	list.ScrollToRegionAnimated(20, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	height = 600
	*now = now.Add(time.Second)
	tt.Frame()
	r, ok := rowBox(tt, &list, 20)
	if !ok || r.H != 600 || math.Abs(float64(r.Y+600-130)) > .01 || list.ScrollAnimating() {
		t.Fatalf("shrunk owner did not clamp to its bottom point: %+v found%v moving%v", r, ok, list.ScrollAnimating())
	}
}

func TestListRegionAnimatedFinalCenterUsesDisplayedTransitionSize(t *testing.T) {
	var list ListState
	height := float32(260)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Width(300).Height(height).Transition(ElementTransition{Size: true, Duration: time.Second, Ease: Linear}).Children(func() {
			coreList(c, &list, 40, func(i int) {
				h := float32(60)
				if i == 20 {
					h = 1000
				}
				coreBox(c).Height(h)
			}).Fill()
		})
	}, 300, 500)
	list.ScrollToRegionAnimated(20, 880, 40, Center, ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	height = 400
	*now = now.Add(time.Second)
	tt.rt.runFrame()
	r, ok := rowBox(tt, &list, 20)
	if !ok || math.Abs(float64(r.Y+880-110)) > .01 || list.ScrollAnimating() {
		t.Fatalf("region endpoint used natural rather than displayed viewport: %+v found%v offset%v", r, ok, list.ScrollOffset())
	}
}
