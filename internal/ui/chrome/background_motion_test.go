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
	before = paints
	tt.Frame()
	if paints-before <= staticPaints {
		t.Fatal("normal gradient stopped requesting animation frames")
	}
	moving := bytes.Clone(tt.Image().Pix)
	time.Sleep(120 * time.Millisecond)
	tt.Frame()
	if bytes.Equal(moving, tt.Image().Pix) {
		t.Fatal("normal gradient phase no longer follows time")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if !bytes.Equal(still, tt.Image().Pix) {
		t.Fatal("normal-to-reduced did not return to its static phase")
	}
}
