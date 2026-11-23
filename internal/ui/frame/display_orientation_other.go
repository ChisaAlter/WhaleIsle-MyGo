//go:build !windows && !darwin

package frame

import "github.com/egoist/mygo/ui"

// MyGo reads no screen here outside Windows, and this frame no native one;
// match the reference's missing-screen viewport-aspect fallback.
func displayLandscape(*ui.Context) (landscape, known bool) { return false, false }
