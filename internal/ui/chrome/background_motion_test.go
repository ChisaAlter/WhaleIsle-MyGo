package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// This simulates desktop preferences through public MyGo UI APIs. It changes
// no OS setting and makes no GPU-fluid or native OS-toggle acceptance claim.
func TestWallpaperReducedMotionStopsSchedulingAndHoldsPhase(t *testing.T) {
	prefs := wallpapers.Defaults()
	prefs.Effect, prefs.Speed, prefs.Count = "gradient", 190, 5
	prefs.EffectExplicit = true
	paints := 0
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter("", prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			paints++
			paint(p, r)
		})
	}, 280, 180)
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	before := paints
	tt.Frame()
	staticPaints := paints - before
	if staticPaints != 1 {
		t.Fatalf("reduced frame scheduled more painting: %d", staticPaints)
	}
	still := bytes.Clone(tt.Image().Pix)
	time.Sleep(120 * time.Millisecond)
	tt.Frame()
	if !bytes.Equal(still, tt.Image().Pix) {
		t.Fatal("reduced ambient phase changed with wall clock")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: false})
	// The gradient schedules its next phase with Painter.After and renders it on a
	// worker, so a headless frame no longer repaints in a loop and the number of
	// paints says nothing. What must hold is that the normal gradient keeps
	// following the clock: drive it with a fixed later time, wait for the worker
	// and paint again.
	moving := bytes.Clone(tt.Image().Pix)
	ambientGradientFrames.mu.Lock()
	renders := ambientGradientFrames.renders
	ambientGradientFrames.mu.Unlock()
	tt.SetTime(time.Now().Add(120 * time.Millisecond))
	waitRenders(t, &ambientGradientFrames, renders+1)
	tt.Frame()
	if bytes.Equal(moving, tt.Image().Pix) {
		t.Fatal("normal gradient phase no longer follows time")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if !bytes.Equal(still, tt.Image().Pix) {
		t.Fatal("normal-to-reduced did not return to its static phase")
	}
}
