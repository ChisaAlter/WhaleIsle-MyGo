package chrome

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

func TestDefaultGradientIsASoftFullBleedWash(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter("", prefs, true)
		ui.Box(c).Fill().Draw(paint)
	}, 320, 220)
	image := tt.Image()
	if gradientBackdrop.frame != nil || ambientGradientFrames.frame != nil {
		t.Fatal("GPU ambient gradient triggered legacy CPU frame cache")
	}
	if backgroundState.colors == nil {
		t.Fatal("paint did not populate representative backdrop color map")
	}
	if _, ok := SampleBackdrop(ui.Rect{W: 40, H: 40}); !ok || backgroundState.box.W != 320 || backgroundState.box.H != 220 {
		t.Fatalf("sample box=%v valid=%v", backgroundState.box, ok)
	}
	for _, pt := range [][2]int{{2, 2}, {317, 2}, {2, 217}, {317, 217}} {
		c := image.RGBAAt(pt[0], pt[1])
		if c.A != 255 || c.R == 255 && c.G == 255 && c.B == 255 {
			t.Fatalf("empty backdrop corner %v=%v", pt, c)
		}
	}
	worst := 0
	for y := 0; y < image.Rect.Dy(); y++ {
		for x := 1; x < image.Rect.Dx(); x++ {
			a, b := image.RGBAAt(x-1, y), image.RGBAAt(x, y)
			for _, d := range []int{absInt(int(a.R) - int(b.R)), absInt(int(a.G) - int(b.G)), absInt(int(a.B) - int(b.B))} {
				worst = max(worst, d)
			}
		}
	}
	if worst > 12 {
		t.Fatalf("soft source backdrop contains %d-level step", worst)
	}
	before := bytes.Clone(image.Pix)
	tt.Frame()
	if !bytes.Equal(before, tt.Image().Pix) {
		t.Fatal("reduced source changed across repaint")
	}
	ConfigureBackdrop("", wallpapers.Defaults())
	if gradientBackdrop.frame != nil || ambientGradientFrames.frame != nil || backgroundState.colors != nil {
		t.Fatal("closed gradient retained stale bitmap/cache")
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
