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

var whaleHeadDecoded = sync.OnceValue(func() image.Image {
	src, err := png.Decode(bytes.NewReader(whaleHeadPNG))
	if err != nil {
		return nil
	}
	return src
})

var whaleHeadOnce sync.Once
var whaleHeadBMP *ui.Bitmap

var fishLogoOnce sync.Once
var fishLogoBMP *ui.Bitmap

var railWashOnce sync.Once
var railWashBMP *ui.Bitmap

// WarmBrandAssets pre-decodes brand bitmaps and alpha masks on a background
// worker so the first UI frame never blocks on 1254x1254 PNG decompression.
func WarmBrandAssets() {
	_ = BrandAvatar()
	_ = officialHeroAlpha()
	_ = FishLogo()
	_ = RailWash()
}

// WarmHeroBrandInks pre-generates tinted hero bitmaps for the built-in theme
// palettes on a background worker after critical startup icons are ready.
func WarmHeroBrandInks() {
	for _, hex := range []string{
		"#0f1115", "#81858c", "#f9fafb", "#adb2b8",
		"#0d0d0d", "#8a8a8a", "#f5f5f5", "#8e8e8e",
		"#262624", "#8f8d84", "#f5f4ef",
	} {
		_ = HeroBrandMark(Color(hex))
	}
}

// BrandAvatar returns the shared MyGo鲸屿 mascot bitmap (nil if decode fails).
func BrandAvatar() *ui.Bitmap {
	whaleHeadOnce.Do(func() {
		if src := whaleHeadDecoded(); src != nil {
			whaleHeadBMP = ui.NewBitmap(src)
			whaleHeadBMP.WarmLevels(64)
			return
		}
		bmp, err := ui.DecodeBitmap(whaleHeadPNG)
		if err == nil {
			bmp.WarmLevels(64)
			whaleHeadBMP = bmp
		}
	})
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
