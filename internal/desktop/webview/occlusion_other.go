//go:build !windows && !(linux && (amd64 || arm64))

package webview

import "github.com/egoist/mygo"

// macOS currently uses a separate NSWindow for this live browser. A CALayer
// mask only changes rendering, not cross-window pointer targeting; toggling
// ignoresMouseEvents would incorrectly disable the entire browser. Proper
// parity requires backend-level hosting/input work, not a visual-only mask.
// Unsupported targets therefore retain their live view without claiming
// floating-pane occlusion support. Linux/GDK support lives in occlusion_linux.go.
func setNativeOcclusions(_ *mygo.Window, _, _ int, _ []mygo.Rectangle) {}
