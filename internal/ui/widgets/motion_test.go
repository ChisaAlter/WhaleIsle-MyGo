package widgets

import (
	"testing"
	"time"
)

func TestMotionTokens(t *testing.T) {
	cases := []struct {
		name string
		m    Motion
		want time.Duration
	}{
		{"overlay", OverlayMotion, 200 * time.Millisecond},
		{"popover", PopoverMotion, 160 * time.Millisecond},
		{"swap", SwapMotion, 120 * time.Millisecond},
		{"flip", FlipMotion, 300 * time.Millisecond},
	}
	for _, c := range cases {
		if c.m.Duration != c.want {
			t.Errorf("%s takes %v, want %v", c.name, c.m.Duration, c.want)
		}
		if c.m.Ease == nil {
			t.Errorf("%s has no easing", c.name)
		}
	}
	if OverlayDuration != OverlayMotion.Duration || PopoverDuration != PopoverMotion.Duration ||
		SwapDuration != SwapMotion.Duration || FlipDuration != FlipMotion.Duration {
		t.Fatal("the duration constants and the motion tokens disagree")
	}
	// Every token rides the design system's shared curve, cubic-bezier(0.4, 0,
	// 0.2, 1): 0 to 1 over the duration, ahead of the midpoint in the middle.
	for _, m := range []Motion{OverlayMotion, PopoverMotion, SwapMotion, FlipMotion} {
		if got := m.Ease(0); got != 0 {
			t.Errorf("easing at 0 is %v", got)
		}
		if got := m.Ease(1); got != 1 {
			t.Errorf("easing at 1 is %v", got)
		}
		for _, at := range []float32{0.25, 0.5} {
			if got, want := m.Ease(at), DesignEase(at); got != want {
				t.Errorf("easing at %v is %v, want the design curve's %v", at, got, want)
			}
		}
		if got := m.Ease(0.5); got <= 0.5 {
			t.Errorf("easing at 0.5 is %v, want the design curve ahead of the middle", got)
		}
	}
	// A token with no curve of its own animates on the design curve too.
	if got := (Motion{Duration: time.Second}).Ease; got != nil {
		t.Fatalf("a bare motion should leave its ease nil, got %v", got)
	}
}
