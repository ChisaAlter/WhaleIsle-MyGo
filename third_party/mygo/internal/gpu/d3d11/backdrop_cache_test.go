//go:build windows

package d3d11

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// Backdrops are computed again only where what is under them changed, and
// the frames drawn with those kept show what the CPU renderer draws.
func TestBackdropCache(t *testing.T) {
	if backdropCacheOff {
		t.Skip("MYGO_BACKDROP_CACHE=off")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.Scene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	frame := func(s *scene.Scene) (hits, misses int) {
		t.Helper()
		r.bdHits, r.bdMisses = 0, 0
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		pix, stride := r.readBack(t)
		gputest.Compare(t, "d3d11", pix, stride, s)
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
		return r.bdHits, r.bdMisses
	}
	if hits, misses := frame(s); hits != 0 || misses == 0 {
		t.Fatalf("first frame: %d kept, %d computed", hits, misses)
	}
	n := len(r.b.Backdrops)
	// The same scene (its images and atlases keep their IDs).
	if hits, misses := frame(s); hits != n || misses != 0 {
		t.Fatalf("the same frame again: %d kept, %d computed of %d", hits, misses, n)
	}
	// A fill under the first backdrop: it alone is computed again.
	first := r.b.Backdrops[0].Area
	s = gputest.Scene()
	at := 0
	for i, op := range s.Ops {
		if op.Kind == scene.OpEffect {
			at = i
			break
		}
	}
	under := scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(first.Min.X + 4), Y: float32(first.Min.Y + 4), W: 6, H: 6}, Color: scene.Color{R: 200, A: 255}}
	s.Ops = append(s.Ops[:at], append([]scene.Op{under}, s.Ops[at:]...)...)
	if hits, misses := frame(s); misses == 0 || hits == 0 {
		t.Fatalf("a change under the first backdrop: %d kept, %d computed", hits, misses)
	}
	// Another clear color computes them all again.
	s = gputest.Scene()
	s.Clear = scene.Color{R: 10, G: 20, B: 30, A: 255}
	if hits, _ := frame(s); hits != 0 {
		t.Fatalf("another clear color kept %d backdrops", hits)
	}
}
