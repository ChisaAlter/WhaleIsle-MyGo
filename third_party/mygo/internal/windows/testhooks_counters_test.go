//go:build windows && (amd64 || arm64) && !release

package windows

import "testing"

func TestFrameCountersAndWebViewFailureGate(t *testing.T) {
	EnableTestFrameCounters(false)
	TestFailWebViews(0)
	t.Cleanup(func() {
		EnableTestFrameCounters(false)
		TestFailWebViews(0)
	})

	beforeReq := testFrameRequests.Load()
	s := &surface{}
	s.RequestFrame()
	if got := testFrameRequests.Load(); got != beforeReq {
		t.Fatalf("RequestFrame incremented counter while disabled: got %d, want %d", got, beforeReq)
	}

	reqBefore, _ := TestFrameCounters()
	s.RequestFrame()
	reqAfter, _ := TestFrameCounters()
	if reqAfter != reqBefore+1 {
		t.Fatalf("TestFrameCounters did not increment after activation: %d -> %d", reqBefore, reqAfter)
	}

	if consumeTestWebViewFailure() {
		t.Fatal("consumeTestWebViewFailure returned true at 0")
	}
	TestFailWebViews(2)
	if !consumeTestWebViewFailure() || !consumeTestWebViewFailure() || consumeTestWebViewFailure() {
		t.Fatal("consumeTestWebViewFailure countdown mismatch")
	}
}
