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
		{"swap", SwapMotion, 100 * time.Millisecond},
		{"flip", FlipMotion, 400 * time.Millisecond},
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

// TestMotionTokensMatchTheReferenceSheet pins every duration to the reference
// token sheet (reference/electron/vendor/deepseek-harness/packages/client/
// ui-theme/src/styles/base.css:13-20), which is the only native source that
// states a duration as a number:
//
//	--ds-transition-duration       0.2s
//	--ds-transition-duration-fast  0.1s
//	--ds-transition-duration-slow  0.3s   (no widget token uses it)
//	--ds-motion-duration-overlay        var(--ds-transition-duration)      0.2s
//	--ds-motion-duration-overlay-out    var(--ds-transition-duration-fast) 0.1s
//	--ds-motion-duration-popover        0.16s
//	--ds-motion-duration-swap           var(--ds-transition-duration-fast) 0.1s
//	--ds-motion-duration-flip           0.4s
//
// Before this lock swap was 120ms (+20) and flip 300ms (-100): the sheet's swap
// token reuses the fast duration rather than inventing one, and flip is the
// longest token in the sheet.
func TestMotionTokensMatchTheReferenceSheet(t *testing.T) {
	const second = int64(time.Second)
	sheet := []struct {
		token string
		got   time.Duration
		want  time.Duration
	}{
		{"--ds-transition-duration", OverlayDuration, 200 * time.Millisecond},
		{"--ds-motion-duration-overlay", OverlayMotion.Duration, 200 * time.Millisecond},
		{"--ds-transition-duration-fast", FastDuration, 100 * time.Millisecond},
		{"--ds-motion-duration-swap", SwapDuration, 100 * time.Millisecond},
		{"--ds-motion-duration-popover", PopoverDuration, 160 * time.Millisecond},
		{"--ds-motion-duration-flip", FlipDuration, 400 * time.Millisecond},
	}
	for _, row := range sheet {
		if row.got != row.want {
			t.Errorf("%s = %dms, want the sheet's %dms (delta %dms)", row.token, row.got.Milliseconds(), row.want.Milliseconds(), (row.got - row.want).Milliseconds())
		}
		if row.got <= 0 || int64(row.got)%(second/1000) != 0 {
			t.Errorf("%s = %v is not a whole-millisecond duration", row.token, row.got)
		}
	}
	// The sheet derives swap and overlay-out from the fast token; the package
	// must derive them the same way instead of repeating the literal.
	if SwapDuration != FastDuration {
		t.Errorf("swap = %v, want the fast token %v", SwapDuration, FastDuration)
	}
	if HoverMotion.Duration != FastDuration {
		t.Errorf("hover motion = %v, want the fast token %v", HoverMotion.Duration, FastDuration)
	}
	if HoverMotion.Ease == nil || HoverMotion.Ease(0.5) != DesignEase(0.5) {
		t.Error("hover motion does not ride the design curve")
	}
	// Flip is the longest token; nothing in the sheet outlasts it.
	for _, d := range []time.Duration{OverlayDuration, PopoverDuration, SwapDuration, FastDuration} {
		if d > FlipDuration {
			t.Errorf("token %v outlasts the sheet's longest token %v", d, FlipDuration)
		}
	}
}
