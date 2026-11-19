package ui

import (
	"sync"
	"syscall"
	"unsafe"
)

// Windows counts the cycles a thread uses, which is the only clock fine enough
// for a frame: the monotonic clock advances in steps of about half a millisecond
// here, so a frame that takes 300 µs reads as no time at all. QueryThreadCycleTime
// counts what the calling thread used, so a measurement is only the work of a
// frame while the frame stays on one thread: the caller compares the thread with
// the one it read before and drops the difference when the goroutine moved, as a
// goroutine does in Go.

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getCurrentThread     = kernel32.NewProc("GetCurrentThread")
	getCurrentThreadID   = kernel32.NewProc("GetCurrentThreadId")
	queryThreadCycleTime = kernel32.NewProc("QueryThreadCycleTime")
	cycleOnce            sync.Once
	cycleOK              bool
)

// readThreadCycles returns the cycles the calling thread has used and which
// thread that was, or 0 and 0 where the system does not count cycles.
func readThreadCycles() (uint64, uint32) {
	cycleOnce.Do(func() {
		h, _, _ := getCurrentThread.Call()
		var n uint64
		r, _, _ := queryThreadCycleTime.Call(h, uintptr(unsafe.Pointer(&n)))
		cycleOK = r != 0
	})
	if !cycleOK {
		return 0, 0
	}
	tid, _, _ := getCurrentThreadID.Call()
	h, _, _ := getCurrentThread.Call()
	var n uint64
	if r, _, _ := queryThreadCycleTime.Call(h, uintptr(unsafe.Pointer(&n))); r == 0 {
		return 0, 0
	}
	return n, uint32(tid)
}
