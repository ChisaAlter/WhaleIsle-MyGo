package chrome

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/whale-head.png
var whaleHeadPNG []byte

//go:embed assets/fish-logo.png
var fishLogoPNG []byte

//go:embed assets/rail-wash.png
var railWashPNG []byte

var whaleAssetsOnce sync.Once
var whaleHeadBMP *ui.Bitmap
var whaleHeroAlpha *image.Alpha

var fishLogoOnce sync.Once
var fishLogoBMP *ui.Bitmap

var railWashOnce sync.Once
var railWashBMP *ui.Bitmap

// loadWhaleBrandAssets decodes whale-head.png once, extracts its 1-byte alpha
// silhouette (1.5 MiB) for bit-exact mip-tiled HeroBrandMark tinting, streams
// BrandAvatar directly to its compact 157×157 mip level (98 KiB), and lets the
// 1254×1254 decoded source image be garbage-collected immediately.
func loadWhaleBrandAssets() {
	whaleAssetsOnce.Do(func() {
		src, err := png.Decode(bytes.NewReader(whaleHeadPNG))
		if err != nil {
			bmp, err := ui.DecodeBitmap(whaleHeadPNG)
			if err == nil {
				bmp.WarmLevels(32)
				whaleHeadBMP = bmp
			}
			return
		}
		avatarRGBA, alphaMask := extractBrandAssets(src)
		whaleHeroAlpha = alphaMask
		if avatarRGBA != nil {
			whaleHeadBMP = ui.NewBitmap(avatarRGBA)
			whaleHeadBMP.WarmLevels(8)
		}
	})
}

func halvePixel4(a, b, c, d [4]uint8) [4]uint8 {
	return [4]uint8{
		uint8((uint32(a[0]) + uint32(b[0]) + uint32(c[0]) + uint32(d[0]) + 2) / 4),
		uint8((uint32(a[1]) + uint32(b[1]) + uint32(c[1]) + uint32(d[1]) + 2) / 4),
		uint8((uint32(a[2]) + uint32(b[2]) + uint32(c[2]) + uint32(d[2]) + 2) / 4),
		uint8((uint32(a[3]) + uint32(b[3]) + uint32(c[3]) + uint32(d[3]) + 2) / 4),
	}
}

// halve3Tiled downsamples a w0×h0 pixel source by 3 exact ui.Bitmap 2×2
// halvings (1254 → 627 → 314 → 157) in 8×8 stack tiles without allocating any
// full-resolution intermediate RGBA buffers.
func halve3Tiled(w0, h0 int, sample func(x, y int) [4]uint8) *image.RGBA {
	if w0 <= 0 || h0 <= 0 {
		return nil
	}
	w1, h1 := max(1, (w0+1)/2), max(1, (h0+1)/2)
	w2, h2 := max(1, (w1+1)/2), max(1, (h1+1)/2)
	w3, h3 := max(1, (w2+1)/2), max(1, (h2+1)/2)
	out := image.NewRGBA(image.Rect(0, 0, w3, h3))
	for oy := 0; oy < h3; oy++ {
		outRow := out.Pix[oy*out.Stride:]
		for ox := 0; ox < w3; ox++ {
			var l1 [4][4][4]uint8
			for dy := 0; dy < 4; dy++ {
				y2 := min(2*oy+dy/2, h2-1)
				y1 := min(2*y2+dy%2, h1-1)
				y0a, y0b := 2*y1, min(2*y1+1, h0-1)
				for dx := 0; dx < 4; dx++ {
					x2 := min(2*ox+dx/2, w2-1)
					x1 := min(2*x2+dx%2, w1-1)
					x0a, x0b := 2*x1, min(2*x1+1, w0-1)
					l1[dy][dx] = halvePixel4(
						sample(x0a, y0a),
						sample(x0b, y0a),
						sample(x0a, y0b),
						sample(x0b, y0b),
					)
				}
			}
			var l2 [2][2][4]uint8
			for iy := 0; iy < 2; iy++ {
				for ix := 0; ix < 2; ix++ {
					l2[iy][ix] = halvePixel4(
						l1[2*iy][2*ix],
						l1[2*iy][2*ix+1],
						l1[2*iy+1][2*ix],
						l1[2*iy+1][2*ix+1],
					)
				}
			}
			p := halvePixel4(l2[0][0], l2[0][1], l2[1][0], l2[1][1])
			i := ox * 4
			outRow[i] = p[0]
			outRow[i+1] = p[1]
			outRow[i+2] = p[2]
			outRow[i+3] = p[3]
		}
	}
	return out
}

