package chrome

import (
	"context"
	"image"
	"math"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// Painting and sampling share the same immutable masked ambient generation.
var gradientBackdrop struct {
	frame *gradientFrame
	box   ui.Rect
}

func publishGradientBackdrop(frame *gradientFrame, box ui.Rect) {
	if frame == nil {
		return
	}
	if gradientBackdrop.frame != frame || gradientBackdrop.box != box {
		backgroundState.version++
		gradientPanelFrames.reset()
	}
	gradientBackdrop.frame, gradientBackdrop.box = frame, box
	backgroundState.pixels, backgroundState.box, backgroundState.hasBox = frame.pixels, box, true
	backgroundState.dark = frame.config.dark
}
func clearGradientBackdrop() {
	if gradientBackdrop.frame != nil {
		gradientBackdrop.frame = nil
		gradientBackdrop.box = ui.Rect{}
		gradientPanelFrames.reset()
	}
}

// SettingsPanelGlass paints one Panel-opacity fill after the settings CSS
// backdrop-filter, blur(40px) saturate(150%). canvas and modalMask are explicit
// colors from the actual caller; modalMask includes reveal alpha. They are only
// reconstructed in the sampler, not drawn again onto the canvas.
func SettingsPanelGlass(base, canvas, modalMask ui.Color, radius ...float32) func(*ui.Painter, ui.Rect) {
	round := firstRadius(radius)
	return func(p *ui.Painter, r ui.Rect) {
		prefs := effectivePrefs(backgroundState.prefs)
		if !backgroundState.active || gradientBackdrop.frame == nil {
			DrawGlass(p, r, base, MaterialPanel, round)
			return
		}
		alpha := MaterialAlpha(MaterialPanel, prefs, p.Theme().Dark)
		if alpha <= 0 {
			return
		}
		if bmp := gradientPanelFrames.get(gradientBackdrop.frame, canvas, modalMask, prefs, p.Theme().Dark); bmp != nil {
			p.Clip(r, round, func() { p.Image(bmp, gradientBackdrop.box, ui.FillBox) })
		}
		p.Fill(r, MaterialFill(base, alpha), round)
	}
}

type gradientPanelKey struct {
	frame             *gradientFrame
	canvas, modalMask ui.Color
	glass             float64
	dark              bool
}

type gradientBlurKey struct {
	pixels           *image.RGBA
	w, h             int
	configW, configH float64
	firstPix         [4]uint8
}

type gradientPanelEntry struct {
	key    gradientPanelKey
	bitmap *ui.Bitmap
}

type gradientPanelCache struct {
	mu       sync.Mutex
	entries  [8]gradientPanelEntry
	next     int
	blurKey  gradientBlurKey
	satBlur  []float32
	rgbaKey  gradientPanelKey
	rgbaBlur gradientBlurKey
	rgba     *image.RGBA
}

func (s *gradientPanelCache) reset() {
	s.mu.Lock()
	s.entries = [8]gradientPanelEntry{}
	s.next = 0
	s.blurKey = gradientBlurKey{}
	s.satBlur = nil
	s.rgbaKey = gradientPanelKey{}
	s.rgbaBlur = gradientBlurKey{}
	s.rgba = nil
	s.mu.Unlock()
}

func (s *gradientPanelCache) get(frame *gradientFrame, canvas, modalMask ui.Color, prefs wallpapers.Preferences, dark bool) *ui.Bitmap {
	key := gradientPanelKey{frame: frame, canvas: canvas, modalMask: modalMask, glass: prefs.Glass, dark: dark}
	s.mu.Lock()
	for i := range s.entries {
		if s.entries[i].bitmap != nil && s.entries[i].key == key {
			bmp := s.entries[i].bitmap
			s.mu.Unlock()
			return bmp
		}
	}
	s.mu.Unlock()
	pixels := gradientPanelPixels(frame, canvas, prefs, dark, modalMask)
	if pixels == nil {
		return nil
	}
	bmp := ui.NewBitmap(pixels)
	s.mu.Lock()
	s.entries[s.next] = gradientPanelEntry{key: key, bitmap: bmp}
	s.next = (s.next + 1) % len(s.entries)
	s.mu.Unlock()
	return bmp
}

var gradientPanelFrames gradientPanelCache

func makeGradientBlurKey(frame *gradientFrame) gradientBlurKey {
	src := frame.pixels
	var first [4]uint8
	if len(src.Pix) >= 4 {
		copy(first[:], src.Pix[:4])
	}
	return gradientBlurKey{
		pixels:   src,
		w:        src.Rect.Dx(),
		h:        src.Rect.Dy(),
		configW:  frame.config.width,
		configH:  frame.config.height,
		firstPix: first,
	}
}

func (s *gradientPanelCache) getSatBlur(frame *gradientFrame) ([]float32, gradientBlurKey) {
	bkey := makeGradientBlurKey(frame)
	s.mu.Lock()
	if s.satBlur != nil && s.blurKey == bkey {
		res := s.satBlur
		s.mu.Unlock()
		return res, bkey
	}
	s.mu.Unlock()

	src := frame.pixels
	w, h := bkey.w, bkey.h
	values, temp := make([]float32, w*h*4), make([]float32, w*h*4)
	const inv255 = 1.0 / 255.0
	for i := 0; i < len(values); i += 4 {
		values[i] = float32(src.Pix[i]) * inv255
		values[i+1] = float32(src.Pix[i+1]) * inv255
		values[i+2] = float32(src.Pix[i+2]) * inv255
		values[i+3] = 1
	}
	sx, sy := 40*float64(w)/frame.config.width, 40*float64(h)/frame.config.height
	if !gradientGaussianClamped(context.Background(), values, temp, w, h, sx, sy) {
		return nil, bkey
	}
	satBlur := make([]float32, w*h*3)
	for i, j := 0, 0; i < len(values); i, j = i+4, j+3 {
		r, g, b := float64(values[i]), float64(values[i+1]), float64(values[i+2])
		lum := .213*r + .715*g + .072*b
		satBlur[j] = float32(lum + 1.5*(r-lum))
		satBlur[j+1] = float32(lum + 1.5*(g-lum))
		satBlur[j+2] = float32(lum + 1.5*(b-lum))
	}

	s.mu.Lock()
	s.blurKey = bkey
	s.satBlur = satBlur
	s.mu.Unlock()
	return satBlur, bkey
}

func gradientPanelPixels(frame *gradientFrame, canvas ui.Color, prefs wallpapers.Preferences, dark bool, masks ...ui.Color) *image.RGBA {
	if frame == nil || frame.pixels == nil {
		return nil
	}
	modalMask := ui.RGBA(0, 0, 0, 61.0/255)
	if dark {
		modalMask = ui.RGBA(0, 0, 0, 128.0/255)
	}
	if len(masks) > 0 {
		modalMask = masks[0]
	}
	bkey := makeGradientBlurKey(frame)
	pkey := gradientPanelKey{frame: frame, canvas: canvas, modalMask: modalMask, glass: prefs.Glass, dark: dark}
	gradientPanelFrames.mu.Lock()
	if gradientPanelFrames.rgba != nil && gradientPanelFrames.rgbaKey == pkey && gradientPanelFrames.rgbaBlur == bkey {
		cached := gradientPanelFrames.rgba
		gradientPanelFrames.mu.Unlock()
		return cached
	}
	gradientPanelFrames.mu.Unlock()

	satBlur, bkey := gradientPanelFrames.getSatBlur(frame)
	if satBlur == nil {
		return nil
	}
	w, h := bkey.w, bkey.h
	canvasAlpha := MaterialAlpha(MaterialCanvas, prefs, dark)
	maskAlpha := float32(modalMask.A) / 255
	scale255 := float64((1 - canvasAlpha) * (1 - maskAlpha) * 255)

	var off [3]float64
	for ch, c := range [3]uint8{canvas.R, canvas.G, canvas.B} {
		below := float32(c) / 255 * canvasAlpha
		maskColor := ([3]uint8{modalMask.R, modalMask.G, modalMask.B})[ch]
		off[ch] = float64(float32(maskColor)/255*maskAlpha + below*(1-maskAlpha))
	}
	offLum := .213*off[0] + .715*off[1] + .072*off[2]
	satOff0 := (offLum + 1.5*(off[0]-offLum)) * 255
	satOff1 := (offLum + 1.5*(off[1]-offLum)) * 255
	satOff2 := (offLum + 1.5*(off[2]-offLum)) * 255

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	pix := out.Pix
	for i, j := 0, 0; i < len(pix); i, j = i+4, j+3 {
		r := satOff0 + scale255*float64(satBlur[j])
		g := satOff1 + scale255*float64(satBlur[j+1])
		b := satOff2 + scale255*float64(satBlur[j+2])
		pix[i] = clampRoundByte(r)
		pix[i+1] = clampRoundByte(g)
		pix[i+2] = clampRoundByte(b)
		pix[i+3] = 255
	}

	gradientPanelFrames.mu.Lock()
	gradientPanelFrames.rgbaKey = pkey
	gradientPanelFrames.rgbaBlur = bkey
	gradientPanelFrames.rgba = out
	gradientPanelFrames.mu.Unlock()
	return out
}

func clampRoundByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func gradientGaussianClamped(ctx context.Context, source, temp []float32, w, h int, sigmaX, sigmaY float64) bool {
	kx := gradientKernel(math.Min(gradientMaxSigma, sigmaX))
	ky := gradientKernel(math.Min(gradientMaxSigma, sigmaY))
	rx, ry, stride := len(kx)/2, len(ky)/2, w*4
	for y := 0; y < h; y++ {
		if ctx.Err() != nil {
			return false
		}
		row := source[y*stride : (y+1)*stride]
		dstRow := temp[y*stride : (y+1)*stride]
		for x := 0; x < w; x++ {
			var r, g, b, a float32
			for k, weight := range kx {
				sx := x + k - rx
				if sx < 0 {
					sx = 0
				} else if sx >= w {
					sx = w - 1
				}
				pos := sx * 4
				r += row[pos] * weight
				g += row[pos+1] * weight
				b += row[pos+2] * weight
				a += row[pos+3] * weight
			}
			dst := x * 4
			dstRow[dst], dstRow[dst+1], dstRow[dst+2], dstRow[dst+3] = r, g, b, a
		}
	}
	for y := 0; y < h; y++ {
		if ctx.Err() != nil {
			return false
		}
		dstRow := source[y*stride : (y+1)*stride]
		for x := 0; x < w; x++ {
			var r, g, b, a float32
			colOffset := x * 4
			for k, weight := range ky {
				sy := y + k - ry
				if sy < 0 {
					sy = 0
				} else if sy >= h {
					sy = h - 1
				}
				pos := sy*stride + colOffset
				r += temp[pos] * weight
				g += temp[pos+1] * weight
				b += temp[pos+2] * weight
				a += temp[pos+3] * weight
			}
			dstRow[colOffset], dstRow[colOffset+1], dstRow[colOffset+2], dstRow[colOffset+3] = r, g, b, a
		}
	}
	return true
}
