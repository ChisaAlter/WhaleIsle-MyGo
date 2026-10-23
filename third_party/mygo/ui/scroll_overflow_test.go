package ui

import (
	"strings"
	"testing"
)

func TestScrollOverflowXKeepsWrapAndReachesOverflow(t *testing.T) {
	var state ScrollState
	paragraph := strings.Repeat("ordinary wrapped words ", 12)
	build := func(c *Context) {
		Scroll(c).ScrollOverflowX().Width(200).Height(160).Padding(10).TrackScroll(&state).Children(func() {
			Column(c).FillWidth().MinWidth(0).Children(func() {
				Text(c, paragraph).Label("paragraph").FontSize(12).FixedLineHeight(18).MinWidth(0)
				Row(c).FillWidth().MinWidth(0).AlignItems(Start).Children(func() {
					Box(c).Width(310).Height(20).Shrink(0).Label("wide item")
					Text(c, "end").Label("overflow end").Shrink(0)
				})
			})
		})
	}
	tt := NewTester(build, 300, 350)
	defer tt.rt.close()
	text, ok := tt.Find("paragraph")
	if !ok || text.W != 180 || text.H <= 18 {
		t.Fatalf("paragraph lost viewport wrapping: %+v found=%v", text, ok)
	}
	if state.MaxX <= 100 || state.MaxY <= 0 {
		t.Fatalf("descendant overflow missing: %+v", state)
	}
	before, _ := tt.Find("overflow end")
	state.X = state.MaxX
	state.Y = state.MaxY
	tt.Frame()
	after, _ := tt.Find("overflow end")
	if after.X >= before.X || after.X+after.W > 200 || after.Y+after.H > 160 {
		t.Fatalf("overflow cannot be reached: before=%+v after=%+v scroll=%+v", before, after, state)
	}
	state.X, state.Y = 0, 0
	tt.Frame()
	tt.SetSize(400, 350)
	// The viewport keeps its explicit width across window resize.
	text, _ = tt.Find("paragraph")
	if text.W != 180 {
		t.Fatalf("viewport constraint lost: %+v", text)
	}
}

func TestScrollOverflowXStopsAtNestedClipAndPreservesScrollBoth(t *testing.T) {
	var clipped, canvas ScrollState
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Scroll(c).ScrollOverflowX().Width(200).Height(80).TrackScroll(&clipped).Children(func() {
				Box(c).FillWidth().ClipX().Children(func() { Box(c).Width(500).Height(20) })
			})
			ScrollBoth(c).Width(200).Height(80).TrackScroll(&canvas).Children(func() {
				Text(c, strings.Repeat("wide ", 80)).Label("canvas").FontSize(12)
			})
		})
	}, 300, 300)
	defer tt.rt.close()
	if clipped.MaxX != 0 {
		t.Fatalf("nested clip leaked overflow: %+v", clipped)
	}
	r, _ := tt.Find("canvas")
	if canvas.MaxX <= 0 || r.H > 20 {
		t.Fatalf("ScrollBoth changed its unbounded-width behavior: %+v %+v", canvas, r)
	}
}
