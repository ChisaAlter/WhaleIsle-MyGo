package chrome

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// TestMotionTokensMatchWidgets keeps the chrome overlays in step with the
// widget primitives, which own the motion tokens.
func TestMotionTokensMatchWidgets(t *testing.T) {
	if PopoverDuration != widgets.PopoverDuration {
		t.Fatalf("popover duration %v, want %v", PopoverDuration, widgets.PopoverDuration)
	}
	if OverlayDuration != widgets.OverlayDuration {
		t.Fatalf("overlay duration %v, want %v", OverlayDuration, widgets.OverlayDuration)
	}
	for name, transition := range map[string]ui.ElementTransition{
		"popover": PopoverTransition(),
		"overlay": OverlayTransition(),
	} {
		if transition.Enter == nil || transition.Exit == nil {
			t.Fatalf("%s transition needs an enter and an exit", name)
		}
		if transition.Enter.Opacity != 0 || transition.Exit.Opacity != 0 {
			t.Fatalf("%s should fade in and out", name)
		}
		if transition.Enter.Y <= 0 {
			t.Fatalf("%s should rise into place", name)
		}
		if transition.Ease == nil {
			t.Fatalf("%s needs an easing curve", name)
		}
		if got := transition.Ease(.5); got <= 0 || got >= 1 {
			t.Fatalf("%s easing(0.5) = %v", name, got)
		}
	}
	if PopoverTransition().Duration != PopoverDuration || OverlayTransition().Duration != OverlayDuration {
		t.Fatal("a transition did not use its motion token's duration")
	}
}

// TestPopoverEnterFadesOverItsDuration drives a menu panel through the
// renderer: it appears transparent and settles opaque one popover duration
// later.
func TestPopoverEnterFadesOverItsDuration(t *testing.T) {
	open := false
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			if !open {
				return
			}
			ui.Box(c).Key("popover").Fill().Background(ui.RGB(255, 0, 0)).Transition(PopoverTransition())
		})
	}, 40, 20)
	tt.SetTime(start)
	open = true
	tt.SetTime(start.Add(time.Millisecond))
	faded := tt.Image().RGBAAt(20, 10).R
	tt.SetTime(start.Add(time.Millisecond + PopoverDuration/2))
	middle := tt.Image().RGBAAt(20, 10).R
	tt.SetTime(start.Add(time.Millisecond + PopoverDuration))
	settled := tt.Image().RGBAAt(20, 10).R
	if !(faded < middle && middle < settled) {
		t.Fatalf("popover fade did not progress: %d → %d → %d", faded, middle, settled)
	}
	if settled < 250 {
		t.Fatalf("popover never reached its opaque face: %d", settled)
	}
}

// TestPopoverEnterJumpsWithReduceMotion covers the reduced-motion path: the
// panel is there at once.
func TestPopoverEnterJumpsWithReduceMotion(t *testing.T) {
	open := false
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			if !open {
				return
			}
			ui.Box(c).Key("popover").Fill().Background(ui.RGB(255, 0, 0)).Transition(PopoverTransition())
		})
	}, 40, 20)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTime(start)
	open = true
	tt.SetTime(start.Add(time.Millisecond))
	if got := tt.Image().RGBAAt(20, 10).R; got < 250 {
		t.Fatalf("reduce motion still faded the popover in: %d", got)
	}
}
