package pluginui

import (
	"context"
	"testing"
	"time"
)

func TestMirrorCaptureLeaseExpiresWaitWithoutReleasingRealCapture(t *testing.T) {
	done := make(chan struct{})
	state := &mirrorState{inflight: true, captureDone: done}
	owner := &Owner{mirror: state}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if owner.awaitMirrorCapture(ctx, state) == nil {
		t.Fatal("cancelled worker waited forever")
	}
	select {
	case <-done:
		t.Fatal("context expiry released real native capture lease")
	default:
	}
	if !state.inflight {
		t.Fatal("context expiry cleared inflight capture")
	}
	close(done)
	next, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := owner.awaitMirrorCapture(next, state); err != nil {
		t.Fatal(err)
	}
	owner.mirror = &mirrorState{}
	if owner.awaitMirrorCapture(next, state) == nil {
		t.Fatal("old source lease acquired a new owner")
	}
}
