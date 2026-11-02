package pluginui

import "math"

// NoteMirrorNativeFragment records the clipped native position of the original
// full row. It is paint-only metadata; source scroll runs in the next worker.
func (o *Owner) NoteMirrorNativeFragment(tail SessionTurnTail, relativeCSSOffset, targetViewportY float64) {
	if o == nil || math.IsNaN(relativeCSSOffset) || math.IsNaN(targetViewportY) || math.IsInf(relativeCSSOffset, 0) || math.IsInf(targetViewportY, 0) || relativeCSSOffset < 0 || targetViewportY < 0 {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror != nil && !o.closed && !o.mirror.pendingAnchor.known {
		o.mirror.pendingAnchor = mirrorNativeAnchor{known: true, tail: tail, fragment: true, u: relativeCSSOffset, v: targetViewportY}
	}
}
