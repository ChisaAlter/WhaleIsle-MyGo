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
	calls := make([]map[string]any, 0, len(r.toolFull))
	for key, box := range r.toolFull {
		calls = append(calls, map[string]any{"owner": key.owner, "call": key.call, "full": box, "visible": r.toolRows[key]})
	}
	return map[string]any{"call": r.callID, "calls": calls, "pickCall": n.picker.previewCallID, "pending": r.pending, "locating": r.locating, "waiting": r.waiting, "ageMs": age, "rect": r.rect, "moving": r.centerList != nil && r.centerList.ScrollAnimating(), "scale": r.scale, "settling": !r.settledAt.IsZero(), "pickArea": n.picker.area, "pickPreview": n.picker.preview, "pickPreviewID": n.picker.previewID, "pickRows": r.rows, "pickPressed": n.picker.pressed}
}
