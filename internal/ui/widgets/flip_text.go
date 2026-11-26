package widgets

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
)

// FlipTextOptions configures a 400ms vertical flip-text control
// (motion.css: `perspective: 140px`, `rotateX(80deg)` -> `0` / `-80deg`).
type FlipTextOptions = coss.FlipTextOptions

// FlipText paints a single-line label that flips vertically along rotateX over
// FlipDuration (400ms) on DesignEase when its value changes. When settled or
// under Preferences.ReduceMotion it returns a plain ui.Text with 0 Hz idle
// overhead.
func FlipText(c *ui.Context, value string, opts FlipTextOptions) ui.Element {
	return coss.FlipText(c, value, opts)
}
