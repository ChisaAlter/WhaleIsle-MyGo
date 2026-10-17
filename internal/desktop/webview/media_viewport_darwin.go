//go:build darwin

package webview

import (
	"errors"

	"github.com/egoist/mygo"
)

func clipMediaNativeWindow(w *mygo.Window, _, _ int, holes []mygo.Rectangle) error {
	// This also moves the existing WKWebView into its parent's NSWindow.
	// The backend's sibling hit testing excludes the same painted holes.
	if !w.SetBrowserOcclusions(holes) {
		return errors.New("media: macOS cannot attach and clip this native browser")
	}
	return nil
}
