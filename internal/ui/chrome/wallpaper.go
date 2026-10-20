package chrome

import (
	"encoding/base64"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

var wallpaperCache struct {
	mu   sync.Mutex
	path string
	bmp  *ui.Bitmap
}

// WallpaperBackdrop draws a full-window cover image when path is a readable
// raster file. mygo has no WindowOptions.BackgroundImage yet; this is the
// content-layer stub until native backdrop support exists.
//
// It paints the bare image only: no dim mask, no glass sampling and no ambient
// effects. The shell's live wallpaper layer is WallpaperBackgroundPainter;
// keep this for a caller that wants the untouched image.
func WallpaperBackdrop(c *ui.Context, path string) {
	path = theme.NormalizeWallpaperPath(path)
	if path == "" {
		return
	}
	bmp := loadWallpaperBitmap(path)
	if bmp == nil {
		return
	}
	ui.Overlay(c, func() {
		ui.Image(c, bmp).Absolute().Left(0).Top(0).Right(0).Bottom(0).Fit(ui.Cover)
	})
}

func loadWallpaperBitmap(path string) *ui.Bitmap {
	wallpaperCache.mu.Lock()
	defer wallpaperCache.mu.Unlock()
	if wallpaperCache.path == path && wallpaperCache.bmp != nil {
		return wallpaperCache.bmp
	}
	data, err := wallpaperImageBytes(path)
	if err != nil {
		wallpaperCache.path = path
		wallpaperCache.bmp = nil
		return nil
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		wallpaperCache.path = path
		wallpaperCache.bmp = nil
		return nil
	}
	wallpaperCache.path = path
	wallpaperCache.bmp = bmp
	return bmp
}

// Host ui-theme stores the image itself. Decode accepted data URLs in memory;
// local paths belong only to the existing launcher/offline configuration.
func wallpaperImageBytes(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "data:") {
		return os.ReadFile(source)
	}
	header, payload, _ := strings.Cut(source, ",")
	if strings.HasSuffix(header, ";base64") {
		return base64.StdEncoding.DecodeString(payload)
	}
	decoded, err := url.PathUnescape(payload)
	return []byte(decoded), err
}
