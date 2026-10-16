//go:build !windows

package webview

import "github.com/egoist/mygo"

// Other backends need their own native-surface clipping implementation before
// overlapping live browser floats have parity with the Windows desktop target.
func setNativeOcclusions(_ *mygo.Window, _, _ int, _ []mygo.Rectangle) {}
