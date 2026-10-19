package ui

import (
	"bytes"
	"image"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

func awaitSplash(t *testing.T, worker *splashWorker) *image.RGBA {
	t.Helper()
	select {
	case image := <-worker.results:
		return image
	case <-time.After(3 * time.Second):
		t.Fatal("background splash did not complete")
		return nil
	}
}

func splashTestConfig() DecorativeEffects {
	return DecorativeEffects{PointerEnabled: true, PointerEffect: "splash", Speed: 100, Size: 100, ColorCount: 2, Colors: [6]Color{RGB(255, 0, 0), RGB(0, 128, 255)}}
}

func TestSplashNativePaintSchedulingRetuneAndDisposal(t *testing.T) {
	config := splashTestConfig()
	tt := newRepaintTester(func(c *context) {
		SetDecorativeEffects(makeContext(c), config)
		coreBox(c).Fill()
	})
	t.Cleanup(tt.rt.close)
	s := &tt.rt.decorations
	if s.fluidWorker != nil {
		t.Fatal("enabled but unused effect allocated a worker")
	}
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 90, Y: 40})
	tt.frame(17 * time.Millisecond)
	worker := s.fluidWorker
	if worker == nil || !s.fluidBusy || cap(worker.jobs) != 1 || cap(worker.results) != 1 {
		t.Fatal("first movement did not create one bounded background job")
	}
	result := awaitSplash(t, worker)
	worker.results <- result
	if built, more := tt.frame(17 * time.Millisecond); built || !more || s.fluidBitmap == nil {
		t.Fatalf("completion rebuilt the view or stopped active repaint: built=%v more=%v", built, more)
	}
	bitmap := s.fluidBitmap
	result = awaitSplash(t, worker)
	worker.results <- result
	config.Size, config.Speed = 160, 150
	tt.rt.changed()
	tt.frame(17 * time.Millisecond)
	if s.fluidWorker != worker || s.fluidBitmap != bitmap {
		t.Fatal("live retuning cleared fields or replaced the retained scene bitmap")
	}
	result = awaitSplash(t, worker)
	worker.results <- result
	tt.h.w, tt.h.h = 100, 200
	tt.frame(17 * time.Millisecond)
	if s.fluidWorker != worker {
		t.Fatal("resize discarded the solver owner instead of resampling")
	}
	result = awaitSplash(t, worker)
	worker.results <- result
	if built, more := tt.frame(17 * time.Millisecond); built || !more || s.fluidBitmap.w >= s.fluidBitmap.h {
		t.Fatalf("portrait result not consumed without rebuilding: built=%v more=%v image=%dx%d", built, more, s.fluidBitmap.w, s.fluidBitmap.h)
	}
	config.PointerEnabled = false
	tt.rt.changed()
	tt.frame(17 * time.Millisecond)
	if s.fluidWorker != nil || s.fluidBitmap != nil || s.fluidBusy || tt.rt.repainting {
		t.Fatal("disable retained or continued scheduling the effect")
	}
	select {
	case <-worker.done:
	case <-time.After(3 * time.Second):
		t.Fatal("disable failed to release worker")
	}
}

func TestSplashIdleHiddenAndCloseDoNotSchedule(t *testing.T) {
	config := splashTestConfig()
	tt := newRepaintTester(func(c *context) { SetDecorativeEffects(makeContext(c), config) })
	t.Cleanup(tt.rt.close)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 80, Y: 30})
	tt.frame(17 * time.Millisecond)
	s, worker := &tt.rt.decorations, tt.rt.decorations.fluidWorker
	result := awaitSplash(t, worker)
	worker.results <- result
	if _, more := tt.frame(splashIdleTail + time.Millisecond); more || tt.rt.repainting || s.fluidBusy || len(worker.jobs) != 0 || len(worker.results) != 0 {
		t.Fatal("idle tail did not park all frame and job scheduling")
	}
	tt.h.hidden = true
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 90, Y: 30})
	tt.frame(17 * time.Millisecond)
	if len(worker.jobs) != 0 || s.fluidBusy || !tt.rt.held || tt.h.requested.Load() {
		t.Fatal("hidden native host submitted background simulation")
	}
	tt.h.hidden = false
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if !tt.h.requested.Load() {
		t.Fatal("visibility restoration failed to resume held idle-tail animation")
	}
	tt.frame(17 * time.Millisecond)
	if !s.fluidBusy {
		t.Fatal("visible native host failed to resume one bounded job")
	}
	tt.rt.close()
	worker.close() // Closing an already closed owner is harmless.
	select {
	case <-worker.done:
	case <-time.After(3 * time.Second):
		t.Fatal("window destruction retained idle worker")
	}
	if s.fluidWorker != nil || s.fluidBitmap != nil || len(worker.jobs) != 0 {
		t.Fatal("closed engine retained splash resources")
	}
}

