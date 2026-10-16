package frame

import (
	"github.com/egoist/mygo/ui"
	"time"
)

// AppFrame's grid tracks use the shared 300ms cubic-bezier(.4, 0, .2, 1).
// Transition changes real layout boxes (including native child bounds), not
// painter transforms. MyGo already bypasses it on window resize/reduced motion.
var shellTrackTransition = ui.ElementTransition{
	Duration: 300 * time.Millisecond,
	Ease:     shellTrackEase,
	Position: true,
	Size:     true,
}

func shellTrackEase(progress float32) float32 {
	if progress <= 0 {
		return 0
	}
	if progress >= 1 {
		return 1
	}
	// Invert the timing curve's x coordinate before evaluating y. ui.EaseInOut
	// is a different polynomial and does not match the source product curve.
	lo, hi := float32(0), float32(1)
	for i := 0; i < 18; i++ {
		t := (lo + hi) / 2
		u := 1 - t
		x := 3*u*u*t*.4 + 3*u*t*t*.2 + t*t*t
		if x < progress {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	return 3*(1-t)*t*t + t*t*t
}
