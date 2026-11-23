package market

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/desktop/media"
)

// renderReadmeImage draws a public README image at its natural width. The
// document never reads a local-file destination or invents a base URL for
// relative media; downloads and decoding belong to the media service.
func renderReadmeImage(c *ui.Context, source, alt string, owner *media.Owner) bool {
	if !imgcache.IsHTTPURL(source) {
		return false
	}
	h := requestImage(owner, source, 0, 0)
	if h.State() != media.Ready {
		return false
	}
	w, ht := h.Size()
	if w <= 0 || ht <= 0 {
		return false
	}
	// Original .readme img: max-width100%, height:auto. Keep the source's
	// natural width and ratio; no chat-thumbnail cap, caption or extra action.
	root := ui.Box(c.Key("market-readme-image:" + source)).Label("README image " + source)
	root.Width(w).MaxWidthPercent(100).MinWidth(0).AspectRatio(w / ht).Radius(12).Clip().Tooltip(alt).Children(func() {
		showImage(c, h).Fill().Fit(ui.Contain)
	})
	return true
}
