package chat

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// PublishedPageSurface reuses the native clipped seat and pixel/input fallback
// used by published chat replies. The caller owns authorization and lifetime.
func PublishedPageSurface(c *ui.Context, surface session.VisualReplySurface, themeID string, dark bool, height float32) {
	if surface == nil {
		return
	}
	c.After(100 * time.Millisecond)
	visualReplySurfaceSeat(c, surface, themeID, dark, height, nil)
}
