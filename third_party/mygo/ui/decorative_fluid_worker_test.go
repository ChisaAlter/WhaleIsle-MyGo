package ui

import (
	"image"
	"math/rand/v2"
	"testing"
	"time"
)

func splashTestConfig() DecorativeEffects {
	return DecorativeEffects{PointerEnabled: true, PointerEffect: "splash", Speed: 100, Size: 100, ColorCount: 2, Colors: [6]Color{RGB(255, 0, 0), RGB(0, 128, 255)}}
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
