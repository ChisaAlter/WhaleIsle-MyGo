package chrome

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
)

// BackgroundEffectPreview owns an independent animation clock so editing a
// scheme does not alter or reset the main window's backdrop phase.
type BackgroundEffectPreview struct{ clock ambientAnimationClock }

func (p *BackgroundEffectPreview) Close() { p.clock.reset() }

func (p *BackgroundEffectPreview) Draw(settings wallpapers.Preferences, reduced bool) func(*ui.Painter, ui.Rect) {
	var startedAt time.Time
	return func(painter *ui.Painter, bounds ui.Rect) {
		if bounds.W <= 0 || bounds.H <= 0 {
			return
		}
		now := painter.Now()
		freeze := shouldFreezeAmbientAnimation(reduced, &startedAt, now) || renderpolicy.NoBackdrop()
		elapsed, nextIn := p.clock.stepPaced(now, reduced, freeze, ambientFrameInterval)
		config := gradientConfiguration(settings, float64(bounds.W), float64(bounds.H), painter.Theme().Dark, reduced)
		config.ambientBackdrop(elapsed).PaintMaterial(painter, bounds, [4]float32{})
		if !reduced && !freeze {
			painter.After(nextIn)
		}
	}
}
