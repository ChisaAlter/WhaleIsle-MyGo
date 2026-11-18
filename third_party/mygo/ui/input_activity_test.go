package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
	"time"
)

func TestLastInputTimeExcludesFramesAndResize(t *testing.T) {
	var input time.Time
	tt := NewTester(func(c *Context) { input = c.LastInputTime(); Box(c).Fill() }, 120, 80)
	now := time.Unix(1000, 0)
	tt.SetTime(now)
	if !input.IsZero() {
		t.Fatal("frame counted as input")
	}
	for _, kind := range []platform.SurfaceEventKind{platform.PointerMove, platform.PointerDown, platform.PointerScroll, platform.KeyPressed, platform.TextInput, platform.TextComposition, platform.SurfaceFocus} {
		now = now.Add(time.Second)
		tt.SetTime(now)
		tt.rt.event(platform.SurfaceEvent{Kind: kind, X: 30, Y: 20})
		tt.Frame()
		if !input.Equal(now) {
			t.Fatalf("event %v did not record input: %v", kind, input)
		}
		tt.SetTime(now.Add(time.Second / 2))
		tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceResize})
		tt.Frame()
		if !input.Equal(now) {
			t.Fatalf("background frame/resize advanced input: %v", input)
		}
		if kind == platform.PointerMove {
			tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 30, Y: 20})
			tt.Frame()
			if !input.Equal(now) {
				t.Fatal("stationary pointer notification renewed user activity")
			}
		}
	}
}
