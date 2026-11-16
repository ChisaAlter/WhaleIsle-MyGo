//go:build !release

package mygo

import "sync/atomic"

// Counters bridging the Invalidate -> contentChanged -> Changed chain for QA
// drivers that cannot reach the internal packages. Disabled in production until
// enabled explicitly or read by a QA driver so release hot paths do zero writes.
var (
	testCountersEnabled                                                             atomic.Bool
	testInvalidateCalls, testInvalidateDropped, testContentChanged, testConnChanged atomic.Int64
)

// EnableTestCounters controls whether Invalidate/contentChanged increment QA
// diagnostic counters.
func EnableTestCounters(enabled bool) { testCountersEnabled.Store(enabled) }

func recordTestCounter(counter *atomic.Int64) {
	if testCountersEnabled.Load() {
		counter.Add(1)
	}
}

func recordTestInvalidateCall()    { recordTestCounter(&testInvalidateCalls) }
func recordTestInvalidateDropped() { recordTestCounter(&testInvalidateDropped) }
func recordTestContentChanged()    { recordTestCounter(&testContentChanged) }
func recordTestConnChanged()       { recordTestCounter(&testConnChanged) }

func TestInvalidateCounters() (calls, dropped, changed, connChanged int64) {
	testCountersEnabled.Store(true)
	return testInvalidateCalls.Load(), testInvalidateDropped.Load(), testContentChanged.Load(), testConnChanged.Load()
}
