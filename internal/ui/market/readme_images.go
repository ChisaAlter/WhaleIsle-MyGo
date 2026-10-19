package market

import (
	"os"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

var readmeRemoteImages struct {
	sync.RWMutex
	cache *imgcache.Cache
}

// SetRemoteImageCache shares the existing desktop image cache. The document
// never reads a local-file destination or invents a base URL for relative media.
func SetRemoteImageCache(cache *imgcache.Cache) {
	readmeRemoteImages.Lock()
	readmeRemoteImages.cache = cache
	readmeRemoteImages.Unlock()
}

type readmeDecodedImage struct {
	path   string
	bitmap *ui.Bitmap
	svg    *ui.SVG
	width  float32
	height float32
}

func renderReadmeImage(c *ui.Context, source, alt string) bool {
	if !imgcache.IsHTTPURL(source) {
		return false
	}
	readmeRemoteImages.RLock()
	cache := readmeRemoteImages.cache
	readmeRemoteImages.RUnlock()
	if cache == nil {
		return false
	}
	status, path, _ := cache.State(source)
	if status == imgcache.StatusMissing {
		services := c.Services()
		cache.Ensure(source, services.Invalidate)
		return false
	}
	if status != imgcache.StatusReady {
		return false
	}
	root := ui.Box(c.Key("market-readme-image:" + source)).Label("README image " + source)
	decoded := ui.Local(root, "decoded-image", func() readmeDecodedImage { return readmeDecodedImage{} })
	if decoded.path != path {
		*decoded = readmeDecodedImage{path: path}
		if data, err := os.ReadFile(path); err == nil {
			if bitmap, err := ui.DecodeBitmap(data); err == nil {
				decoded.bitmap = bitmap
				width, height := bitmap.Size()
				decoded.width, decoded.height = float32(width), float32(height)
			} else if svg, err := ui.ParseSVG(data); err == nil {
				decoded.svg = svg
				decoded.width, decoded.height = svg.Size()
			}
		}
	}
	if decoded.width <= 0 || decoded.height <= 0 {
		return false
	}
	// Original .readme img: max-width100%, height:auto. Keep the source's
	// natural width and ratio; no chat-thumbnail cap, caption or extra action.
	root.Width(decoded.width).MaxWidthPercent(100).MinWidth(0).AspectRatio(decoded.width / decoded.height).Radius(12).Clip().Tooltip(alt).Children(func() {
		if decoded.bitmap != nil {
			ui.Image(c, decoded.bitmap).Fill().Fit(ui.Contain)
		} else {
			ui.Image(c, decoded.svg).Fill().Fit(ui.Contain)
		}
	})
	return true
}
