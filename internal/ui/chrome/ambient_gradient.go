package chrome

import (
	"context"
	"image"
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	// A decorative backdrop is sampled in logical screen space, never in device
	// pixels. These bounds apply equally to 100%, 150%, zoomed and 4K windows.
	gradientRasterEdge     = 128
	gradientPaddedEdge     = 224
	gradientMaxSigma       = 8.0
	gradientFrameInterval  = time.Second / 30
	gradientCacheByteLimit = 4 << 20
)

type gradientPlan struct {
	width, height, padX, padY, fullWidth, fullHeight int
	scaleX, scaleY, sigmaX, sigmaY                   float64
}

func planGradient(c gradientConfig) (gradientPlan, bool) {
	if !gradientFinite(c.width) || !gradientFinite(c.height) || c.width <= 0 || c.height <= 0 {
		return gradientPlan{}, false
	}
	sigma := c.blur()
	scale := math.Min(1, float64(gradientRasterEdge)/math.Max(c.width, c.height))
	scale = math.Min(scale, gradientMaxSigma/sigma)
	// Include transparent filter input outside the clipping viewport. Blur is
	// applied to the isolated blob group before it is clipped by #dsh-gradient.
	scale = math.Min(scale, float64(gradientPaddedEdge-4)/(math.Max(c.width, c.height)+6*sigma))
	w, h := max(1, int(math.Floor(c.width*scale))), max(1, int(math.Floor(c.height*scale)))
	sx, sy := float64(w)/c.width, float64(h)/c.height
	if sigma*sx > gradientMaxSigma+1e-9 || sigma*sy > gradientMaxSigma+1e-9 {
		// Very tiny dimensions cannot represent this CSS blur without exceeding
		// the fixed kernel budget. No unbounded kernel is allowed on that path.
		return gradientPlan{}, false
	}
	px, py := int(math.Ceil(3*sigma*sx)), int(math.Ceil(3*sigma*sy))
	p := gradientPlan{w, h, px, py, w + 2*px, h + 2*py, sx, sy, sigma * sx, sigma * sy}
	if p.fullWidth > gradientPaddedEdge || p.fullHeight > gradientPaddedEdge {
		return gradientPlan{}, false
	}
	return p, true
}

type gradientFrame struct {
	pixels *image.RGBA // already masked, the exact bitmap painted and sampled
	bitmap *ui.Bitmap
	config gradientConfig
	tick   int64
}

func gradientTick(now time.Time, epoch time.Time, reduced bool) int64 {
	if reduced {
		return -1
	}
	elapsed := now.Sub(epoch)
	if elapsed < 0 {
		return 0
	}
	return int64(elapsed / gradientFrameInterval)
}

// The cache retains one immutable result, not an unbounded phase/size map.
// New configurations revoke the prior generation. Rendering checks cancellation
// each scanline and before publishing; no goroutine or timer survives a source
// switch. The first and subsequent cache misses do bounded work synchronously.
type gradientFrameCache struct {
	mu         sync.Mutex
	config     gradientConfig
	configured bool
	generation uint64
	cancel     context.CancelFunc
	epoch      time.Time
	frame      *gradientFrame
	renders    uint64
	// rendering is set while latest has a phase advance in flight on a worker
	// goroutine, so the painting thread never waits for it.
	rendering bool
}

