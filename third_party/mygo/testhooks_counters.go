package mygo

import "sync/atomic"

// Counters bridging the Invalidate -> contentChanged -> Changed chain for QA
// drivers that cannot reach the internal packages.
var testInvalidateCalls, testInvalidateDropped, testContentChanged, testConnChanged atomic.Int64

func TestInvalidateCounters() (calls, dropped, changed, connChanged int64) {
	return testInvalidateCalls.Load(), testInvalidateDropped.Load(), testContentChanged.Load(), testConnChanged.Load()
}
