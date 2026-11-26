//go:build !release

package boot

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/renderpolicy"
)

func TestRevealCurtainsSplitsAt62PercentAndStopsIdle(t *testing.T) {
	var curtains RevealCurtains
	curtains.Trigger(true, false, false)
	if !curtains.Active() {
		t.Fatal("curtains did not arm")
	}

	start := time.Unix(1000, 0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Box(c).Fill().Background(ui.RGB(255, 0, 0))
		curtains.Paint(c)
	}, 200, 100)
	tt.SetTime(start)

	// At t=0, both top sky and bottom sea cover the red workbench.
	img0 := tt.Image()
	if got := img0.RGBAAt(100, 30); got.R > 40 {
		t.Fatalf("at start sky curtain did not cover workbench: %+v", got)
	}
	if got := img0.RGBAAt(100, 80); got.R > 40 {
		t.Fatalf("at start sea curtain did not cover workbench: %+v", got)
	}

	// Halfway through 600ms (with DesignEase ~0.78 along), center around the 62%
	// horizon (y=62) is already revealed while top/bottom edges remain covered.
	tt.SetTime(start.Add(RevealDuration / 2))
	imgHalf := tt.Image()
	if got := imgHalf.RGBAAt(100, 62); got.R < 250 {
		t.Fatalf("at midpoint horizon center not revealed: %+v", got)
	}
	if got := imgHalf.RGBAAt(100, 5); got.R > 40 {
		t.Fatalf("at midpoint top sky edge already gone too early: %+v", got)
	}
	if got := imgHalf.RGBAAt(100, 95); got.R > 40 {
		t.Fatalf("at midpoint bottom sea edge already gone too early: %+v", got)
	}

	// After 600ms, curtains retire cleanly and stop requesting frames (0 Hz).
	tt.SetTime(start.Add(RevealDuration))
	if curtains.Active() {
		t.Fatal("curtains remained active after RevealDuration")
	}
	imgDone := tt.Image()
	if got := imgDone.RGBAAt(100, 5); got.R < 250 {
		t.Fatalf("after reveal workbench top edge still covered: %+v", got)
	}
	builds = 0
	tt.Frame()
	if builds != 1 {
		t.Fatalf("settled reveal caused %d extra builds, want 1 (0 Hz)", builds)
	}
}

func TestRevealCurtainsRespectsReduceMotionAndMinimal(t *testing.T) {
	var curtains RevealCurtains
	curtains.Trigger(false, false, true)
	if curtains.Active() {
		t.Fatal("curtains armed despite reduceMotion")
	}

	renderpolicy.SetMinimal(true)
	defer renderpolicy.SetMinimal(false)
	curtains.Trigger(false, false, false)
	if curtains.Active() {
		t.Fatal("curtains armed under renderpolicy.Minimal()")
	}
}
