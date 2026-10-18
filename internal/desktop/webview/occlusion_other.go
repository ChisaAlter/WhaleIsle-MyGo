//go:build !windows && !darwin && !(linux && (amd64 || arm64))

package webview

import "github.com/egoist/mygo"

// Unsupported targets retain their live view without claiming clipping.
func setNativeOcclusions(_ *mygo.Window, _, _ int, _ []mygo.Rectangle) {}
