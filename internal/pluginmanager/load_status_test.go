package pluginmanager

import (
	"context"
	"testing"
)

func TestLoadCaptureFailureLeavesARetryableErrorNotLoading(t *testing.T) {
	c := &Controller{}
	if err := c.Load(context.Background()); err == nil {
		t.Fatal("load without a captured client succeeded")
	}
	state := c.Snapshot()
	if state.Loading {
		t.Fatal("capture failure left the page loading")
	}
	if state.Status != "error" || state.Error == "" {
		t.Fatalf("capture failure state=%q error=%q, want a retryable error", state.Status, state.Error)
	}
}
