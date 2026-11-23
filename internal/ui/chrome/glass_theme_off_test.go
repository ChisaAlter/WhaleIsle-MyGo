package chrome

import (
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/theme"
)

// With the glass theme off (the default), a profile that stores a wallpaper
// and an explicit ambient gradient paints neither: no wallpaper decode, no
// gradient frame, no glass sampling, no frost bitmap and no extra goroutine,
// however many frames run.
func TestGlassThemeOffRunsNoBackdropMachinery(t *testing.T) {
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(false)
	ambientGradientFrames.reset()
	ambientGradientFrames.mu.Lock()
	rendersBefore := ambientGradientFrames.renders
	ambientGradientFrames.mu.Unlock()
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	stored := wallpapers.Defaults()
	stored.Effect, stored.EffectExplicit, stored.Preset = "gradient", true, "aurora"
	stored.Transparent = true
	path := renderpolicy.Wallpaper("C:/wallpapers/sea.png")
	prefs := renderpolicy.Background(wallpapers.NormalizeEffect(path, stored))
	painter := WallpaperBackgroundPainter(path, prefs)
	before := runtime.NumGoroutine()
	colors := theme.Resolve("deepseek.dark").Active(true)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Draw(painter).Children(func() {
			GlassSurface(ui.Box(c).Size(200, 80), Color(colors.BG), MaterialPanel, 12)
			MenuSurface(ui.Box(c).Size(120, 60), MenuBaseFill(colors, true), MenuBorderColor(colors, true), 8)
			if _, ok := SampleBackdrop(ui.Rect{W: 40, H: 40}); ok {
				t.Error("sampled a backdrop with the glass theme off")
			}
		})
	}, 640, 400)
	for range 20 {
		tt.Frame()
	}
	time.Sleep(50 * time.Millisecond)
	if BackdropActive() {
		t.Fatal("backdrop active with the glass theme off")
	}
	ambientGradientFrames.mu.Lock()
	renders, frame := ambientGradientFrames.renders, ambientGradientFrames.frame
	ambientGradientFrames.mu.Unlock()
	if renders != rendersBefore || frame != nil || gradientBackdrop.frame != nil {
		t.Fatalf("ambient gradient rendered %d frames", renders-rendersBefore)
	}
	if backgroundState.colors != nil {
		t.Fatal("wallpaper pixels prepared with the glass theme off")
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines grew from %d to %d", before, after)
	}
}
