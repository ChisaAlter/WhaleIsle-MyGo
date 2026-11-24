package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// The ambient gradient is baked once per configuration into a static
// texture: frames at later clock times, with or without reduced motion,
// render nothing new and schedule no repaint loop.
func TestAmbientGradientIsBakedOnceAndNeverTicks(t *testing.T) {
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
	ambientGradientFrames.mu.Lock()
	renders := ambientGradientFrames.renders
	ambientGradientFrames.mu.Unlock()
	still := bytes.Clone(tt.Image().Pix)
	tt.SetTime(time.Now().Add(2 * time.Second))
	before := paints
	tt.Frame()
	if paints-before != 1 {
		t.Fatalf("the gradient scheduled more painting: %d", paints-before)
	}
	time.Sleep(120 * time.Millisecond)
	tt.SetTime(time.Now().Add(5 * time.Second))
	tt.Frame()
	if !bytes.Equal(still, tt.Image().Pix) {
		t.Fatal("the ambient gradient changed with the clock")
	}
	ambientGradientFrames.mu.Lock()
	defer ambientGradientFrames.mu.Unlock()
	if ambientGradientFrames.renders != renders || ambientGradientFrames.rendering {
		t.Fatalf("the gradient rendered %d more frames", ambientGradientFrames.renders-renders)
	}
}
