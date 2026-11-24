package gpu_test

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

func keysOf(s *scene.Scene) []uint64 {
	var b gpu.Builder
	b.Build(s, func(*scene.Image) uintptr { return 1 })
	return b.BackdropKeys(nil)
}

// A backdrop's key follows what is drawn under its area, and only that.
func TestBackdropKeys(t *testing.T) {
	s := gputest.Scene()
	base := keysOf(s)
	if len(base) < 2 {
		t.Fatalf("%d backdrops in the test scene", len(base))
	}
	if again := keysOf(gputest.Scene()); !equal(again, base) {
		t.Fatalf("the same scene keys %v, then %v", base, again)
	}
	// A fill under the first backdrop, which the effects at the bottom do
	// not reach.
	var b gpu.Builder
	b.Build(s, func(*scene.Image) uintptr { return 1 })
	first := b.Backdrops[0].Area
	s = gputest.Scene()
	at := len(s.Ops)
	for i, op := range s.Ops {
		if op.Kind == scene.OpEffect {
			at = i
			break
		}
	}
	under := scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(first.Min.X + 2), Y: float32(first.Min.Y + 2), W: 4, H: 4}, Color: scene.Color{R: 9, A: 255}}
	s.Ops = append(s.Ops[:at], append([]scene.Op{under}, s.Ops[at:]...)...)
	changed := keysOf(s)
	if changed[0] == base[0] {
		t.Error("a fill under the first backdrop left its key")
	}
	// One far from every backdrop, at the top right of the frame.
	s = gputest.Scene()
	far := scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(s.Width - 3), Y: 0, W: 2, H: 2}, Color: scene.Color{R: 9, A: 255}}
	s.Ops = append([]scene.Op{far}, s.Ops...)
	for i, k := range keysOf(s) {
		var b gpu.Builder
		b.Build(s, func(*scene.Image) uintptr { return 1 })
		if a := b.Backdrops[i].Area; a.Max.X >= s.Width-4 && a.Min.Y <= 2 {
			continue // it reaches the corner
		}
		if k != base[i] {
			t.Errorf("a fill far from backdrop %d changed its key", i)
		}
	}
}

func equal(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
