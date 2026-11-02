package pluginui

import "reflect"

// Pixel invalidation is independent of the pinned input geometry. Focus rings,
// selection and caret pixels may change on press without moving its target.
func sameMirrorInputGeometry(a, b mirrorGeometry) bool {
	a.Dirty, b.Dirty = 0, 0
	a.Animated, b.Animated = false, false
	return reflect.DeepEqual(a, b)
}
