package boot

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/widgets"
)

func TestBootCurtainSplitAndExit(t *testing.T) {
	r := ui.Rect{X: 0, Y: 48, W: 1000, H: 800}
	sky, sea := curtainRects(r, 0)
	if sky.H != 496 || sea.H != 304 || sea.Y != 544 {
		t.Fatalf("not 62/38 split: %v %v", sky, sea)
	}
	sky, sea = curtainRects(r, 1)
	if sky.Y+sky.H != r.Y || sea.Y != r.Y+r.H {
		t.Fatalf("ready curtains cover content: %v %v", sky, sea)
	}
}

// TestBootCurtainCoverage pins the covered area: the boot page's box, from just
// under the shell caption strip (48 DIP, where cmd/whaleisle paints the boot
// body) to the bottom of the window, full width (boot-reveal.css inset:0 inside
// a curtain window sized to getContentBounds, window.js:494).
func TestBootCurtainCoverage(t *testing.T) {
	box := curtainCoverage(1060, 660)
	if box.X != 0 || box.Y != 48 || box.W != 1060 || box.H != 612 {
		t.Fatalf("coverage = %+v, want 0,48 1060x612", box)
	}
	sky, sea := curtainRects(box, 0)
	if sky.X != 0 || sky.W != 1060 || sky.Y != 48 || sea.Y+sea.H != 660 {
		t.Fatalf("covered frame leaves gaps: %v %v", sky, sea)
	}
	// A window shorter than the caption strip never asks for a negative box.
	if box := curtainCoverage(300, 20); box.H < 0 || box.W != 300 {
		t.Fatalf("degenerate coverage = %+v", box)
	}
}

// TestBootCurtainRigidSlices pins the translation: each window shows a rigid
// slice, so the scene moves exactly as far as its window does
// (translateY(-100%) / translateY(100%), boot-reveal.css:12-13).
func TestBootCurtainRigidSlices(t *testing.T) {
	r := ui.Rect{X: 10, Y: 48, W: 1000, H: 800}
	for _, progress := range []float32{0, .25, .5, .78, 1} {
		sky, sea := curtainRects(r, progress)
		skyShift, seaShift := curtainShifts(r, progress)
		if want := -r.H * .62 * progress; skyShift != want || sky.Y != r.Y+skyShift {
			t.Fatalf("p=%v sky window %v shift %v want %v", progress, sky, skyShift, want)
		}
		if want := r.H * .38 * progress; seaShift != want || sea.Y != r.Y+r.H*.62+seaShift {
			t.Fatalf("p=%v sea window %v shift %v want %v", progress, sea, seaShift, want)
		}
		if sky.H != r.H*.62 || sea.H != r.H*.38 {
			t.Fatalf("p=%v split %v %v", progress, sky, sea)
		}
	}
}

// TestBootCurtainDesignTiming samples the release on a fixed clock: the 600ms
// duration is calc(var(--ds-transition-duration,.2s) * 3) and the curve is
// --ds-ease-in-out, cubic-bezier(.4,0,.2,1) (boot-reveal.css:6), which is
// widgets.DesignEase, not the framework's symmetric EaseInOut.
func TestBootCurtainDesignTiming(t *testing.T) {
	start := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	var samples []float32
	curtainProgressHook = func(p float32) { samples = append(samples, p) }
	defer func() { curtainProgressHook = nil }()
	m := Model{Status: harness.Status{State: harness.Starting}, ThemeID: "deepseek"}
	tt := ui.NewTester(func(c *ui.Context) { bootOverlay(c, m) }, 1000, 800)
	tt.SetTime(start)
	if len(samples) == 0 || samples[len(samples)-1] != 0 {
		t.Fatalf("covered frame progress = %v, want 0", samples)
	}

	box := curtainCoverage(1000, 800)
	m.Status.State = harness.Ready
	for _, tc := range []struct {
		at   time.Duration
		want float32
	}{
		{0, 0},
		{150 * time.Millisecond, widgets.DesignEase(.25)},
		{300 * time.Millisecond, widgets.DesignEase(.5)},
		{600 * time.Millisecond, 1},
	} {
		samples = samples[:0]
		tt.SetTime(start.Add(tc.at))
		if len(samples) == 0 {
			t.Fatalf("+%v: reveal frame missing (already finished?)", tc.at)
		}
		got := samples[len(samples)-1]
		if got != tc.want {
			t.Fatalf("+%v progress = %v, want %v", tc.at, got, tc.want)
		}
		// The offsets at that sample: sky up by 62% of its height, sea down by
		// 38%, both linear in progress (the easing lives in time, not space).
		sky, sea := curtainRects(box, got)
		if want := box.Y - box.H*.62*got; sky.Y != want {
			t.Fatalf("+%v sky.Y = %v, want %v", tc.at, sky.Y, want)
		}
		if want := box.Y + box.H*.62 + box.H*.38*got; sea.Y != want {
			t.Fatalf("+%v sea.Y = %v, want %v", tc.at, sea.Y, want)
		}
	}
	if got := widgets.DesignEase(.5); got < .77 || got > .79 {
		t.Fatalf("design ease midpoint = %v, want about .78", got)
	}
	// A finished reveal leaves no curtain frame behind it.
	samples = samples[:0]
	tt.SetTime(start.Add(700 * time.Millisecond))
	if len(samples) != 0 {
		t.Fatalf("reveal kept painting after 600ms: %v", samples)
	}
}

// TestBootCurtainReducedMotionJumps pins the reference's
// `@media (prefers-reduced-motion: reduce) { transition: none }`
// (boot-reveal.css:14): the release lands on the finished offsets at once.
func TestBootCurtainReducedMotionJumps(t *testing.T) {
	start := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	var samples []float32
	curtainProgressHook = func(p float32) { samples = append(samples, p) }
	defer func() { curtainProgressHook = nil }()
	m := Model{Status: harness.Status{State: harness.Starting}, ThemeID: "deepseek"}
	tt := ui.NewTester(func(c *ui.Context) { bootOverlay(c, m) }, 1000, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTime(start)
	m.Status.State = harness.Ready
	samples = samples[:0]
	tt.SetTime(start)
	if len(samples) == 0 || samples[len(samples)-1] != 1 {
		t.Fatalf("reduced motion progress = %v, want 1 at once", samples)
	}
	sky, sea := curtainRects(curtainCoverage(1000, 800), 1)
	if sky.Y+sky.H != 48 || sea.Y != 800 {
		t.Fatalf("reduced motion did not land on the finished offsets: %v %v", sky, sea)
	}
}
