//go:build !windows

package ui

// readThreadCycles returns no cycles and no thread where the system does not
// count the cycles a thread uses: the trace then keeps the monotonic clock's own
// resolution.
func readThreadCycles() (uint64, uint32) { return 0, 0 }
