package pluginui

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func requireMirrorRect(t *testing.T, got, want MirrorRect) {
	t.Helper()
	for name, pair := range map[string][2]float64{"x": {got.X, want.X}, "y": {got.Y, want.Y}, "w": {got.W, want.W}, "h": {got.H, want.H}} {
		if math.Abs(pair[0]-pair[1]) > 1e-8*math.Max(1, math.Abs(pair[1])) {
			t.Errorf("%s = %.12g, want %.12g", name, pair[0], pair[1])
		}
	}
}

func TestMirrorFragmentTallRowAndNegativeTop(t *testing.T) {
	viewport := MirrorRect{W: 640, H: 900}
	for _, top := range []float64{0, -1150, -2750} {
		full := MirrorRect{X: 40, Y: top, W: 500, H: 3200}
		fragment, ok := NewMirrorFragment(full, viewport, viewport)
		if !ok {
			t.Fatalf("top %v: no visible fragment", top)
		}
		want := MirrorRect{X: 40, Y: 0, W: 500, H: math.Min(900, top+3200)}
		requireMirrorRect(t, fragment.Full, full)
		requireMirrorRect(t, fragment.Viewport, viewport)
		requireMirrorRect(t, fragment.Visible, want)
		dst := ui.Rect{X: 100, Y: 20, W: 250, H: 1600}
		mapped, ok := MapMirrorFragment(dst, ui.Rect{W: 1000, H: 2000}, fragment)
		if !ok {
			t.Fatalf("top %v: mapping failed", top)
		}
		if mapped.Destination != dst || mapped.Scale != .5 {
			t.Fatalf("top %v: full row transform changed: %+v", top, mapped)
		}
		requireMirrorRect(t, mirrorFragmentFromNative(mapped.Clip), MirrorRect{X: 100, Y: 20 - top*.5, W: 250, H: want.H * .5})
		requireMirrorRect(t, mapped.SourceRect, want)
	}
}

func TestMirrorFragmentAncestorAndNativeClipAtUniformScale(t *testing.T) {
	full := MirrorRect{X: 100, Y: -400, W: 400, H: 3200}
	viewport := MirrorRect{X: 20, Y: 0, W: 600, H: 900}
	ancestor := MirrorRect{X: 160, Y: 80, W: 300, H: 520}
	fragment, ok := NewMirrorFragment(full, viewport, ancestor)
	if !ok {
		t.Fatal("ancestor fragment missing")
	}
	requireMirrorRect(t, fragment.Visible, ancestor)
	// A cropped fragment is narrower than the full row. The full width still
	// supplies scale2; the fragment must not be stretched back to row width.
	dst := ui.Rect{X: 30, Y: -100, W: 800, H: 6400}
	nativeClip := ui.Rect{X: 210, Y: 1000, W: 300, H: 500}
	mapped, ok := MapMirrorFragment(dst, nativeClip, fragment)
	if !ok {
		t.Fatal("mapping missing")
	}
	if mapped.Scale != 2 {
		t.Fatalf("scale = %v", mapped.Scale)
	}
	requireMirrorRect(t, mirrorFragmentFromNative(mapped.Clip), MirrorRect{X: 210, Y: 1000, W: 300, H: 500})
	requireMirrorRect(t, mapped.SourceRect, MirrorRect{X: 190, Y: 150, W: 150, H: 250})
	for _, point := range [][2]float64{{210, 1000}, {509.5, 1499.5}, {360, 1250}} {
		x := full.X + (point[0]-float64(dst.X))/mapped.Scale
		y := full.Y + (point[1]-float64(dst.Y))/mapped.Scale
		if x < mapped.SourceRect.X || x >= mapped.SourceRect.X+mapped.SourceRect.W || y < mapped.SourceRect.Y || y >= mapped.SourceRect.Y+mapped.SourceRect.H {
			t.Fatalf("native hit %v inverse (%v,%v) escaped source %+v", point, x, y, mapped.SourceRect)
		}
	}
	// The same source section is also valid in a narrower native row.
	narrow, ok := MapMirrorFragment(ui.Rect{X: 5, Y: 10, W: 200, H: 1600}, ui.Rect{W: 1000, H: 3000}, fragment)
	if !ok || narrow.Scale != .5 {
		t.Fatalf("narrow mapping: %+v %v", narrow, ok)
	}
	requireMirrorRect(t, mirrorFragmentFromNative(narrow.Clip), MirrorRect{X: 35, Y: 250, W: 150, H: 260})
}

