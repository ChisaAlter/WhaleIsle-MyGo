//go:build !windows

package frame

// windowPixels has no device-pixel source outside Windows; the viewport is
// already the client size in DIPs there.
func windowPixels() (float32, float32, uint32) { return 0, 0, 0 }
