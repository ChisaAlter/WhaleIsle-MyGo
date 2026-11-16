package chrome

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// BackgroundEffectPreview owns a separate frame cache. Editing a scheme must
// not replace the window's backdrop or the glass surface sampling state.
type BackgroundEffectPreview struct{ frames gradientFrameCache }

func (p *BackgroundEffectPreview) Close() { p.frames.reset() }

func (p *BackgroundEffectPreview) Draw(settings wallpapers.Preferences, reduced bool) func(*ui.Painter, ui.Rect) {
	var startedAt time.Time
	return func(painter *ui.Painter, bounds ui.Rect) {
		if bounds.W <= 0 || bounds.H <= 0 {
			return
		}
		now := painter.Now()
		config := gradientConfiguration(settings, float64(bounds.W), float64(bounds.H), painter.Theme().Dark, reduced)
		freeze := shouldFreezeAmbientAnimation(reduced, &startedAt, now)
		var frame *gradientFrame
		if freeze {
			frame = p.frames.frozen(config, now)
		} else {
			frame = p.frames.latest(config, now)
		}
		if frame != nil {
			painter.Image(frame.bitmap, bounds, ui.FillBox)
		}
		if !reduced && !freeze {
			painter.After(gradientFrameInterval)
		}
	}
}