func extractBrandAssets(src image.Image) (*image.RGBA, *image.Alpha) {
	if src == nil {
		return nil, nil
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, nil
	}
	alpha := image.NewAlpha(image.Rect(0, 0, w, h))
	switch img := src.(type) {
	case *image.NRGBA:
		for y := 0; y < h; y++ {
			srcRow := img.Pix[y*img.Stride:]
			dstAlpha := alpha.Pix[y*alpha.Stride : y*alpha.Stride+w]
			for x := 0; x < w; x++ {
				dstAlpha[x] = srcRow[x*4+3]
			}
		}
		avatar := halve3Tiled(w, h, func(x, y int) [4]uint8 {
			i := y*img.Stride + x*4
			a := uint32(img.Pix[i+3])
			if a == 0 {
				return [4]uint8{}
			}
			return [4]uint8{
				uint8((uint32(img.Pix[i]) * 0x101 * a / 0xff) >> 8),
				uint8((uint32(img.Pix[i+1]) * 0x101 * a / 0xff) >> 8),
				uint8((uint32(img.Pix[i+2]) * 0x101 * a / 0xff) >> 8),
				uint8(a),
			}
		})
		return avatar, alpha
	case *image.RGBA:
		for y := 0; y < h; y++ {
			srcRow := img.Pix[y*img.Stride:]
			dstAlpha := alpha.Pix[y*alpha.Stride : y*alpha.Stride+w]
			for x := 0; x < w; x++ {
				dstAlpha[x] = srcRow[x*4+3]
			}
		}
		avatar := halve3Tiled(w, h, func(x, y int) [4]uint8 {
			i := y*img.Stride + x*4
			return [4]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
		})
		return avatar, alpha
	default:
		for y := 0; y < h; y++ {
			dstAlpha := alpha.Pix[y*alpha.Stride : y*alpha.Stride+w]
			for x := 0; x < w; x++ {
				_, _, _, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
				dstAlpha[x] = uint8(a >> 8)
			}
		}
		avatar := halve3Tiled(w, h, func(x, y int) [4]uint8 {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			return [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
		})
		return avatar, alpha
	}
}

// WarmBrandAssets pre-decodes brand bitmaps and the compact alpha mask on a
// background worker so the first UI frame never blocks on PNG decompression.
func WarmBrandAssets() {
	_ = BrandAvatar()
	_ = officialHeroAlpha()
	_ = FishLogo()
	_ = RailWash()
}

// WarmHeroBrandInks is retained for caller compatibility. Tinted hero marks are
// now compact 157×157 bitmaps generated lazily on demand in HeroBrandMark so
// startup never allocates 11 full-resolution theme bitmaps up front.
func WarmHeroBrandInks() {}

// BrandAvatar returns the shared MyGo鲸屿 mascot bitmap (nil if decode fails).
func BrandAvatar() *ui.Bitmap {
	loadWhaleBrandAssets()
	return whaleHeadBMP
}

// FishLogo returns the Electron EmptyHero fish silhouette (FISH_LOGO_PATH),
// rasterized at 68×50 for crisp 34 DIP display (nil if decode fails).
func FishLogo() *ui.Bitmap {
	fishLogoOnce.Do(func() {
		bmp, err := ui.DecodeBitmap(fishLogoPNG)
		if err == nil {
			bmp.WarmLevels(8)
			fishLogoBMP = bmp
		}
	})
	return fishLogoBMP
}

// RailWash returns the Electron-sampled desk-rail vertical wash (nil if decode fails).
func RailWash() *ui.Bitmap {
	railWashOnce.Do(func() {
		bmp, err := ui.DecodeBitmap(railWashPNG)
		if err == nil {
			railWashBMP = bmp
		}
	})
	return railWashBMP
}
