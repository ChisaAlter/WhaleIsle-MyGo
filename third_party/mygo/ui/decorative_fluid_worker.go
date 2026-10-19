package ui

import (
	"image"
	"math/rand/v2"
	"sync"
	"time"
)

// The native CPU profile keeps the existing short-edge64 paint budget. The
// source128/1440 profile is supported by splashSolver, but is not silently
// enabled on the UI thread. Both resolution and half-float/GPU parity remain
// explicit limitations; the numerical passes and input rules are shared.
const nativeSplashSimResolution = 64
const nativeSplashDyeResolution = 64
const splashFrameInterval = time.Second / 60
const splashIdleTail = 4 * time.Second

type splashRequest struct {
	now            time.Time
	width, height  int
	config         DecorativeEffects
	x, y, dx, dy   float32
	moved, press   bool
	pressX, pressY float32
}

// The owner submits one job only after taking the previous result. Output is
// worker-owned until take; it is copied into one retained scene bitmap before
// the next submission. No simulation buffer or mutable scene crosses threads.
type splashWorker struct {
	jobs      chan splashRequest
	results   chan *image.RGBA
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func newSplashWorker(seed uint64) *splashWorker {
	w := &splashWorker{jobs: make(chan splashRequest, 1), results: make(chan *image.RGBA, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go w.run(seed)
	return w
}

func (w *splashWorker) close() {
	w.closeOnce.Do(func() { close(w.stop) })
}

func (w *splashWorker) run(seed uint64) {
	defer close(w.done)
	random := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var solver *splashSolver
	var output *image.RGBA
	var last time.Time
	var color [3]float32
	var colorClock float32
	for {
		var job splashRequest
		select {
		case <-w.stop:
			return
		case job = <-w.jobs:
		}
		select {
		case <-w.stop:
			return
		default:
		}
		if solver == nil {
			solver = newSplashSolver(job.width, job.height, nativeSplashSimResolution, nativeSplashDyeResolution)
			color = splashColor(random, job.config)
		} else {
			solver.Resize(job.width, job.height)
		}
		dt := float32(job.now.Sub(last).Seconds())
		if last.IsZero() {
			dt = .016666
		}
		dt = min(max(dt, 0), .016666)
		last = job.now
		colorClock += dt * 10 * job.config.Speed / 100
		if colorClock >= 1 {
			colorClock -= float32(int(colorClock))
			color = splashColor(random, job.config)
		}
		if job.moved {
			force := 6000 * job.config.Speed / 100
			solver.Splat(job.x, job.y, job.dx*force, job.dy*force, color, job.config.Size)
		}
		if job.press {
			bright := splashColor(random, job.config)
			for i := range bright {
				bright[i] *= 10
			}
			solver.Splat(job.pressX, job.pressY, 10*(random.Float32()-.5), 30*(random.Float32()-.5), bright, job.config.Size)
		}
		solver.Step(dt, job.config.Speed)
		width, height := solver.ImageSize()
		if output == nil || output.Rect.Dx() != width || output.Rect.Dy() != height {
			output = image.NewRGBA(image.Rect(0, 0, width, height))
		}
		solver.Render(output, true)
		select {
		case <-w.stop:
			return
		case w.results <- output: // At most one result; the UI never waits.
		}
	}
}

func splashColor(random *rand.Rand, config DecorativeEffects) [3]float32 {
	if config.ColorCount > 0 {
		color := config.Colors[random.IntN(config.ColorCount)]
		return [3]float32{float32(color.R) / 255 * .15, float32(color.G) / 255 * .15, float32(color.B) / 255 * .15}
	}
	// Original empty fluid palette uses a random fully saturated HSV color.
	h := random.Float32() * 6
	sector := int(h)
	f := h - float32(sector)
	colors := [6][3]float32{{1, f, 0}, {1 - f, 1, 0}, {0, 1, f}, {0, 1 - f, 1}, {f, 0, 1}, {1, 0, 1 - f}}
	out := colors[sector]
	for i := range out {
		out[i] *= .15
	}
	return out
}

func (s *decorationState) noteSplashPointer(x, y float32, now time.Time) {
	if !s.config.PointerEnabled || s.config.PointerEffect != "splash" || s.width <= 0 || s.height <= 0 {
		return
	}
	u, v := x/s.width, 1-y/s.height
	// The original keeps the last DOM event delta, not the sum since a paint.
	// Recording here preserves that rule when many events share one frame.
	s.queueSplash(x, y, (u-s.fluidPointerX)*s.width, -(v-s.fluidPointerY)*s.height, now)
	s.fluidPointerX, s.fluidPointerY = u, v
}

func (s *decorationState) queueSplash(x, y, dx, dy float32, now time.Time) {
	if s.width <= 0 || s.height <= 0 {
		return
	}
	s.fluidInput.x, s.fluidInput.y = x/s.width, 1-y/s.height
	s.fluidInput.dx, s.fluidInput.dy = dx/s.width, -dy/s.height
	aspect := s.width / s.height
	if aspect < 1 {
		s.fluidInput.dx *= aspect
	} else {
		s.fluidInput.dy /= aspect
	}
	s.fluidInput.moved = s.fluidInput.dx != 0 || s.fluidInput.dy != 0
	s.fluidLastInput = now
}

func (s *decorationState) paintSplash(p *Painter, w, h float32, now time.Time) {
	if p.rt.host.occluded() {
		if !s.fluidLastInput.IsZero() && now.Sub(s.fluidLastInput) <= splashIdleTail {
			// Let next() mark the animation held, without requesting a hidden
			// frame. SurfaceShown then resumes it just as visibilitychange does.
			p.AnimationFrame()
		}
		return // Native next() also parks repaint scheduling while hidden.
	}
	if s.pendingPress {
		s.fluidInput.press = true
		s.fluidInput.pressX, s.fluidInput.pressY = s.pressX/w, 1-s.pressY/h
		s.fluidLastInput = now
		s.pendingPress = false
	}
	if s.fluidWorker == nil && !s.fluidLastInput.IsZero() {
		s.fluidWorker = newSplashWorker(uint64(now.UnixNano()))
	}
	if s.fluidWorker == nil {
		return
	}
	select {
	case result := <-s.fluidWorker.results:
		if s.fluidBitmap == nil || s.fluidBitmap.w != result.Rect.Dx() || s.fluidBitmap.h != result.Rect.Dy() {
			s.fluidBitmap = NewBitmap(image.NewRGBA(result.Rect))
		}
		copy(s.fluidBitmap.img.Pix, result.Pix)
		s.fluidBitmap.img.Changed()
		s.fluidBusy = false
	default:
	}
	if s.fluidBitmap != nil {
		p.Image(s.fluidBitmap, Rect{W: w, H: h}, FillBox)
	}
	if now.Sub(s.fluidLastInput) > splashIdleTail {
		return // Keep the last source frame, but schedule no further work.
	}
	if !s.fluidBusy && !now.Before(s.fluidNext) {
		job := s.fluidInput
		// Source display shading samples drawing-buffer texels, not CSS/DIP
		// texels. Input normalization stays in DIPs; only this size uses DPI.
		job.now, job.width, job.height, job.config = now, max(1, int(w*p.scale)), max(1, int(h*p.scale)), s.config
		s.fluidWorker.jobs <- job
		s.fluidBusy = true
		s.fluidInput.moved, s.fluidInput.press = false, false
		s.fluidNext = now.Add(splashFrameInterval)
	}
	// Keep the engine's paint-only route: invalidating from the worker would
	// rebuild the entire view every frame. Native next() parks this while hidden.
	p.AnimationFrame()
}
