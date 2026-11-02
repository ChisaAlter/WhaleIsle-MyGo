package chrome

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

var desktopGlassVignette = sync.OnceValue(func() *ui.Bitmap {
	const n = 128
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := (float64(x)+.5)/n*2 - 1
			dy := (float64(y)+.5)/n*2 - 1
			// A broad elliptical vignette dims the corners most, without a
			// visible edge or a per-frame desktop readback/blur allocation.
			d := math.Min(1, (dx*dx+dy*dy)/1.6)
			a := uint8(math.Round(255 * .26 * d * d))
			img.SetRGBA(x, y, color.RGBA{A: a})
		}
	}
	return ui.NewBitmap(img)
})

// DesktopGlassVignette paints the dim veil and corners over native Acrylic.
// Desktop blur itself belongs to DWM, so moving windows behind it updates the
// material without frame timers or screenshots. Wallpaper sampling is separate.
func DesktopGlassVignette(p *ui.Painter, r ui.Rect, prefs wallpapers.Preferences) {
	if !p.Theme().Dark || r.W <= 0 || r.H <= 0 {
		return
	}
	alpha := float32(.08) + float32(clampPercent(prefs.Glass)/100)*.14
	p.Fill(r, ui.RGBA(0, 0, 0, alpha), 0)
	p.Image(desktopGlassVignette(), r, ui.FillBox)
}
