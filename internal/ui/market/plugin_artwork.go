package market

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/desktop/media"
)

// marketMedia is the process image service the Market draws from; tests
// replace it with a service they drive.
var marketMedia = media.Default

// requestImage asks the media service for a data: URL or public http(s)
// image, sized for a w×h DIP slot (zero: intrinsic). Build code only reads the
// returned handle: decoding, base64 and downloads run on media workers.
func requestImage(owner *media.Owner, source string, w, h float32) *media.Handle {
	var key media.Key
	var src media.Source
	switch {
	case strings.HasPrefix(source, "data:image/"):
		key, src = media.DataURL(source)
	case imgcache.IsHTTPURL(source):
		key, src = media.Remote(source)
	default:
		return nil
	}
	// Two device pixels per DIP keeps artwork crisp on high-DPI screens.
	return marketMedia().Request(owner, key.Sized(int(2*w), int(2*h)), src)
}

// showImage draws a ready handle's current frame, scheduling the next one
// only while it is painted (a hidden animation costs nothing).
func showImage(c *ui.Context, h *media.Handle) ui.Element {
	bmp, wait := h.Frame(time.Now())
	if wait > 0 {
		c.After(wait)
	}
	if bmp != nil {
		return ui.Image(c, bmp)
	}
	p, _ := h.Picture()
	return ui.Image(c, p.SVG)
}

// PluginImage presents package metadata artwork and bundled previews through
// the process media service. A missing, pending or failed image leaves its
// caller's default artwork in place, as the original isolated img element did.
func PluginImage(c *ui.Context, source string, width, height float32, owner *media.Owner) bool {
	ready, _ := pluginImage(c, source, width, height, owner)
	return ready
}

func pluginImage(c *ui.Context, source string, width, height float32, owner *media.Owner) (bool, *media.Handle) {
	if source == "" {
		return false, nil
	}
	h := requestImage(owner, source, width, height)
	if h.State() != media.Ready {
		return false, h
	}
	w, ht := h.Size()
	if w <= 0 || ht <= 0 {
		return false, h
	}
	root := ui.Box(c)
	if height <= 0 {
		root.Width(width).MaxWidthPercent(100).MinWidth(0).AspectRatio(w / ht)
	} else {
		root.Width(width).Height(height).Shrink(0)
	}
	root.Clip().Children(func() { showImage(c, h).Fill().Fit(ui.Contain) })
	return true, h
}
