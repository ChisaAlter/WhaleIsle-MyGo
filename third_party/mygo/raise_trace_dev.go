//go:build !release

package mygo

import "runtime/debug"

// RaiseTrace, when set, receives the name of every call that tries to raise a
// window and the stack that made it. A development build installs it to find out
// which surface raises a window a launch was not allowed to raise. The release
// build replaces traceRaise with a no-op, so no shipping path pays for the check.
var RaiseTrace func(call string, inactiveOnly bool, stack []byte)

// RaiseTraceNative receives the same reports from the platform layer, below the
// window API this package exposes. The class fix lives in Window, so a raise
// recorded here is one that fix never sees.
var RaiseTraceNative func(call string, stack []byte)

func traceRaise(call string, inactiveOnly bool) {
	if RaiseTrace == nil {
		return
	}
	RaiseTrace(call, inactiveOnly, debug.Stack())
}
