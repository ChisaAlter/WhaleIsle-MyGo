package frame

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// AppFrame's grid tracks use the shared 300ms cubic-bezier(.4, 0, .2, 1).
// Transition changes real layout boxes (including native child bounds), not
// painter transforms. MyGo already bypasses it on window resize/reduced motion.
var shellTrackTransition = ui.ElementTransition{
	Duration: widgets.SlowDuration,
	Ease:     widgets.DesignEase,
	Position: true,
	Size:     true,
}

func shellTrackEase(progress float32) float32 {
	return widgets.DesignEase(progress)
}
