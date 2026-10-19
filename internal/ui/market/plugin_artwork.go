package market

import (
	"bytes"
	"encoding/base64"
	"image"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/ui/svgimage"
)

// PluginImage presents package metadata artwork and bundled previews through
// the existing bounded Market media decoder. A missing/failed image leaves its
// caller's default artwork in place, matching the original isolated img element.
// Browser-backed SVGs require the native window's existing image owner; only
// visible PaintClipped calls renew the shared seat's lease.
func PluginImage(c *ui.Context, source string, width, height float32, owners ...*svgimage.Owner) bool {
	if source == "" {
		return false
	}
	path := ""
	var data []byte
	if strings.HasPrefix(source, "data:image/") {
		head, encoded, ok := strings.Cut(source, ",")
		if !ok || !strings.HasSuffix(head, ";base64") || len(encoded) > int(imagepreview.MaxBytes*4/3+4) {
			return false
		}
		var err error
		data, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil || int64(len(data)) > imagepreview.MaxBytes {
			return false
		}
		path = source
	} else {
		if !imgcache.IsHTTPURL(source) {
			return false
		}
		readmeRemoteImages.RLock()
		cache := readmeRemoteImages.cache
		readmeRemoteImages.RUnlock()
		if cache == nil {
			return false
		}
		status, cached, _ := cache.State(source)
		if status == imgcache.StatusMissing {
			cache.Ensure(source, c.Services().Invalidate)
			return false
		}
		if status != imgcache.StatusReady {
			return false
		}
		path = cached
	}
	root := ui.Box(c.Key("plugin-image:" + source))
	decoded := ui.Local(root, "decoded-image", func() readmeDecodedImage { return readmeDecodedImage{} })
	if decoded.path != path {
		if decoded.svgSeat != nil {
			decoded.svgSeat.Close()
		}
		*decoded = readmeDecodedImage{path: path}
		if data == nil {
			decoded.load(path, c.Services().Invalidate)
		} else {
			loadPluginImageData(decoded, data, c.Services().Invalidate)
		}
	}
	decoded.poll(path)
	if decoded.width <= 0 || decoded.height <= 0 {
		return false
	}
	if decoded.svgSeat != nil {
		var owner *svgimage.Owner
		if len(owners) > 0 {
			owner = owners[0]
		}
		if owner == nil {
			decoded.svgSeat.Close()
			return false
		}
		decoded.svgSeat.Prepare(c, owner)
		if decoded.svgSeat.Error() != nil {
			return false
		}
	} else if decoded.bitmap == nil && decoded.svg == nil {
		return false
	}
	if wait := decoded.advance(time.Now()); wait > 0 {
		c.After(wait)
	}
	if height <= 0 {
		root.Width(width).MaxWidthPercent(100).MinWidth(0).AspectRatio(decoded.width / decoded.height)
	} else {
		root.Width(width).Height(height).Shrink(0)
	}
	root.Clip().Children(func() {
		if decoded.svgSeat != nil {
			ui.Box(c).Fill().DrawOver(func(p *ui.Painter, r ui.Rect) {
				decoded.svgSeat.PaintClipped(p, r, ui.Contain, 0, nil)
			})
		} else if decoded.bitmap != nil {
			ui.Image(c, decoded.bitmap).Fill().Fit(ui.Contain)
		} else {
			ui.Image(c, decoded.svg).Fill().Fit(ui.Contain)
		}
	})
	return true
}
func loadPluginImageData(decoded *readmeDecodedImage, data []byte, notify func()) {
	if imagepreview.NeedsBrowserSVG(data) {
		decoded.svgSeat, _ = svgimage.NewSeat(data)
		if decoded.svgSeat != nil {
			decoded.width, decoded.height = decoded.svgSeat.Size()
		}
		return
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil && (cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width) > imagepreview.MaxPixels/int64(cfg.Height)) {
		return
	}
	animated := format == "gif" || imagepreview.IsAnimatedWebP(data) || imagepreview.IsAnimatedAVIF(data)
	if err == nil && !animated {
		decoded.bitmap, _ = ui.DecodeBitmap(data)
		decoded.size()
		return
	}
	svg := supportedReadmeSVG(data)
	ico := len(data) >= 4 && bytes.Equal(data[:4], []byte{0, 0, 1, 0})
	avif := len(data) >= 12 && string(data[4:8]) == "ftyp"
	if !animated && !svg && !ico && !avif {
		return
	}
	decoded.pending = make(chan readmeDecodedImage, 1)
	pending := decoded.pending
	go func() {
		readmeDecodeSlots <- struct{}{}
		value := decodeReadmeMedia(data, animated, svg, ico)
		<-readmeDecodeSlots
		pending <- value
		notify()
	}()
}
