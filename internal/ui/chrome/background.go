package chrome

import (
	"github.com/egoist/mygo/ui"
	"image"
	"image/draw"
	"math"
	"os"
	"whaleisle/internal/desktop/wallpapers"
)

var backgroundState struct {
	prefs       wallpapers.Preferences
	active      bool
	path        string
	blur, pixel float64
	bitmap      *ui.Bitmap
}

func ConfigureBackdrop(path string, p wallpapers.Preferences) {
	backgroundState.prefs = p
	backgroundState.active = path != "" || p.Effect == "gradient"
}
func BackdropCanvas(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	p := backgroundState.prefs
	if p.Transparent {
		return c.Alpha(0)
	}
	k := p.Glass
	v := k * .375
	if k > 40 {
		v = 15 + (k-40)*.75
	}
	if k > 80 {
		v = 45 + (k-80)*2.75
	}
	return c.Alpha(float32(min(45.0, v) / 100))
}
func BackdropSidebar(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	p := backgroundState.prefs
	if p.Transparent || p.HideSidebarMask {
		return c.Alpha(0)
	}
	k := p.Glass
	v := k * .375
	if k > 40 {
		v = 15 + (k-40)*.75
	}
	if k > 80 {
		v = 45 + (k-80)*2.75
	}
	return c.Alpha(float32((k + v) / 200))
}
func BackdropSurface(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	if backgroundState.prefs.Transparent {
		return c.Alpha(0)
	}
	return c.Alpha(float32(backgroundState.prefs.Glass / 100))
}
func BackdropTerminal(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	return c.Alpha(float32(backgroundState.prefs.Terminal / 100))
}
func WallpaperBackgroundPainter(path string, p wallpapers.Preferences) func(*ui.Painter, ui.Rect) {
	ConfigureBackdrop(path, p)
	if path != "" {
		bmp := backgroundBitmap(path, p)
		return func(painter *ui.Painter, r ui.Rect) {
			if bmp != nil {
				painter.Image(bmp, r, ui.Cover)
			}
		}
	}
	if p.Effect != "gradient" {
		return func(*ui.Painter, ui.Rect) {}
	}
	return func(painter *ui.Painter, r ui.Rect) {
		colors := p.Colors
		resolved := [7]ui.Color{painter.Theme().Background, painter.Theme().Surface, painter.Theme().Accent, ui.Hex("#a78bfa"), ui.Hex("#38bdf8"), ui.Hex("#f472b6"), ui.Hex("#34d399")}
		for i := range colors {
			if colors[i] != "" {
				resolved[i] = Color(colors[i])
			}
		}
		t := float64(painter.Now().UnixMilli()) / 1000 * p.Speed / 100
		painter.FillGradient(r, ui.LinearGradient{From: resolved[0], To: resolved[1], Angle: 40 + float32(math.Sin(t*.05)*20)}, 0)
		for i := 0; i < min(5, max(1, int(p.Count))); i++ {
			phase := float64(i) * 1.73
			x := r.X + r.W*float32(.5+.36*math.Sin(t*.13+phase))
			y := r.Y + r.H*float32(.5+.35*math.Cos(t*.11+phase))
			w, h := r.W*.45, r.H*.55
			radius := min(w, h) / 2
			switch p.Variant {
			case "aurora":
				w = r.W * .8
				h = r.H * .18
				radius = h / 2
			case "rays":
				w = r.W * .18
				h = r.H * 1.4
				radius = 0
			case "chaos":
				w *= float32(.7 + .3*math.Sin(t*.2+phase))
				h *= float32(.8 + .4*math.Cos(t*.17+phase))
			}
			painter.Shadow(ui.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}, radius, 0, 0, 60, 0, resolved[i+2].Alpha(.38))
		}
		painter.AnimationFrame()
	}
}
func backgroundBitmap(path string, p wallpapers.Preferences) *ui.Bitmap {
	s := &backgroundState
	if s.path == path && s.blur == p.Blur && s.pixel == p.Pixelate && s.bitmap != nil {
		return s.bitmap
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := wallpapers.Decode(data)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	block := 1 + int(p.Pixelate/100*19)
	if block > 1 {
		for y := 0; y < b.Dy(); y += block {
			for x := 0; x < b.Dx(); x += block {
				col := dst.RGBAAt(min(x+block/2, b.Dx()-1), min(y+block/2, b.Dy()-1))
				for yy := y; yy < min(y+block, b.Dy()); yy++ {
					for xx := x; xx < min(x+block, b.Dx()); xx++ {
						dst.SetRGBA(xx, yy, col)
					}
				}
			}
		}
	}
	radius := int(p.Blur * .4)
	if radius > 0 {
		dst = blurBitmap(dst, radius)
	}
	s.path, s.blur, s.pixel = path, p.Blur, p.Pixelate
	s.bitmap = ui.NewBitmap(dst)
	return s.bitmap
}

// Separable running-sum blur keeps slider updates linear in the image size.
func blurBitmap(src *image.RGBA, radius int) *image.RGBA {
	b := src.Bounds()
	tmp := image.NewRGBA(b)
	dst := image.NewRGBA(b)
	for pass := 0; pass < 2; pass++ {
		in, out := src, tmp
		length, lines := b.Dx(), b.Dy()
		if pass == 1 {
			in, out = tmp, dst
			length, lines = b.Dy(), b.Dx()
		}
		for line := 0; line < lines; line++ {
			var sum [4]int
			at := func(pos int) int {
				if pass == 0 {
					return in.PixOffset(pos, line)
				}
				return in.PixOffset(line, pos)
			}
			for pos := 0; pos <= min(radius, length-1); pos++ {
				idx := at(pos)
				for ch := 0; ch < 4; ch++ {
					sum[ch] += int(in.Pix[idx+ch])
				}
			}
			for pos := 0; pos < length; pos++ {
				n := min(length-1, pos+radius) - max(0, pos-radius) + 1
				idx := at(pos)
				for ch := 0; ch < 4; ch++ {
					out.Pix[idx+ch] = byte(sum[ch] / n)
				}
				drop, add := pos-radius, pos+radius+1
				if drop >= 0 {
					idx = at(drop)
					for ch := 0; ch < 4; ch++ {
						sum[ch] -= int(in.Pix[idx+ch])
					}
				}
				if add < length {
					idx = at(add)
					for ch := 0; ch < 4; ch++ {
						sum[ch] += int(in.Pix[idx+ch])
					}
				}
			}
		}
	}
	return dst
}