func TestSplashWorkerSeededInputsAndBoundedCoalescing(t *testing.T) {
	a, b := newSplashWorker(2026), newSplashWorker(2026)
	t.Cleanup(a.close)
	t.Cleanup(b.close)
	job := splashRequest{now: time.Unix(1000, 0), width: 160, height: 100, config: splashTestConfig(), x: .5, y: .5, dx: .02, dy: -.03, moved: true, press: true, pressX: .4, pressY: .6}
	for frame := range 12 {
		job.now = job.now.Add(17 * time.Millisecond)
		job.x += .005
		job.press = frame == 0
		a.jobs <- job
		b.jobs <- job
		if !bytes.Equal(awaitSplash(t, a).Pix, awaitSplash(t, b).Pix) {
			t.Fatal("same seed and input sequence changed numerical output")
		}
	}
	s := decorationState{config: splashTestConfig(), width: 200, height: 100}
	for i := 0; i < 10000; i++ {
		s.queueSplash(float32(i%200), 25, 2, 3, job.now)
	}
	if s.fluidInput.x != float32(199)/200 || s.fluidInput.dx != .01 || s.fluidInput.dy != -.015 || !s.fluidInput.moved || s.fluidWorker != nil {
		t.Fatalf("coalesced normalized input is incorrect: %+v", s.fluidInput)
	}
	s.noteSplashPointer(100, 50, job.now)
	s.noteSplashPointer(104, 48, job.now)
	if diff := s.fluidInput.dx - .02; diff < -.000001 || diff > .000001 {
		t.Fatalf("native input accumulated frame distance instead of latest event delta: %+v", s.fluidInput)
	}
}

func TestSplashNativeDPIUsesPhysicalDisplayTexels(t *testing.T) {
	tt := NewTester(func(c *Context) {}, 1440, 900)
	defer tt.rt.close()
	now := time.Unix(1000, 0)
	s := &tt.rt.decorations
	s.config, s.width, s.height = splashTestConfig(), 1440, 900
	s.fluidWorker = &splashWorker{jobs: make(chan splashRequest, 1), results: make(chan *image.RGBA, 1), stop: make(chan struct{})}
	s.fluidLastInput = now
	p := &tt.rt.painter
	p.scale = 2
	s.paintSplash(p, 1440, 900, now)
	job := <-s.fluidWorker.jobs
	if job.width != 2880 || job.height != 1800 || s.width != 1440 || s.height != 900 {
		t.Fatalf("display texels did not apply DPI independently of input: job=%dx%d input=%vx%v", job.width, job.height, s.width, s.height)
	}
}

func TestPointerTrailRetuneRandomPaletteAndRoundedSpacing(t *testing.T) {
	config := splashTestConfig()
	config.PointerEffect, config.Size = "pixel", 103
	tt := NewTester(func(c *Context) { SetDecorativeEffects(c, config) }, 200, 100)
	t.Cleanup(tt.rt.close)
	s := &tt.rt.decorations
	s.random = rand.New(rand.NewPCG(19, 2026))
	var colors []Color
	for range 16 {
		colors = append(colors, s.color())
	}
	var differs bool
	for i := 2; i < len(colors); i++ {
		differs = differs || colors[i] == colors[i-1]
	}
	if !differs {
		t.Fatal("trail still deterministically alternates palette slots")
	}
	s.feed(57.4, 50, 37.4, 0, time.Now())
	want := decorationState{config: config}
	want.stampTrail(57.4, 50, time.Now())
	if len(s.cells) != len(want.cells) {
		t.Fatal("interpolation used unrounded24.72px instead of source25px spacing")
	}
	for cell := range want.cells {
		if _, ok := s.cells[cell]; !ok {
			t.Fatal("rounded-cell endpoint stamp differs")
		}
	}
	s.stampTrail(50, 50, time.Now())
	before := len(s.cells)
	config.Size = 150
	tt.Frame()
	if before == 0 || len(s.cells) != before {
		t.Fatal("live size update discarded source trail cells")
	}
	config.PointerEnabled = false
	tt.Frame()
	if len(s.cells) != 0 {
		t.Fatal("disable retained trail cells")
	}
}

func BenchmarkSplashPaintHandoff(b *testing.B) {
	tt := NewTester(func(c *Context) {}, 1440, 900)
	defer tt.rt.close()
	now := time.Now()
	s := &tt.rt.decorations
	s.fluidWorker = &splashWorker{results: make(chan *image.RGBA, 1), stop: make(chan struct{})}
	s.fluidBusy, s.fluidLastInput = true, now
	s.fluidBitmap = NewBitmap(image.NewRGBA(image.Rect(0, 0, 102, 64)))
	s.fluidNext = now.Add(time.Second)
	result := image.NewRGBA(image.Rect(0, 0, 102, 64))
	p := &tt.rt.painter
	p.rt, p.s, p.scale, p.opacity, p.clip = tt.rt, &tt.rt.scene, 1, 1, Rect{W: 1440, H: 900}
	for _, copyResult := range []bool{false, true} {
		name := "Waiting"
		if copyResult {
			name = "CopyResult"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if copyResult {
					s.fluidWorker.results <- result
				}
				tt.rt.scene.Reset(1440, 900, tt.rt.scene.Clear)
				s.paintSplash(p, 1440, 900, now)
			}
		})
	}
}

// Includes channel handoff and worker scheduling, unlike the solver-only mean.
// This is diagnostic data; hardware-dependent latency is not a unit assertion.
func BenchmarkSplashWorkerFrameDistribution(b *testing.B) {
	worker := newSplashWorker(2026)
	defer worker.close()
	job := splashRequest{now: time.Unix(1000, 0), width: 1440, height: 900, config: splashTestConfig(), x: .5, y: .5, dx: .02, dy: -.03, moved: true}
	worker.jobs <- job
	<-worker.results // Allocate and warm all retained numerical/output buffers.
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		job.now = job.now.Add(splashFrameInterval)
		start := time.Now()
		worker.jobs <- job
		<-worker.results
		samples[i] = time.Since(start)
	}
	b.StopTimer()
	slices.Sort(samples)
	b.ReportMetric(float64(samples[(len(samples)-1)*50/100].Nanoseconds()), "p50-ns")
	b.ReportMetric(float64(samples[(len(samples)-1)*95/100].Nanoseconds()), "p95-ns")
	b.ReportMetric(float64(samples[len(samples)-1].Nanoseconds()), "max-ns")
}
