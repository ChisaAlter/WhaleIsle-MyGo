package market

import (
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/ui/svgimage"
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
	svgSeat   *svgimage.Seat
	path      string
	bitmap    *ui.Bitmap
	svg       *ui.SVG
	width     float32
	height    float32
	animation *imagepreview.Animation
	frames    []*ui.Bitmap
	started   time.Time
	pending   chan readmeDecodedImage
}

func renderReadmeImage(c *ui.Context, source, alt string, owners ...*svgimage.Owner) bool {
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
		if decoded.svgSeat != nil {
			decoded.svgSeat.Close()
		}
		*decoded = readmeDecodedImage{path: path}
		decoded.load(path, c.Services().Invalidate)
	}
	decoded.poll(path)
	if decoded.svgSeat != nil {
		var owner *svgimage.Owner
		if len(owners) > 0 {
			owner = owners[0]
		}
		decoded.svgSeat.Prepare(c, owner)
		decoded.width, decoded.height = decoded.svgSeat.Size()
	}
	if decoded.width <= 0 || decoded.height <= 0 {
		return false
	}
	if wait := decoded.advance(time.Now()); wait > 0 {
		c.After(wait)
	}
	// Original .readme img: max-width100%, height:auto. Keep the source's
	// natural width and ratio; no chat-thumbnail cap, caption or extra action.
	root.Width(decoded.width).MaxWidthPercent(100).MinWidth(0).AspectRatio(decoded.width / decoded.height).Radius(12).Clip().Tooltip(alt).Children(func() {
		if decoded.svgSeat != nil {
			ui.Box(c).Fill().DrawOver(func(p *ui.Painter, r ui.Rect) { decoded.svgSeat.PaintClipped(p, r, ui.Contain, 12, nil) })
		} else if decoded.bitmap != nil {
			ui.Image(c, decoded.bitmap).Fill().Fit(ui.Contain)
		} else {
			ui.Image(c, decoded.svg).Fill().Fit(ui.Contain)
		}
	})
	return true
}
