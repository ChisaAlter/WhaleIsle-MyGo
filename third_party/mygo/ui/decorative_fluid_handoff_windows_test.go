//go:build windows

package ui

import (
	"errors"
	"testing"
	"time"
)

// These lifecycle tests use the existing frame/presenter seam and never
// construct a D3D device, window, or input event. Real sharing has a separate,
// explicitly opted-in test in internal/gpu/d3d11.
type handoffTestFrame struct {
	width, height int
	closes        int
}

func (f *handoffTestFrame) ImageSize() (int, int) { return f.width, f.height }
func (f *handoffTestFrame) Close()                { f.closes++ }

type handoffTestPresenter struct {
	ready        bool
	err          error
	closes       int
	seen         []splashNativeFrame
	copiedWidth  int
	copiedHeight int
}

func (p *handoffTestPresenter) paint(_ *Painter, frame splashNativeFrame, _, _ float32) (bool, error) {
	p.seen = append(p.seen, frame)
	if p.err != nil {
		return false, p.err
	}
	if frame != nil && p.ready {
		p.copiedWidth, p.copiedHeight = frame.ImageSize()
		return true, nil
	}
	return false, nil
}
func (p *handoffTestPresenter) close() { p.closes++ }

func TestSplashNativePendingHoldsLeaseAndDefersResizedJob(t *testing.T) {
	tt := NewTester(func(*Context) {}, 200, 100)
	t.Cleanup(tt.rt.close)
	now := time.Unix(1000, 0)
	frame := &handoffTestFrame{width: 400, height: 200}
	presenter := &handoffTestPresenter{copiedWidth: 80, copiedHeight: 60}
	worker := &splashWorker{
		jobs: make(chan splashRequest, 1), stop: make(chan struct{}),
		nativeResults: make(chan splashNativeFrame, 1), presenter: presenter,
	}
	worker.nativeResults <- frame
	s := &tt.rt.decorations
	s.config, s.fluidWorker, s.fluidBusy = splashTestConfig(), worker, true
	s.fluidLastInput, s.fluidNext = now, now.Add(splashFrameInterval)
	p := &tt.rt.painter
	p.scale = 2

	s.paintSplash(p, 200, 100, now)
	s.paintSplash(p, 300, 150, now.Add(time.Millisecond))
	if worker.nativePending != frame || frame.closes != 0 || !s.fluidBusy || len(worker.jobs) != 0 || len(worker.nativeResults) != 0 {
		t.Fatal("unready native frame was dropped, closed, or allowed a replacement job")
	}
	if len(presenter.seen) != 2 || presenter.seen[0] != frame || presenter.seen[1] != frame || presenter.copiedWidth != 80 || presenter.copiedHeight != 60 {
		t.Fatal("unready polling replaced the pending lease or previous presentation")
	}

	presenter.ready = true
	s.paintSplash(p, 300, 150, now.Add(2*time.Millisecond))
	if worker.nativePending != nil || frame.closes != 1 || s.fluidBusy || len(worker.jobs) != 0 || presenter.copiedWidth != 400 || presenter.copiedHeight != 200 {
		t.Fatal("completed import did not release exactly its lease before the next frame")
	}
	s.config.Size = 150
	s.paintSplash(p, 300, 150, now.Add(splashFrameInterval))
	if !s.fluidBusy || len(worker.jobs) != 1 || frame.closes != 1 {
		t.Fatal("import completion did not admit exactly one subsequent job")
	}
	job := <-worker.jobs
	if job.width != 600 || job.height != 300 || job.config.Size != 150 {
		t.Fatalf("next job lost current physical size or live settings: %+v", job)
	}
	s.clearPointer()
	worker.close()
	if frame.closes != 1 || presenter.closes != 1 || worker.presenter != nil || s.fluidWorker != nil {
		t.Fatal("disable repeated lease release or retained presentation ownership")
	}
}

func TestSplashNativeCloseDrainsQueuedAndPendingLeases(t *testing.T) {
	pending := &handoffTestFrame{width: 10, height: 10}
	queued := &handoffTestFrame{width: 20, height: 20}
	presenter := &handoffTestPresenter{}
	worker := &splashWorker{
		stop: make(chan struct{}), done: make(chan struct{}),
		nativeResults: make(chan splashNativeFrame, 1), nativePending: pending, presenter: presenter,
	}
	worker.nativeResults <- queued
	worker.close()
	if pending.closes != 1 || queued.closes != 0 || presenter.closes != 1 {
		t.Fatal("UI close crossed queued-result ownership or failed to retire its own lease")
	}
	// A closed stop and no jobs make run return before backend creation. Its
	// real deferred cleanup owns any result published before shutdown won.
	worker.run(1)
	worker.close()
	select {
	case <-worker.done:
	default:
		t.Fatal("stopped worker did not finish")
	}
	if pending.closes != 1 || queued.closes != 1 || presenter.closes != 1 || len(worker.nativeResults) != 0 || worker.nativePending != nil || worker.presenter != nil {
		t.Fatal("shutdown leaked or double-released a native lease")
	}
}

func TestSplashNativePresentationFailureRetiresOwner(t *testing.T) {
	for _, factoryFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "import", true: "no-native-surface"}[factoryFailure], func(t *testing.T) {
			tt := NewTester(func(*Context) {}, 200, 100)
			t.Cleanup(tt.rt.close)
			now := time.Unix(1000, 0)
			frame := &handoffTestFrame{width: 200, height: 100}
			cause := errors.New("presentation owner was lost")
			presenter := &handoffTestPresenter{err: cause}
			worker := &splashWorker{jobs: make(chan splashRequest, 1), stop: make(chan struct{}), nativePending: frame}
			if !factoryFailure {
				worker.presenter = presenter
			}
			s := &tt.rt.decorations
			s.config, s.fluidWorker, s.fluidBusy, s.fluidLastInput = splashTestConfig(), worker, true, now
			s.paintSplash(&tt.rt.painter, 200, 100, now)
			if s.fluidError == nil || !factoryFailure && !errors.Is(s.fluidError, cause) || s.fluidWorker != nil || s.fluidBusy || s.fluidBitmap != nil || frame.closes != 1 || len(worker.jobs) != 0 {
				t.Fatal("presentation failure was reported as a dynamic success or retained its lease")
			}
			select {
			case <-worker.stop:
			default:
				t.Fatal("failed presentation did not stop its simulation owner")
			}
			s.paintSplash(&tt.rt.painter, 200, 100, now.Add(time.Second))
			if s.fluidWorker != nil || frame.closes != 1 || !factoryFailure && presenter.closes != 1 {
				t.Fatal("failed effect silently restarted or repeated resource release")
			}
		})
	}
}

func TestSplashNativeFirstFrameWaitsForRealRendererChoice(t *testing.T) {
	host := &windowHost{}
	p := &Painter{rt: &engine{host: host}}
	presenter, err := newNativeSplashPresenter(p)
	if presenter != nil || err != nil {
		t.Fatal("first decoration paint guessed a renderer before surface presentation")
	}
	host.gpuTried = true
	if presenter, err := newNativeSplashPresenter(p); presenter != nil || err == nil {
		t.Fatal("actual missing renderer was not an explicit failure")
	}
}
