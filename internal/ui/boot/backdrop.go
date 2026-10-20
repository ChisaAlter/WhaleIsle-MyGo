package boot

import (
	"github.com/egoist/mygo/ui"
)

// bootBackdrop paints the boot page's static scene: the reference page's sky
// gradient, horizon glow, suspended particles and the abyss band, without any
// animation. Layers are fixed so a redraw is pixel-identical; the moving
// version belongs to the later scene pass.
func bootBackdrop(bg, accent ui.Color, dark bool) func(*ui.Painter, ui.Rect) {
	deep := bg.Mix(accent, .10)
	if dark {
		deep = bg.Mix(accent, .18)
	}
	return func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{From: bg, To: deep, Angle: 90}, 0)

		// Horizon: one wide, very soft glow at 62% height (boot.css .horizon).
		width, height := r.W*.78, r.H*.22
		p.Shadow(
			ui.Rect{X: r.X + (r.W-width)/2, Y: r.Y + r.H*.62 - height*.5, W: width, H: height},
			height*.5, 0, 0, r.W*.16, 0, accent.Alpha(.10),
		)

		// Suspended particles: a fixed lattice with alternating radii, denser
		// toward the bottom where the reference page's snow drifts.
		snow := accent.Alpha(.16)
		if dark {
			snow = ui.RGBA(255, 255, 255, .14)
		}
		for i := 0; i < 22; i++ {
			fx := float32((i*37)%100) / 100
			fy := float32((i*53)%100) / 100
			x := r.X + r.W*(.04+.92*fx)
			y := r.Y + r.H*(.10+.80*fy)
			d := float32(1.5)
			if i%3 == 0 {
				d = 2.5
			}
			if i%5 == 0 {
				d = 3.5
			}
			p.Fill(ui.Rect{X: x, Y: y, W: d, H: d}, snow.Alpha(float32(.06+.10*float32(i%4))/2), d/2)
		}

		// Abyss band under the horizon line.
		abyssTop := r.Y + r.H*.80
		p.FillGradient(
			ui.Rect{X: r.X, Y: abyssTop, W: r.W, H: r.H - (abyssTop - r.Y)},
			ui.LinearGradient{From: deep.Alpha(0), To: bg.Mix(ui.RGBA(0, 0, 0, .5), .55), Angle: 90},
			0,
		)
	}
}
