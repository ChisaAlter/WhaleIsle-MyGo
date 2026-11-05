package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// BackgroundEffectPreview owns a separate frame cache. Editing a scheme must
// not replace the window's backdrop or the glass surface sampling state.
type BackgroundEffectPreview struct{ frames gradientFrameCache }

func (p *BackgroundEffectPreview) Close() { p.frames.reset() }

func (p *BackgroundEffectPreview) Draw(settings wallpapers.Preferences, reduced bool) func(*ui.Painter, ui.Rect) {
	return func(painter *ui.Painter, bounds ui.Rect) {
		config := gradientConfiguration(settings, float64(bounds.W), float64(bounds.H), painter.Theme().Dark, reduced)
		if frame := p.frames.latest(config, painter.Now()); frame != nil {
			painter.Image(frame.bitmap, bounds, ui.FillBox)
		}
		if !reduced {
			painter.After(gradientFrameInterval)
		}
	}
}
