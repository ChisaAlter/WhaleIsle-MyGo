//go:build !windows && !darwin

package frame

// No owned native screen is available through this frame's non-Windows pixel
// source; match the reference's missing-screen viewport-aspect fallback.
func displayLandscape() (landscape, known bool) { return false, false }
