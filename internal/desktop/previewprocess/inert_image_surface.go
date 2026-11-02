package previewprocess

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// ImageSurfaceShape is trusted native image geometry, never supplied by SVG.
type ImageSurfaceShape struct {
	Bounds ui.Rect
	Radius float32
}
type ImageSurfaceClip struct {
	Bounds  ui.Rect
	Radius  float32
	Exclude []ImageSurfaceShape
}

// PaintImagePixels reuses the existing one-frame-in-flight helper stream for an
// inert image seat. It adds no focus or input handler. The helper renders the
// contained image at its fixed viewport center; Cover crops only the empty
// margins when the native caller supplies the image's aspect-fitted rectangle.
// Windows returns false and retains its verified native child presentation.
func (s *Session) PaintImagePixels(p *ui.Painter, r ui.Rect) bool {
	ps := s.pixels
	if ps == nil {
		return false
	}
	select {
	case <-s.Done():
		return true
	default:
	}
	clip := p.NativeSurfaceClip()
	visible := min(r.X+r.W, clip.X+clip.W) > max(r.X, clip.X) && min(r.Y+r.H, clip.Y+clip.H) > max(r.Y, clip.Y)
	ps.mu.Lock()
	if visible {
		ps.lastPaint = time.Now()
	}
	bitmap := ps.bitmap
	ps.mu.Unlock()
	if bitmap != nil {
		p.Clip(r, 0, func() { p.Image(bitmap, r, ui.Cover) })
	}
	// Ask for the next frame only while the surface can be seen; one scrolled out
	// of its clip would otherwise keep waking the window ten times a second.
	if visible {
		p.After(100 * time.Millisecond)
	}
	return true
}
