package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDesignEaseMatchesTheDesignCurve pins cubic-bezier(0.4, 0, 0.2, 1)
// (ui-theme/src/styles/base.css:12) against values solved independently from
// the curve's parametric form.
func TestDesignEaseMatchesTheDesignCurve(t *testing.T) {
	for _, tc := range []struct{ t, want float32 }{
		{0, 0},
		{0.1, 0.0258630},
		{0.2, 0.1339468},
		{0.25, 0.2365874},
		{0.35, 0.5},
		{0.5, 0.7755613},
		{0.75, 0.9593677},
		{0.9, 0.9943539},
		{1, 1},
	} {
		if got := DesignEase(tc.t); abs32(got-tc.want) > 1e-4 {
			t.Errorf("DesignEase(%v) = %v, want %v", tc.t, got, tc.want)
		}
	}
	if DesignEase(0) != 0 || DesignEase(1) != 1 {
		t.Fatalf("ends: %v..%v, want exactly 0..1", DesignEase(0), DesignEase(1))
	}
	// The curve is not the framework's cubic in-out, which reaches 0.0625 at a
	// quarter of the time; the design curve is already at ~0.24.
	if got := ui.EaseInOut(0.25); abs32(got-0.0625) > 1e-6 {
		t.Fatalf("EaseInOut(0.25) = %v, the comparison baseline moved", got)
	}
	if DesignEase(0.25) < 0.2 {
		t.Fatalf("DesignEase(0.25) = %v, the cubic in-out was kept", DesignEase(0.25))
	}
}

// TestDesignEaseIsMonotonicAndClamped walks the curve: it never goes back, and
// a time fraction outside the duration (or not a number) stays on an end.
func TestDesignEaseIsMonotonicAndClamped(t *testing.T) {
	previous := float32(-1)
	for i := 0; i <= 1000; i++ {
		at := float32(i) / 1000
		got := DesignEase(at)
		if got < previous {
			t.Fatalf("DesignEase(%v) = %v went back from %v", at, got, previous)
		}
		if got < 0 || got > 1 {
			t.Fatalf("DesignEase(%v) = %v left 0..1", at, got)
		}
		previous = got
	}
	if got := DesignEase(-0.5); got != 0 {
		t.Fatalf("DesignEase(-0.5) = %v, want 0", got)
	}
	if got := DesignEase(1.5); got != 1 {
		t.Fatalf("DesignEase(1.5) = %v, want 1", got)
	}
	nan := float32(0)
	nan = nan / nan
	if got := DesignEase(nan); got != 0 {
		t.Fatalf("DesignEase(NaN) = %v, want 0", got)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
