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
		// A surface rides the design system's cubic-bezier(0.4, 0, 0.2, 1), not
		// the framework's cubic in-out (which is far behind early).
		if got, want := transition.Ease(.25), widgets.DesignEase(.25); got != want {
			t.Fatalf("%s easing(0.25) = %v, want the design curve's %v", name, got, want)
		}
	}
	if PopoverTransition().Duration != PopoverDuration || OverlayTransition().Duration != OverlayDuration {
		t.Fatal("a transition did not use its motion token's duration")
	}
}

// TestOverlayRecipeMatchesTheDesignSystem pins the overlay's exact recipe:
// its enter rises 8 DIP from 96% of its size over 200 ms, and it goes on the
// fast 100 ms token (ui-theme/src/styles/base.css:16-17,21,23). A popover keeps
// its 160 ms and 4 DIP and never shrinks.
func TestOverlayRecipeMatchesTheDesignSystem(t *testing.T) {
	overlay := OverlayTransition()
	if overlay.Enter.Y != 8 || overlay.Exit.Y != 8 {
		t.Fatalf("overlay rise = %v in, %v out, want 8", overlay.Enter.Y, overlay.Exit.Y)
	}
	if overlay.Enter.Scale != 0.96 || overlay.Exit.Scale != 0.96 {
		t.Fatalf("overlay scale = %v in, %v out, want 0.96", overlay.Enter.Scale, overlay.Exit.Scale)
	}
	if overlay.Duration != 200*time.Millisecond || overlay.ExitDuration != 100*time.Millisecond {
		t.Fatalf("overlay %v in, %v out, want 200ms in and the 100ms fast token out", overlay.Duration, overlay.ExitDuration)
	}
	popover := PopoverTransition()
	if popover.Enter.Y != 4 || popover.Exit.Y != 4 {
		t.Fatalf("popover rise = %v, want 4", popover.Enter.Y)
	}
	if popover.Enter.Scale > 0 && popover.Enter.Scale != 1 {
		t.Fatalf("popover shrinks to %v, want no scale", popover.Enter.Scale)
	}
	if popover.Duration != 160*time.Millisecond || popover.ExitDuration != 0 {
		t.Fatalf("popover %v with a %v exit, want its own 160ms both ways", popover.Duration, popover.ExitDuration)
	}
}

// TestOverlayTransitionScalesInAndExitsFast drives the recipe through the
// renderer: the panel starts at 96% of its width, settles at its own, and its
// copy is gone 100 ms after it closes rather than 200.
func TestOverlayTransitionScalesInAndExitsFast(t *testing.T) {
	open := false
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			if !open {
				return
			}
			ui.Box(c).Key("overlay-panel").Label("overlay panel").Width(100).Height(60).
				Background(ui.RGB(255, 0, 0)).Transition(OverlayTransition())
		})
	}, 200, 120)
	tt.SetTime(start)
	open = true
	tt.SetTime(start.Add(time.Millisecond))
	panel, ok := tt.Find("overlay panel")
	if !ok {
		t.Fatal("the overlay panel never showed")
	}
	if panel.W > 98 || panel.W < 95 || panel.X < 1 {
		t.Fatalf("the panel starts at %+v, want 96%% of 100 wide, centred", panel)
	}
	tt.SetTime(start.Add(OverlayDuration / 2))
	half, _ := tt.Find("overlay panel")
	if half.W <= panel.W || half.W >= 100 {
		t.Fatalf("halfway the panel is %v wide, want between %v and 100", half.W, panel.W)
	}
	tt.SetTime(start.Add(OverlayDuration + time.Millisecond))
	if settled, _ := tt.Find("overlay panel"); settled.W != 100 || settled.X != 0 {
		t.Fatalf("the panel settled at %+v, want its own box", settled)
	}
	// Closing: the copy is still painted halfway through the 100 ms exit, and
	// gone once that ends, before the 200 ms enter token would have finished.
	open = false
	tt.SetTime(start.Add(OverlayDuration + 2*time.Millisecond))
	tt.Frame()
	x, y := 50, 30
	closing := tt.Image().RGBAAt(x, y).R
	tt.SetTime(start.Add(OverlayDuration + 2*time.Millisecond + OverlayExitDuration/2))
	mid := tt.Image().RGBAAt(x, y).R
	tt.SetTime(start.Add(OverlayDuration + 2*time.Millisecond + OverlayExitDuration))
	if done := tt.Image().RGBAAt(x, y).R; done != 0 {
		t.Fatalf("the copy is still painted %v after a %v exit: %d", OverlayExitDuration, OverlayExitDuration, done)
	}
	if !(closing > mid) {
		t.Fatalf("the exit did not fade: %d → %d", closing, mid)
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

// TestOverlayRevealFadesPanelAndMask checks the hand-run overlay motion: a
// DialogBase panel and its dim mask fade in over the overlay token, because a
// transition's enter can never fire inside an overlay born in the same frame.
func TestOverlayRevealFadesPanelAndMask(t *testing.T) {
	open := true
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0))
		ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
			progress := Reveal(c, backdrop, OverlayDuration)
			RevealMask(backdrop, ui.RGBA(200, 200, 200, .5), progress)
			panel.Key("reveal-dialog").Width(40).Height(20).Radius(8).Opacity(progress).
				Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, ui.RGB(255, 0, 0), 8) })
		})
	}, 60, 40)
	tt.SetTime(start)
	hidden := tt.Image().RGBAAt(30, 20)
	tt.SetTime(start.Add(OverlayDuration / 4))
	quarter := tt.Image().RGBAAt(30, 20)
	tt.SetTime(start.Add(OverlayDuration / 2))
	half := tt.Image().RGBAAt(30, 20)
	tt.SetTime(start.Add(OverlayDuration))
	settled := tt.Image().RGBAAt(30, 20)
	if !(hidden.R < quarter.R && quarter.R < half.R && half.R < settled.R) {
		t.Fatalf("overlay fade did not progress: %d → %d → %d → %d", hidden.R, quarter.R, half.R, settled.R)
	}
	// The design curve is already ~0.24 along a quarter into the fade; the
	// framework's cubic in-out would only be at 0.06 (about 1/4 of this).
	if quarter.R < 40 {
		t.Fatalf("overlay eased in on the cubic in-out, not the design curve: quarter=%d", quarter.R)
	}
	if settled.R < 250 {
		t.Fatalf("dialog never reached full opacity: %d", settled.R)
	}
	// Outside the panel the mask still dims the window, and it is darker at the
	// start than once the panel settled.
	if got := tt.Image().RGBAAt(5, 35).R; got != 100 {
		t.Fatalf("mask at full opacity = %d, want 100 over black", got)
	}
	if hidden.R != 0 {
		t.Fatalf("the mask painted before the overlay's clock started: %d", hidden.R)
	}
}

