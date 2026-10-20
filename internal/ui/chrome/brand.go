package chrome

import (
	_ "embed"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/whale-head.png
var whaleHeadPNG []byte

//go:embed assets/fish-logo.png
var fishLogoPNG []byte

//go:embed assets/rail-wash.png
var railWashPNG []byte

var whaleHeadOnce sync.Once
var whaleHeadBMP *ui.Bitmap

var fishLogoOnce sync.Once
var fishLogoBMP *ui.Bitmap

var railWashOnce sync.Once
var railWashBMP *ui.Bitmap

// BrandAvatar returns the shared MyGo鲸屿 mascot bitmap (nil if decode fails).
func BrandAvatar() *ui.Bitmap {
	whaleHeadOnce.Do(func() {
		bmp, err := ui.DecodeBitmap(whaleHeadPNG)
		if err == nil {
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