func (s *gradientFrameCache) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.rendering = false
	s.configured = false
	s.frame = nil
}
func (s *gradientFrameCache) get(c gradientConfig, now time.Time) *gradientFrame {
	if _, ok := planGradient(c); !ok {
		return nil
	}
	s.mu.Lock()
	if !s.configured || s.config != c {
		if s.cancel != nil {
			s.cancel()
		}
		s.generation++
		s.rendering = false
		s.configured = true
		s.config = c
		s.epoch = now
		s.frame = nil
	}
	tick := gradientTick(now, s.epoch, c.reduced)
	if s.frame != nil && s.frame.tick == tick {
		f := s.frame
		s.mu.Unlock()
		return f
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.generation++
	s.rendering = false
	generation := s.generation
	s.mu.Unlock()
	seconds := float64(max(0, tick)) * gradientFrameInterval.Seconds()
	pixels := renderGradient(ctx, c, seconds)
	if pixels == nil {
		return nil
	}
	frame := &gradientFrame{pixels: pixels, bitmap: ui.NewBitmap(pixels), config: c, tick: tick}
	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation || ctx.Err() != nil {
		return nil
	}
	s.renders++
	s.frame = frame
	return frame
}

// latest is get for the painting thread. The first frame of a configuration is
// still rendered synchronously, so something is always shown. Advancing the
// phase of an existing frame costs 6-12 ms of CPU, which is most of a 60 Hz
// frame: it renders on a worker goroutine instead and the painter keeps the
// previous immutable frame until the next paint picks up the finished one (at
// most one gradientFrameInterval later, which a slow ambient drift cannot show).
// At most one such render is in flight, and a reset, a configuration change or
// a synchronous get revokes it through the generation check.
func (s *gradientFrameCache) latest(c gradientConfig, now time.Time) *gradientFrame {
	if _, ok := planGradient(c); !ok {
		return nil
	}
	s.mu.Lock()
	if !s.configured || s.config != c || s.frame == nil {
		s.mu.Unlock()
		return s.get(c, now)
	}
	frame := s.frame
	tick := gradientTick(now, s.epoch, c.reduced)
	if frame.tick == tick || s.rendering {
		s.mu.Unlock()
		return frame
	}
	ctx, cancel := context.WithCancel(context.Background())
	if s.cancel != nil {
		s.cancel()
	}
	s.cancel = cancel
	s.rendering = true
	generation := s.generation
	s.mu.Unlock()
	go func() {
		seconds := float64(max(0, tick)) * gradientFrameInterval.Seconds()
		pixels := renderGradient(ctx, c, seconds)
		var next *gradientFrame
		if pixels != nil {
			next = &gradientFrame{pixels: pixels, bitmap: ui.NewBitmap(pixels), config: c, tick: tick}
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if generation != s.generation {
			// A reset, configuration change or synchronous get took over; it
			// already cleared rendering and owns the cache now.
			return
		}
		s.rendering = false
		if next == nil || ctx.Err() != nil {
			return
		}
		s.renders++
		s.frame = next
	}()
	return frame
}

var ambientGradientFrames gradientFrameCache

type gradientFloatPair struct {
	source    []float32
	temporary []float32
}

var gradientFloatBufPool = sync.Pool{
	New: func() any {
		return &gradientFloatPair{}
	},
}

func renderGradient(ctx context.Context, c gradientConfig, seconds float64) *image.RGBA {
	p, ok := planGradient(c)
	if !ok || ctx.Err() != nil {
		return nil
	}
	n := p.fullWidth * p.fullHeight * 4
	// Two premultiplied float buffers plus one RGBA output and its immutable
	// ui.Bitmap copy stay below the byte ceiling, even during a cache replacement.
	if n*8+p.width*p.height*16 > gradientCacheByteLimit {
		return nil
	}
	pair := gradientFloatBufPool.Get().(*gradientFloatPair)
	if cap(pair.source) < n {
		pair.source = make([]float32, n)
	} else {
		pair.source = pair.source[:n]
	}
	if cap(pair.temporary) < n {
		pair.temporary = make([]float32, n)
	} else {
		pair.temporary = pair.temporary[:n]
	}
	defer gradientFloatBufPool.Put(pair)
	source := pair.source
	temporary := pair.temporary
	blobs := c.blobs(seconds)
	var prepared [5]preparedGradientBlob
	for i := range blobs {
		prepared[i] = prepareGradientBlob(blobs[i])
	}
	for y := 0; y < p.fullHeight; y++ {
		if ctx.Err() != nil {
			return nil
		}
		cy := (float64(y-p.padY) + .5) / p.scaleY
		for x := 0; x < p.fullWidth; x++ {
			cx := (float64(x-p.padX) + .5) / p.scaleX
			pixel := [4]float32{}
			for i := 0; i < c.count; i++ {
				pixel = gradientBlend(pixel, prepared[i].sample(cx, cy))
			}
			pos := (y*p.fullWidth + x) * 4
			copy(source[pos:pos+4], pixel[:])
		}
	}
	if !gradientGaussian(ctx, source, temporary, p.fullWidth, p.fullHeight, p.sigmaX, p.sigmaY) {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, p.width, p.height))
	keep := 1 - float64(61)/255
	if c.dark {
		keep = 1 - float64(128)/255
	}
	for y := 0; y < p.height; y++ {
		if ctx.Err() != nil {
			return nil
		}
		for x := 0; x < p.width; x++ {
			base := c.baseline((float64(x)+.5)/p.scaleX, (float64(y)+.5)/p.scaleY)
			pos := ((y+p.padY)*p.fullWidth + x + p.padX) * 4
			a := float64(source[pos+3])
			dst := (y*p.width + x) * 4
			for channel, value := range [3]float64{base.r, base.g, base.b} {
				v := (float64(source[pos+channel]) + (1-a)*value) * keep
				out.Pix[dst+channel] = uint8(math.Round(math.Min(1, math.Max(0, v)) * 255))
			}
			out.Pix[dst+3] = 255
		}
	}
	return out
}
func gradientKernel(sigma float64) []float32 {
	if sigma <= .01 {
		return []float32{1}
	}
	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float32, radius*2+1)
	sum := 0.0
	for i := range kernel {
		d := float64(i - radius)
		v := math.Exp(-d * d / (2 * sigma * sigma))
		kernel[i] = float32(v)
		sum += v
	}
	for i := range kernel {
		kernel[i] /= float32(sum)
	}
	return kernel
}