// TestOverlayRevealJumpsWithReduceMotion covers the reduced-motion path of the
// hand-run overlay motion.
func TestOverlayRevealJumpsWithReduceMotion(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0))
		ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
			progress := Reveal(c, backdrop, OverlayDuration)
			RevealMask(backdrop, ui.RGBA(0, 0, 0, .5), progress)
			panel.Key("reveal-dialog").Width(40).Height(20).Radius(8).Opacity(progress).
				Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, ui.RGB(255, 0, 0), 8) })
		})
	}, 60, 40)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTime(time.Unix(1000, 0))
	if got := tt.Image().RGBAAt(30, 20).R; got < 250 {
		t.Fatalf("reduce motion still faded the dialog in: %d", got)
	}
}

// TestOverlayRevealKeepsControlsClickable pins that a panel faded in by hand is
// still hit-testable on its first frames: Reveal only changes paint opacity, so
// a fast click on a control inside a dialog that is still fading in works.
func TestOverlayRevealKeepsControlsClickable(t *testing.T) {
	open := true
	clicked := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0))
		ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
			progress := Reveal(c, backdrop, OverlayDuration)
			RevealMask(backdrop, ui.RGBA(0, 0, 0, .5), progress)
			panel.Key("reveal-clickable").Width(120).Height(60).Radius(8).Opacity(progress).Children(func() {
				if ui.Button(c, "act").Clicked() {
					clicked = true
				}
			})
		})
	}, 120, 80)
	tt.SetTime(time.Unix(1000, 0))
	if !tt.HasText("act") {
		t.Fatal("dialog controls missing while the panel fades in")
	}
	if err := tt.Click("act"); err != nil {
		t.Fatal(err)
	}
	if !clicked {
		t.Fatal("a control inside a fading dialog did not take the click")
	}
}

// TestPopoverRevealFadesMenuIn runs the popover token through a real
// PopoverBase panel, whose own lifetime measures one opening.
func TestPopoverRevealFadesMenuIn(t *testing.T) {
	open := true
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(0, 0, 0))
		anchor := ui.Button(c, "anchor")
		ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
			progress := Reveal(c, panel, PopoverDuration)
			panel.Label("reveal-menu").Key("reveal-menu").Width(40).Height(20).Radius(8).Opacity(progress).
				Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, ui.RGB(255, 0, 0), 8) })
		})
	}, 120, 60)
	tt.SetTime(start)
	menu, ok := tt.Find("reveal-menu")
	if !ok {
		t.Fatal("popover panel missing")
	}
	x, y := int(menu.X+menu.W/2), int(menu.Y+menu.H/2)
	hidden := tt.Image().RGBAAt(x, y).R
	tt.SetTime(start.Add(PopoverDuration / 2))
	half := tt.Image().RGBAAt(x, y).R
	tt.SetTime(start.Add(PopoverDuration))
	settled := tt.Image().RGBAAt(x, y).R
	if !(hidden < half && half < settled) {
		t.Fatalf("popover fade did not progress: %d → %d → %d", hidden, half, settled)
	}
	if settled < 250 {
		t.Fatalf("menu never reached full opacity: %d", settled)
	}
}
