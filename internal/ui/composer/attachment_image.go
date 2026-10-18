package composer

import (
	"image"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
	"whaleisle/internal/desktop/imagepreview"
)

// Motion belongs to one mounted attachment seat, never a global image cache.
// Original-preview seats are released on dismissal by paintAttachmentOriginal.
type attachmentMotion struct {
	animation *imagepreview.Animation
	frames    []*ui.Bitmap
	started   time.Time
}

func attachmentBitmap(decoded image.Image, full bool) *ui.Bitmap {
	if !full {
		bounds := decoded.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		if w > 240 || h > 160 {
			ratio := min(float64(240)/float64(w), float64(160)/float64(h))
			dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*ratio)), max(1, int(float64(h)*ratio))))
			draw.ApproxBiLinear.Scale(dst, dst.Bounds(), decoded, bounds, draw.Src, nil)
			decoded = dst
		}
	}
	return ui.NewBitmap(decoded)
}

func (v *attachmentDisplay) decodeAnimation(data []byte, full bool) {
	// Shared validation bounds bytes, frames and cumulative composited pixels
	// before GIF allocation, and preserves disposal, delay and finite-loop rules.
	animation, err := imagepreview.DecodeAnimation(data)
	if err != nil {
		v.err = err
		return
	}
	motion := &attachmentMotion{animation: animation}
	for i, frame := range animation.Frames {
		motion.frames = append(motion.frames, attachmentBitmap(frame, full))
		animation.Frames[i] = nil
	}
	// Retain only the timing metadata and the bounded display bitmaps.
	animation.Frames = nil
	v.motion = motion
	v.bitmap = motion.frames[0]
}

// advance is paint-driven, matching the Files viewer. Hidden/closed surfaces
// schedule no recurring timer; finite GIFs stop and late paints skip old frames.
func (v *attachmentDisplay) advance(now time.Time) time.Duration {
	if v == nil || v.motion == nil {
		return 0
	}
	m := v.motion
	if m.started.IsZero() {
		m.started = now
	}
	index, wait := m.animation.FrameAt(now.Sub(m.started))
	v.bitmap = m.frames[index]
	return wait
}

func (v *attachmentDisplay) animate(c *ui.Context) {
	if wait := v.advance(time.Now()); wait > 0 {
		c.After(wait)
	}
}
