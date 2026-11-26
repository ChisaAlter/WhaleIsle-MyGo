package design

import (
	"math"
	"testing"
	"time"
)

func TestDesignMotionFiveTierTokensAndCurve(t *testing.T) {
	if MotionFast != 100*time.Millisecond ||
		MotionPopover != 160*time.Millisecond ||
		MotionDefault != 200*time.Millisecond ||
		MotionSlow != 300*time.Millisecond ||
		MotionFlip != 400*time.Millisecond {
		t.Fatalf("unexpected 5-tier motion tokens: %v %v %v %v %v", MotionFast, MotionPopover, MotionDefault, MotionSlow, MotionFlip)
	}
	if CurveEaseInOut != (CubicBezier{0.4, 0, 0.2, 1}) {
		t.Fatalf("CurveEaseInOut = %+v, want {0.4, 0, 0.2, 1}", CurveEaseInOut)
	}
	if EaseInOut(0) != 0 || EaseInOut(1) != 1 || EaseInOut(0.5) <= 0.5 {
		t.Fatalf("unexpected EaseInOut samples: 0=%v 0.5=%v 1=%v", EaseInOut(0), EaseInOut(0.5), EaseInOut(1))
	}
}

func TestButtonMarqueeDurationAndKeyframes(t *testing.T) {
	if got := ButtonMarqueeDuration(10); got != 3600*time.Millisecond {
		t.Fatalf("short overflow duration = %v, want 3600ms", got)
	}
	if got := ButtonMarqueeDuration(48); got != 5600*time.Millisecond {
		t.Fatalf("48 DIP overflow duration = %v, want 5600ms", got)
	}
	period := ButtonMarqueeDuration(24) // 3600ms
	cases := []struct {
		phase float64
		want  float32
	}{
		{0.00, 0},
		{0.15, 0},
		{0.30, -12},
		{0.45, -24},
		{0.60, -24},
		{0.75, -12},
		{0.90, 0},
		{1.00, 0},
	}
	for _, tc := range cases {
		elapsed := time.Duration(tc.phase * float64(period))
		got := ButtonMarqueeOffset(elapsed, 24, false)
		if math.Abs(float64(got-tc.want)) > 0.1 {
			t.Fatalf("phase %.2f offset = %v, want %v", tc.phase, got, tc.want)
		}
	}
	if got := ButtonMarqueeOffset(period/2, 24, true); got != 0 {
		t.Fatalf("reduce-motion marquee offset = %v, want 0", got)
	}
}

func TestDurationSkeletonMatchesMotionSpec(t *testing.T) {
	if DurationSkeleton != 2600*time.Millisecond {
		t.Fatalf("DurationSkeleton = %v, want 2600ms", DurationSkeleton)
	}
}
