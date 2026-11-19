//go:build windows && (amd64 || arm64) && !release

package mygo

import win "github.com/egoist/mygo/internal/windows"

func init() {
	win.RaiseTrace = func(call string, stack []byte) {
		if RaiseTraceNative == nil {
			return
		}
		RaiseTraceNative(call, stack)
	}
}
