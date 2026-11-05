package voiceinput

import (
	"context"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/desktop/voicecapture"
)

type liveFixture struct {
	done    chan struct{}
	once    sync.Once
	level   float64
	failure error
}

func (f *liveFixture) Level() float64        { return f.level }
func (f *liveFixture) Done() <-chan struct{} { return f.done }
func (f *liveFixture) Cancel()               { f.once.Do(func() { close(f.done) }) }
func (f *liveFixture) Stop() ([]byte, error) { f.Cancel(); return []byte{1, 2}, f.failure }
func TestLiveMeterAndNativeCompletion(t *testing.T) {
	capture := &liveFixture{done: make(chan struct{}), level: .1}
	transcribed := make(chan struct{})
	var once sync.Once
	c := New(Options{Prepare: func(context.Context) (voicecapture.Limits, Transcribe, error) {
		return voicecapture.Limits{MaxDuration: time.Minute}, func(context.Context, []byte) (string, error) {
			once.Do(func() { close(transcribed) })
			return "words", nil
		}, nil
	}, Capture: func(context.Context, voicecapture.Limits) (voicecapture.Capture, error) { return capture, nil }, Dispatch: func(fn func()) { fn() }, Insert: func(string) bool { return true }})
	c.Start()
	deadline := time.Now().Add(time.Second)
	for c.Snapshot().Phase != "recording" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s := c.Snapshot(); s.Phase != "recording" || s.Level < .66 || s.Level > .67 {
		t.Fatalf("real level %+v", s)
	}
	capture.Cancel()
	select {
	case <-transcribed:
	case <-time.After(time.Second):
		t.Fatal("native completion waited for stop timer")
	}
	c.Cancel()
}