// SVG/CSS blur operates on premultiplied group pixels. Transparent extension
// (not edge clamping) is important where a moving bloom crosses the viewport.
func gradientGaussian(ctx context.Context, source, temp []float32, w, h int, sigmaX, sigmaY float64) bool {
	kx, ky := gradientKernel(sigmaX), gradientKernel(sigmaY)
	rx, ry, stride, colStride := len(kx)/2, len(ky)/2, w*4, h*4
	for y := 0; y < h; y++ {
		if ctx.Err() != nil {
			return false
		}
		row := source[y*stride : (y+1)*stride]
		for x := 0; x < w; x++ {
			var r, g, b, a float32
			lo, hi := max(0, rx-x), min(len(kx), w+rx-x)
			pos := (x + lo - rx) * 4
			for k := lo; k < hi; k++ {
				weight := kx[k]
				pixel := row[pos : pos+4]
				r += pixel[0] * weight
				g += pixel[1] * weight
				b += pixel[2] * weight
				a += pixel[3] * weight
				pos += 4
			}
			dstOff := x*colStride + y*4
			dst := temp[dstOff : dstOff+4]
			dst[0], dst[1], dst[2], dst[3] = r, g, b, a
		}
	}
	for y := 0; y < h; y++ {
		if ctx.Err() != nil {
			return false
		}
		lo, hi := max(0, ry-y), min(len(ky), h+ry-y)
		startOff := (y + lo - ry) * 4
		for x := 0; x < w; x++ {
			var r, g, b, a float32
			col := temp[x*colStride : (x+1)*colStride]
			pos := startOff
			for k := lo; k < hi; k++ {
				weight := ky[k]
				pixel := col[pos : pos+4]
				r += pixel[0] * weight
				g += pixel[1] * weight
				b += pixel[2] * weight
				a += pixel[3] * weight
				pos += 4
			}
			dst := source[y*stride+x*4 : y*stride+x*4+4]
			dst[0], dst[1], dst[2], dst[3] = r, g, b, a
		}
	}
	return true
}
