package voicecapture

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// These fixtures never call Start or any native permission/device API.
func TestBoundedCaptureLifecycle(t *testing.T) {
	capacity, duration, err := boundedLimits(Limits{MaxBytes: 48, MaxDuration: time.Minute})
	if err != nil || capacity != 4 || duration != 125*time.Microsecond {
		t.Fatalf("bounds %d %s %v", capacity, duration, err)
	}
	c := newRecording(context.Background())
	go func() { <-c.stop; c.finish([]byte{1, 0, 2, 0}, nil) }()
	wav, err := c.Stop()
	if err != nil || len(wav) != 48 || binary.LittleEndian.Uint32(wav[24:28]) != 16000 {
		t.Fatal("invalid bounded WAV", err)
	}
	if _, err = c.Stop(); !errors.Is(err, ErrStopped) {
		t.Fatal("result consumed twice")
	}
	canceled := newRecording(context.Background())
	go func() { <-canceled.cancel; canceled.finish([]byte{1, 0}, nil) }()
	canceled.Cancel()
	if _, err = canceled.Stop(); !errors.Is(err, ErrCanceled) {
		t.Fatal("cancel retained audio")
	}
}
