package chrome

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

// WallpaperBackdrop draws a full-window cover image when path is a readable
// raster file or data URL. It paints the bare image only: no dim mask, no
// glass sampling and no ambient effects. The shell's live wallpaper layer is
// WallpaperBackgroundPainter; keep this for a caller that wants the untouched
// image. The decode runs on the media service.
func WallpaperBackdrop(c *ui.Context, path string) {
	path = theme.NormalizeWallpaperPath(path)
	if path == "" {
		return
	}
	pic, ok := wallpaperRequest(path, wallpapers.Preferences{}, 0, 0).Picture()
	if !ok || pic.Bitmap == nil {
		return
	}
	ui.Overlay(c, func() {
		ui.Image(c, pic.Bitmap).Absolute().Left(0).Top(0).Right(0).Bottom(0).Fit(ui.Cover)
	})
}
