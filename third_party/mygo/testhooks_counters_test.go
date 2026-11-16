//go:build !release

package mygo

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestInvalidateCountersGatedUntilEnabled(t *testing.T) {
	EnableTestCounters(false)
	t.Cleanup(func() { EnableTestCounters(false) })

	before := testInvalidateCalls.Load()
	w, _, _ := contentWindow(t, func(c *ui.Context) {})
	w.Invalidate()
	onMain(func() {})
	if got := testInvalidateCalls.Load(); got != before {
		t.Fatalf("Invalidate mutated counter while disabled: got %d, want %d", got, before)
	}

	callsBefore, _, _, _ := TestInvalidateCounters()
	w.Invalidate()
	onMain(func() {})
	callsAfter, _, _, _ := TestInvalidateCounters()
	if callsAfter <= callsBefore {
		t.Fatalf("Invalidate counters did not record after TestInvalidateCounters: %d -> %d", callsBefore, callsAfter)
	}
}
