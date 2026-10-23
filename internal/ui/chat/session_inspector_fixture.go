//go:build inspectorfixture

package chat

import "time"

// InspectorRevealFixtureState exposes the in-memory landing to the opt-in
// desktop probe. Production builds have no fixture diagnostics.
func InspectorRevealFixtureState(n *InspectorNavigation) map[string]any {
	if n == nil {
		return nil
	}
	r := &n.reveal
	age := float64(-1)
	if !r.started.IsZero() {
		age = float64(time.Since(r.started)) / float64(time.Millisecond)
	}
	return map[string]any{"pending": r.pending, "locating": r.locating, "waiting": r.waiting, "ageMs": age, "rect": r.rect}
}
