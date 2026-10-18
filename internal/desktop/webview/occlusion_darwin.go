//go:build darwin

package webview

import "github.com/egoist/mygo"

func setNativeOcclusions(win *mygo.Window, _, _ int, holes []mygo.Rectangle) {
	win.SetBrowserOcclusions(holes)
}