func TestMirrorFragmentRejectsInvalidAndDisjointGeometry(t *testing.T) {
	valid := MirrorRect{W: 400, H: 900}
	for _, bad := range []MirrorRect{{}, {W: -1, H: 5}, {W: 1, H: -1}, {X: math.NaN(), W: 1, H: 1}, {Y: math.Inf(-1), W: 1, H: 1}, {W: math.Inf(1), H: 1}, {X: math.MaxFloat64, W: math.MaxFloat64, H: 1}, {X: 1e100, W: 1, H: 1}} {
		for position := range 3 {
			args := [3]MirrorRect{valid, valid, valid}
			args[position] = bad
			if _, ok := NewMirrorFragment(args[0], args[1], args[2]); ok {
				t.Errorf("accepted invalid position%d rect %+v", position, bad)
			}
		}
	}
	for _, disjoint := range []MirrorRect{{X: 400, W: 1, H: 1}, {Y: 900, W: 1, H: 1}, {X: -3, W: 2, H: 1}} {
		if _, ok := NewMirrorFragment(valid, valid, disjoint); ok {
			t.Errorf("accepted disjoint ancestor %+v", disjoint)
		}
	}
	fragment, _ := NewMirrorFragment(valid, valid, valid)
	for _, dst := range []ui.Rect{{}, {W: 1, H: -1}, {W: 400, H: 450}, {X: float32(math.Inf(1)), W: 400, H: 900}, {W: float32(math.NaN()), H: 900}, {X: math.MaxFloat32, W: math.MaxFloat32, H: math.MaxFloat32}} {
		if _, ok := MapMirrorFragment(dst, ui.Rect{W: 1000, H: 2000}, fragment); ok {
			t.Errorf("accepted invalid/nonuniform native row %+v", dst)
		}
	}
	if _, ok := MapMirrorFragment(ui.Rect{W: 400, H: 900}, ui.Rect{X: 400, W: 1, H: 1}, fragment); ok {
		t.Error("accepted native clip touching only an edge")
	}
	if _, ok := MapMirrorFragment(ui.Rect{W: 400, H: 900}, ui.Rect{}, fragment); ok {
		t.Error("accepted empty native clip")
	}
	fragment.Visible.X = math.NaN()
	if _, ok := MapMirrorFragment(ui.Rect{W: 400, H: 900}, ui.Rect{W: 1000, H: 2000}, fragment); ok {
		t.Error("accepted forged invalid visible area")
	}
}

func TestMirrorFragmentFloat32RoundingAndOverflow(t *testing.T) {
	full := MirrorRect{X: .2, Y: -.3, W: 333.3, H: 3200.1}
	viewport := MirrorRect{W: 900, H: 900}
	fragment, _ := NewMirrorFragment(full, viewport, viewport)
	dst := ui.Rect{X: 13.1, Y: 25.2, W: 201.7, H: float32(3200.1 * float64(float32(201.7)) / 333.3)}
	mapped, ok := MapMirrorFragment(dst, ui.Rect{W: 1000, H: 3000}, fragment)
	if !ok {
		t.Fatal("rejected uniformly rounded float32 layout")
	}
	if mapped.SourceRect.Y < 0 || mapped.SourceRect.Y+mapped.SourceRect.H > 900.0000001 {
		t.Fatalf("rounding escaped viewport: %+v", mapped)
	}
	dst.H *= 1.001
	if _, ok := MapMirrorFragment(dst, ui.Rect{W: 1000, H: 3000}, fragment); ok {
		t.Error("accepted height stretched beyond relative tolerance")
	}
	// Every finite float32 field can still overflow when native edges add.
	large := ui.Rect{X: math.MaxFloat32 / 2, W: math.MaxFloat32, H: 1}
	if _, ok := MapMirrorFragment(large, ui.Rect{W: math.MaxFloat32, H: 1}, fragment); ok {
		t.Error("accepted overflowing float32 native endpoint")
	}
	tiny := MirrorRect{W: math.SmallestNonzeroFloat64, H: 1}
	fragment, _ = NewMirrorFragment(tiny, tiny, tiny)
	if _, ok := MapMirrorFragment(ui.Rect{W: 1, H: 1}, ui.Rect{W: 1, H: 1}, fragment); ok {
		t.Error("accepted overflowing uniform scale")
	}
}

func TestMirrorSourceScrollDeltaPreservesDifferentOffsetsWithinSameTail(t *testing.T) {
	for _, tc := range []struct{ top, offset, target, want float64 }{
		{-2200, 2500, 100, 200}, {-2200, 3000, 100, 700}, {100, 0, 0, 100}, {-100, 50, 0, -50}, {0, 0, 0, 0},
	} {
		got, ok := MirrorSourceScrollDelta(tc.top, tc.offset, tc.target)
		if !ok || got != tc.want {
			t.Errorf("scroll (%v,%v,%v) = %v,%v; want %v", tc.top, tc.offset, tc.target, got, ok, tc.want)
		}
	}
	for _, args := range [][3]float64{{math.NaN(), 0, 0}, {0, math.Inf(1), 0}, {0, 0, math.Inf(-1)}, {0, -1, 0}, {0, 0, -1}, {math.MaxFloat64, math.MaxFloat64, 0}} {
		if _, ok := MirrorSourceScrollDelta(args[0], args[1], args[2]); ok {
			t.Errorf("accepted invalid scroll %v", args)
		}
	}
}
