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
		{"popover", PopoverMotion, 150 * time.Millisecond},
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
	// The easings run from 0 to 1, and start and end slowly.
	for _, m := range []Motion{OverlayMotion, PopoverMotion, SwapMotion, FlipMotion} {
		if got := m.Ease(0); got != 0 {
			t.Errorf("easing at 0 is %v", got)
		}
		if got := m.Ease(1); got != 1 {
			t.Errorf("easing at 1 is %v", got)
		}
		if got := m.Ease(0.5); got != 0.5 {
			t.Errorf("easing at 0.5 is %v, want the middle", got)
		}
	}
}
