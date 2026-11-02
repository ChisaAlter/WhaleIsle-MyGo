//go:build darwin

package frame

import "github.com/egoist/mygo"

// displayLandscape follows the work area of the explicitly bound main window,
// not its viewport or whichever display is primary. MyGo's Darwin Screen source
// reads NSScreen.visibleFrame (excluding the Dock and menu bar), in the same
// screen coordinates as Window.Bounds. Re-read it so moves and Dock changes do
// not leave a stale orientation. No owner means no native call in headless tests.
func displayLandscape() (landscape, known bool) {
	return frameDisplayWindow.landscape(func(window *mygo.Window) (bounds mygo.Rectangle, displays []mygo.Display, ok bool) {
		// A native view builds on the UI thread, so RunOnMain is the immediate
		// path, with no goroutine, post or wait on each frame. It also protects
		// the native snapshot if an internal caller ever runs off that thread.
		mygo.RunOnMain(func() {
			if window.IsDestroyed() || !window.IsVisible() {
				return
			}
			bounds = window.Bounds()
			if bounds.Width <= 0 || bounds.Height <= 0 {
				return
			}
			displays = mygo.Screen.Displays()
			ok = true
		})
		return
	})
}
