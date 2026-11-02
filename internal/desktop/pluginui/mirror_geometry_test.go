package pluginui

import "testing"

func TestMirrorInputPinsGeometryWithoutTreatingFocusPixelsAsMovement(t *testing.T) {
	a := mirrorGeometry{DocumentKey: "A", Revision: 4, Dirty: 12, Owners: []MirrorOccurrence{{Handle: "tail", Rect: MirrorRect{X: 20, Y: -200, W: 700, H: 3200}}}}
	b := a
	b.Dirty = 13
	b.Animated = true
	if !sameMirrorInputGeometry(a, b) {
		t.Fatal("focus pixels invalidated a pinned original gesture")
	}
	b.Owners = append([]MirrorOccurrence(nil), a.Owners...)
	b.Owners[0].Rect.Y++
	if sameMirrorInputGeometry(a, b) {
		t.Fatal("real target movement retained original gesture authority")
	}
	b = a
	b.DocumentKey = "B"
	if sameMirrorInputGeometry(a, b) {
		t.Fatal("new document retained old gesture")
	}
}
