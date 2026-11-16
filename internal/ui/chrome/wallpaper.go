package chrome

import (
	"encoding/base64"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

const (
	maxWallpaperCacheEntries = 8
	wallpaperNegativeTTL     = 15 * time.Second
)

type wallpaperLRUEntry struct {
	bmp      *ui.Bitmap
	failedAt time.Time
}

var wallpaperCache struct {
	mu       sync.Mutex
	entries  map[string]wallpaperLRUEntry
	order    []string
	inFlight map[string]bool
}

func wallpaperTouchLRU(order []string, key string) []string {
	for i, existing := range order {
		if existing == key {
			order = append(order[:i], order[i+1:]...)
			break
		}
	}
	return append(order, key)
}

func wallpaperStoreBounded[V any](entries map[string]V, order *[]string, limit int, key string, value V) {
	entries[key] = value
	*order = wallpaperTouchLRU(*order, key)
	for len(*order) > limit {
		oldest := (*order)[0]
		*order = (*order)[1:]
		delete(entries, oldest)
	}
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
	var invalidate func()
	if c != nil {
		invalidate = c.Services().Invalidate
	}
	bmp := loadWallpaperBitmapAsync(path, invalidate)
	if bmp == nil {
		return
	}
	ui.Overlay(c, func() {
		ui.Image(c, bmp).Absolute().Left(0).Top(0).Right(0).Bottom(0).Fit(ui.Cover)
	})
}

func loadWallpaperBitmapAsync(path string, invalidate func()) *ui.Bitmap {
	now := time.Now()
	wallpaperCache.mu.Lock()
	if entry, ok := wallpaperCache.entries[path]; ok {
		if entry.bmp != nil {
			wallpaperCache.order = wallpaperTouchLRU(wallpaperCache.order, path)
			wallpaperCache.mu.Unlock()
			return entry.bmp
		}
		if now.Sub(entry.failedAt) < wallpaperNegativeTTL {
			wallpaperCache.order = wallpaperTouchLRU(wallpaperCache.order, path)
			wallpaperCache.mu.Unlock()
			return nil
		}
		delete(wallpaperCache.entries, path)
	}
	if wallpaperCache.inFlight[path] {
		wallpaperCache.mu.Unlock()
		return nil
	}
	if wallpaperCache.entries == nil {
		wallpaperCache.entries = make(map[string]wallpaperLRUEntry, maxWallpaperCacheEntries)
	}
	if wallpaperCache.inFlight == nil {
		wallpaperCache.inFlight = make(map[string]bool)
	}
	wallpaperCache.inFlight[path] = true
	wallpaperCache.mu.Unlock()

	go func() {
		var bmp *ui.Bitmap
		if data, err := wallpaperImageBytes(path); err == nil {
			if decoded, err := ui.DecodeBitmap(data); err == nil {
				bmp = decoded
			}
		}
		wallpaperCache.mu.Lock()
		delete(wallpaperCache.inFlight, path)
		entry := wallpaperLRUEntry{bmp: bmp}
		if bmp == nil {
			entry.failedAt = time.Now()
		}
		wallpaperStoreBounded(wallpaperCache.entries, &wallpaperCache.order, maxWallpaperCacheEntries, path, entry)
		wallpaperCache.mu.Unlock()
		if bmp != nil && invalidate != nil {
			invalidate()
		}
	}()
	return nil
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
