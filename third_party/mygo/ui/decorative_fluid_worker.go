package ui

import (
	"image"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

// Non-Windows retains the existing CPU profile. Windows uses persistent
// half-float D3D11 textures at source 128/1440 resolution, on this worker.
const nativeSplashSimResolution = 64
const nativeSplashDyeResolution = 64
const splashFrameInterval = time.Second / 60
const splashIdleTail = 4 * time.Second
const nativeSplashMemoryBudget = 4 << 20

// Include numerical fields and both worker/UI pixel buffers. An extreme
// aspect must not allocate unbounded memory or silently reduce resolution.
func cpuSplashFits(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	aspect := max(float64(width)/float64(height), float64(height)/float64(width))
	simCells := float64(nativeSplashSimResolution) * math.Round(float64(nativeSplashSimResolution)*aspect)
	dyeCells := float64(nativeSplashDyeResolution) * math.Round(float64(nativeSplashDyeResolution)*aspect)
	return 32*simCells+32*dyeCells <= nativeSplashMemoryBudget
}

// Target at most24% of one core while active, with at most60 jobs per second.
// This native processing budget does not claim source frame/pixel parity.
func cpuSplashPacing(work time.Duration) time.Duration {
	return max(splashFrameInterval, work*100/24)
}

type splashRequest struct {
	presses        []splashPress
	now            time.Time
	width, height  int
	config         DecorativeEffects
	x, y, dx, dy   float32
	moved, press   bool
	pressX, pressY float32
}

type splashPress struct {
	x, y   float32
	config DecorativeEffects
}

type splashBackend interface {
	Resize(int, int) error
	Splat(float32, float32, float32, float32, [3]float32, float32)
	Step(float32, float32)
	ImageSize() (int, int)
	Render(*image.RGBA) error
	Close()
}

// Native frames contain an owned GPU lease, never mutable scene pixels.
// The optional backend is used by Windows; the CPU profile keeps its bitmap
// handoff. A nil frame with no error means the GPU handoff is not ready yet.
type splashNativeFrame interface {
	ImageSize() (int, int)
	Close()
}

type splashNativeBackend interface {
	RenderNative() (splashNativeFrame, error)
}

type splashNativePresenter interface {
	paint(*Painter, splashNativeFrame, float32, float32) (bool, error)
	close()
}

// The owner submits one job only after taking the previous result. Output is
// worker-owned until take. The UI copies CPU output into its retained bitmap,
// or imports a native GPU lease before the next submission. No simulation
// buffer, immediate context or mutable scene crosses threads.
type splashWorker struct {
	jobs      chan splashRequest
	results   chan *image.RGBA
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	// Published with the result, read before the owner submits another job.
	lastWork      time.Duration
	err           error
	nativeResults chan splashNativeFrame
	// Only the UI owner touches pending/presenter. The worker owns results
	// until received, and drains abandoned leases when it stops.
	nativePending splashNativeFrame
	presenter     splashNativePresenter
}

func newSplashWorker(seed uint64) *splashWorker {
	w := &splashWorker{jobs: make(chan splashRequest, 1), results: make(chan *image.RGBA, 1), nativeResults: make(chan splashNativeFrame, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go w.run(seed)
	return w
}

func (w *splashWorker) close() {
	w.closeOnce.Do(func() {
		close(w.stop)
		if w.nativePending != nil {
			w.nativePending.Close()
			w.nativePending = nil
		}
		if w.presenter != nil {
			w.presenter.close()
			w.presenter = nil
		}
	})
}

func (w *splashWorker) run(seed uint64) {
	defer close(w.done)
	defer func() {
		for {
			select {
			case frame := <-w.nativeResults:
				frame.Close()
			default:
				return
			}
		}
	}()
	random := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var solver splashBackend
	defer func() {
		if solver != nil {
			solver.Close()
		}
	}()
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
		started := time.Now()
		if solver == nil {
			var err error
			solver, err = newSplashBackend()
			if err != nil {
				w.fail(err)
				return
			}
			color = splashColor(random, job.config)
		}
		if err := solver.Resize(job.width, job.height); err != nil {
			w.fail(err)
			return
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
		presses := job.presses
		if len(presses) == 0 && job.press {
			presses = []splashPress{{job.pressX, job.pressY, job.config}}
		}
		for _, press := range presses {
			bright := splashColor(random, press.config)
			for i := range bright {
				bright[i] *= 10
			}
			solver.Splat(press.x, press.y, 10*(random.Float32()-.5), 30*(random.Float32()-.5), bright, press.config.Size)
		}
		if job.moved {
			force := 6000 * job.config.Speed / 100
			solver.Splat(job.x, job.y, job.dx*force, job.dy*force, color, job.config.Size)
		}
		solver.Step(dt, job.config.Speed)
		if native, ok := solver.(splashNativeBackend); ok {
			var frame splashNativeFrame
			for frame == nil {
				var err error
				frame, err = native.RenderNative()
				if err != nil {
					w.fail(err)
					return
				}
				if frame == nil {
					// Zero-timeout mutex acquisition never stalls the UI or a
					// context. Wait cancelably for another display opportunity;
					// no input or simulation step is repeated by this poll.
					timer := time.NewTimer(splashFrameInterval)
					select {
					case <-w.stop:
						timer.Stop()
						return
					case <-timer.C:
					}
				}
			}
			w.lastWork = time.Since(started)
			select {
			case <-w.stop:
				frame.Close()
				return
			case w.nativeResults <- frame:
			}
			continue
		}
		width, height := solver.ImageSize()
		if output == nil || output.Rect.Dx() != width || output.Rect.Dy() != height {
			output = image.NewRGBA(image.Rect(0, 0, width, height))
		}
		if err := solver.Render(output); err != nil {
			w.fail(err)
			return
		}
		w.lastWork = time.Since(started)
		select {
		case <-w.stop:
			return
		case w.results <- output: // At most one result; the UI never waits.
		}
	}
}

func (w *splashWorker) fail(err error) {
	w.err = err
	select {
	case <-w.stop:
	case w.results <- nil:
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
	if s.fluidError != nil {
		return
	} // source failed GPU mount is inert until remounted
	width, height := max(1, int(w*p.scale)), max(1, int(h*p.scale))
	if !nativeSplashFits(width, height) {
		s.clearPointer() // Pause/release this decoration, preserving preferences.
		return
	}
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
		if result == nil {
			s.fluidError = s.fluidWorker.err
			s.fluidBitmap = nil
			s.fluidBusy = false
			s.fluidWorker.close()
			s.fluidWorker = nil
			return
		}
		if s.fluidBitmap == nil || s.fluidBitmap.w != result.Rect.Dx() || s.fluidBitmap.h != result.Rect.Dy() {
			s.fluidBitmap = NewBitmap(image.NewRGBA(result.Rect))
		}
		copy(s.fluidBitmap.img.Pix, result.Pix)
		s.fluidBitmap.img.Changed()
		s.fluidBusy = false
		if paced := s.fluidSubmitted.Add(splashPacing(s.fluidWorker.lastWork)); paced.After(s.fluidNext) {
			s.fluidNext = paced
		}
	default:
	}
	worker := s.fluidWorker
	if worker.nativePending == nil {
		select {
		case worker.nativePending = <-worker.nativeResults:
		default:
		}
	}
	if worker.nativePending != nil || worker.presenter != nil {
		var err error
		if worker.presenter == nil {
			worker.presenter, err = newNativeSplashPresenter(p)
		}
		var ready bool
		if err == nil && worker.presenter != nil {
			ready, err = worker.presenter.paint(p, worker.nativePending, w, h)
		}
		if err != nil {
			s.fluidError, s.fluidBusy = err, false
			worker.close()
			s.fluidWorker, s.fluidBitmap = nil, nil
			return
		}
		if ready && worker.nativePending != nil {
			worker.nativePending.Close()
			worker.nativePending = nil
			s.fluidBusy = false
		}
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
		job.now, job.width, job.height, job.config = now, width, height, s.config
		s.fluidWorker.jobs <- job
		s.fluidBusy = true
		s.fluidSubmitted = now
		s.fluidInput.moved, s.fluidInput.press = false, false
		s.fluidInput.presses = nil // worker owns the submitted event slice
		s.fluidNext = now.Add(splashFrameInterval)
	}
	// Keep the engine's paint-only route: invalidating from the worker would
	// rebuild the entire view every frame. Native next() parks this while hidden.
	if !s.fluidBusy && s.fluidNext.Sub(now) > splashFrameInterval {
		p.After(s.fluidNext.Sub(now))
	} else {
		p.AnimationFrame()
	}
}
