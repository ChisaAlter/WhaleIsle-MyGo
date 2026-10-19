package ui

import (
	"image"
	"math"
)

// splashSolver is the CPU twin of cursor-fluid.ts's texture passes. Fields use
// bottom-left normalized coordinates and float32 storage (rather than WebGL's
// half-float render targets). Its owner serializes all calls; it starts no work.
type splashSolver struct {
	width, height, simResolution, dyeResolution int
	simW, simH, dyeW, dyeH                      int
	velocity, velocityNext                      [][2]float32
	dye, dyeNext                                [][3]float32
	pressure, pressureNext, divergence, curl    []float32
}

func newSplashSolver(width, height, simResolution, dyeResolution int) *splashSolver {
	s := &splashSolver{simResolution: max(1, simResolution), dyeResolution: max(1, dyeResolution)}
	s.Resize(width, height)
	return s
}

func splashResolution(width, height, resolution int) (int, int) {
	if width > height {
		return int(math.Round(float64(resolution) * float64(width) / float64(height))), resolution
	}
	return resolution, int(math.Round(float64(resolution) * float64(height) / float64(width)))
}

// Resize keeps the readable dye and velocity textures through bilinear
// resampling. Scratch, curl, divergence and pressure restart as in the source.
func (s *splashSolver) Resize(width, height int) {
	width, height = max(1, width), max(1, height)
	if width == s.width && height == s.height {
		return
	}
	sw, sh := splashResolution(width, height, s.simResolution)
	dw, dh := splashResolution(width, height, s.dyeResolution)
	if sw != s.simW || sh != s.simH {
		next := make([][2]float32, sw*sh)
		if len(s.velocity) != 0 {
			for y := 0; y < sh; y++ {
				for x := 0; x < sw; x++ {
					next[y*sw+x] = splashSample2(s.velocity, s.simW, s.simH, (float32(x)+.5)/float32(sw), (float32(y)+.5)/float32(sh))
				}
			}
		}
		s.velocity, s.velocityNext = next, make([][2]float32, sw*sh)
	}
	if dw != s.dyeW || dh != s.dyeH {
		next := make([][3]float32, dw*dh)
		if len(s.dye) != 0 {
			for y := 0; y < dh; y++ {
				for x := 0; x < dw; x++ {
					next[y*dw+x] = splashSample3(s.dye, s.dyeW, s.dyeH, (float32(x)+.5)/float32(dw), (float32(y)+.5)/float32(dh))
				}
			}
		}
		s.dye, s.dyeNext = next, make([][3]float32, dw*dh)
	}
	s.width, s.height, s.simW, s.simH, s.dyeW, s.dyeH = width, height, sw, sh, dw, dh
	s.pressure, s.pressureNext = make([]float32, sw*sh), make([]float32, sw*sh)
	s.divergence, s.curl = make([]float32, sw*sh), make([]float32, sw*sh)
}

// Splat accepts already force-scaled velocity and intensity-scaled RGB.
// Size is the source's 100-based preference, not a radius in pixels.
func (s *splashSolver) Splat(x, y, dx, dy float32, color [3]float32, size float32) {
	if size <= 0 {
		return
	}
	aspect := float32(s.width) / float32(s.height)
	radius := .2 * (size / 100) / 100
	if aspect > 1 {
		radius *= aspect
	}
	for row := 0; row < s.simH; row++ {
		py := (float32(row)+.5)/float32(s.simH) - y
		for col := 0; col < s.simW; col++ {
			px := ((float32(col)+.5)/float32(s.simW) - x) * aspect
			weight := float32(math.Exp(float64(-(px*px + py*py) / radius)))
			i := row*s.simW + col
			s.velocity[i][0] += dx * weight
			s.velocity[i][1] += dy * weight
			if s.simW == s.dyeW && s.simH == s.dyeH {
				s.dye[i][0] += color[0] * weight
				s.dye[i][1] += color[1] * weight
				s.dye[i][2] += color[2] * weight
			}
		}
	}
	if s.simW == s.dyeW && s.simH == s.dyeH {
		return
	}
	for row := 0; row < s.dyeH; row++ {
		py := (float32(row)+.5)/float32(s.dyeH) - y
		for col := 0; col < s.dyeW; col++ {
			px := ((float32(col)+.5)/float32(s.dyeW) - x) * aspect
			weight := float32(math.Exp(float64(-(px*px + py*py) / radius)))
			i := row*s.dyeW + col
			for c := range 3 {
				s.dye[i][c] += color[c] * weight
			}
		}
	}
}

// Step follows curl, confinement, divergence, pressure decay, 20 Jacobi passes,
// gradient subtraction, then velocity and dye advection. Speed changes only
// density dissipation here; input force and color cycling belong to the owner.
func (s *splashSolver) Step(dt, speed float32) {
	dt = min(max(dt, 0), .016666)
	s.computeCurl()
	s.confine(dt)
	s.computeDivergence()
	s.project()
	s.advect(dt, speed)
}

func (s *splashSolver) computeCurl() {
	w, h := s.simW, s.simH
	for y := 0; y < h; y++ {
		b, t := max(y-1, 0)*w, min(y+1, h-1)*w
		for x := 0; x < w; x++ {
			l, r := y*w+max(x-1, 0), y*w+min(x+1, w-1)
			s.curl[y*w+x] = .5 * (s.velocity[r][1] - s.velocity[l][1] - s.velocity[t+x][0] + s.velocity[b+x][0])
		}
	}
}

func (s *splashSolver) confine(dt float32) {
	w, h := s.simW, s.simH
	for y := 0; y < h; y++ {
		b, t := max(y-1, 0)*w, min(y+1, h-1)*w
		for x := 0; x < w; x++ {
			i, l, r := y*w+x, y*w+max(x-1, 0), y*w+min(x+1, w-1)
			fx := .5 * (splashAbs(s.curl[t+x]) - splashAbs(s.curl[b+x]))
			fy := .5 * (splashAbs(s.curl[r]) - splashAbs(s.curl[l]))
			scale := 3 * s.curl[i] / (float32(math.Sqrt(float64(fx*fx+fy*fy))) + .0001)
			s.velocityNext[i] = [2]float32{min(max(s.velocity[i][0]+fx*scale*dt, -1000), 1000), min(max(s.velocity[i][1]-fy*scale*dt, -1000), 1000)}
		}
	}
	s.velocity, s.velocityNext = s.velocityNext, s.velocity
}

func (s *splashSolver) computeDivergence() {
	w, h := s.simW, s.simH
	for y := 0; y < h; y++ {
		b, t := max(y-1, 0)*w, min(y+1, h-1)*w
		for x := 0; x < w; x++ {
			i := y*w + x
			l, r := s.velocity[y*w+max(x-1, 0)][0], s.velocity[y*w+min(x+1, w-1)][0]
			bv, tv := s.velocity[b+x][1], s.velocity[t+x][1]
			if x == 0 {
				l = -s.velocity[i][0]
			}
			if x == w-1 {
				r = -s.velocity[i][0]
			}
			if y == 0 {
				bv = -s.velocity[i][1]
			}
			if y == h-1 {
				tv = -s.velocity[i][1]
			}
			s.divergence[i] = .5 * (r - l + tv - bv)
		}
	}
}

func (s *splashSolver) project() {
	w, h := s.simW, s.simH
	for i := range s.pressure {
		s.pressure[i] *= .1
	}
	for range 20 {
		for y := 0; y < h; y++ {
			b, t := max(y-1, 0)*w, min(y+1, h-1)*w
			for x := 0; x < w; x++ {
				i := y*w + x
				s.pressureNext[i] = (s.pressure[y*w+max(x-1, 0)] + s.pressure[y*w+min(x+1, w-1)] + s.pressure[b+x] + s.pressure[t+x] - s.divergence[i]) * .25
			}
		}
		s.pressure, s.pressureNext = s.pressureNext, s.pressure
	}
	for y := 0; y < h; y++ {
		b, t := max(y-1, 0)*w, min(y+1, h-1)*w
		for x := 0; x < w; x++ {
			i := y*w + x
			s.velocity[i][0] -= s.pressure[y*w+min(x+1, w-1)] - s.pressure[y*w+max(x-1, 0)]
			s.velocity[i][1] -= s.pressure[t+x] - s.pressure[b+x]
		}
	}
}

func (s *splashSolver) advect(dt, speed float32) {
	w, h := s.simW, s.simH
	for y := 0; y < h; y++ {
		v := (float32(y) + .5) / float32(h)
		for x := 0; x < w; x++ {
			i := y*w + x
			velocity := s.velocity[i]
			value := splashSample2(s.velocity, w, h, (float32(x)+.5)/float32(w)-dt*velocity[0]/float32(w), v-dt*velocity[1]/float32(h))
			s.velocityNext[i] = [2]float32{value[0] / (1 + 2*dt), value[1] / (1 + 2*dt)}
		}
	}
	s.velocity, s.velocityNext = s.velocityNext, s.velocity
	decay := 1 + 3.5*(speed/100)*dt
	for y := 0; y < s.dyeH; y++ {
		v := (float32(y) + .5) / float32(s.dyeH)
		for x := 0; x < s.dyeW; x++ {
			u := (float32(x) + .5) / float32(s.dyeW)
			velocity := splashSample2(s.velocity, w, h, u, v)
			value := splashSample3(s.dye, s.dyeW, s.dyeH, u-dt*velocity[0]/float32(w), v-dt*velocity[1]/float32(h))
			for c := range 3 {
				value[c] /= decay
			}
			s.dyeNext[y*s.dyeW+x] = value
		}
	}
	s.dye, s.dyeNext = s.dyeNext, s.dye
}

// Render writes premultiplied pixels, with source alpha=max(R,G,B). The source
// shader already produces premultiplied color, so RGB must not be multiplied
// by alpha again. Output rows are top-down, unlike simulation rows.
func (s *splashSolver) Render(dst *image.RGBA, shading bool) {
	// Bounded native output uses exactly one output pixel per dye texel.
	// Its display neighbors lie within the immediately adjacent texels, so
	// the same linear sampler reduces to a center/neighbor interpolation.
	fx, fy := float32(s.dyeW)/float32(s.width), float32(s.dyeH)/float32(s.height)
	if dst.Rect.Dx() != s.dyeW || dst.Rect.Dy() != s.dyeH || fx > 1 || fy > 1 {
		s.renderSampled(dst, shading)
		return
	}
	w, h := s.dyeW, s.dyeH
	tx, ty := 1/float32(s.width), 1/float32(s.height)
	nz := float32(math.Sqrt(float64(tx*tx + ty*ty)))
	for y := 0; y < h; y++ {
		row := h - 1 - y
		b, t := max(row-1, 0)*w, min(row+1, h-1)*w
		for x := 0; x < w; x++ {
			c := s.dye[row*w+x]
			if shading {
				l, r := s.dye[row*w+max(x-1, 0)], s.dye[row*w+min(x+1, w-1)]
				bc, tc := s.dye[b+x], s.dye[t+x]
				dx := splashLength(splashMix3(c, r, fx)) - splashLength(splashMix3(c, l, fx))
				dy := splashLength(splashMix3(c, tc, fy)) - splashLength(splashMix3(c, bc, fy))
				diffuse := min(max(nz/float32(math.Sqrt(float64(dx*dx+dy*dy+nz*nz)))+.7, .7), 1)
				c[0] *= diffuse
				c[1] *= diffuse
				c[2] *= diffuse
			}
			i := y*dst.Stride + x*4
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2] = splashByte(c[0]), splashByte(c[1]), splashByte(c[2])
			dst.Pix[i+3] = max(dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2])
		}
	}
}

func splashMix3(a, b [3]float32, f float32) [3]float32 {
	return [3]float32{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f, a[2] + (b[2]-a[2])*f}
}

func (s *splashSolver) renderSampled(dst *image.RGBA, shading bool) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	tx, ty := 1/float32(s.width), 1/float32(s.height)
	nz := float32(math.Sqrt(float64(tx*tx + ty*ty)))
	for y := 0; y < h; y++ {
		v := 1 - (float32(y)+.5)/float32(h)
		for x := 0; x < w; x++ {
			u := (float32(x) + .5) / float32(w)
			c := splashSample3(s.dye, s.dyeW, s.dyeH, u, v)
			if shading {
				dx := splashLength(splashSample3(s.dye, s.dyeW, s.dyeH, u+tx, v)) - splashLength(splashSample3(s.dye, s.dyeW, s.dyeH, u-tx, v))
				dy := splashLength(splashSample3(s.dye, s.dyeW, s.dyeH, u, v+ty)) - splashLength(splashSample3(s.dye, s.dyeW, s.dyeH, u, v-ty))
				diffuse := min(max(nz/float32(math.Sqrt(float64(dx*dx+dy*dy+nz*nz)))+.7, .7), 1)
				for channel := range 3 {
					c[channel] *= diffuse
				}
			}
			i := y*dst.Stride + x*4
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2] = splashByte(c[0]), splashByte(c[1]), splashByte(c[2])
			dst.Pix[i+3] = max(dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2])
		}
	}
}

func (s *splashSolver) ImageSize() (int, int) { return s.dyeW, s.dyeH }

// MemoryBytes reports retained numerical buffers, excluding caller-owned output.
func (s *splashSolver) MemoryBytes() int {
	return (cap(s.velocity)+cap(s.velocityNext))*8 + (cap(s.dye)+cap(s.dyeNext))*12 + (cap(s.pressure)+cap(s.pressureNext)+cap(s.divergence)+cap(s.curl))*4
}

func splashAbs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
func splashLength(v [3]float32) float32 {
	return float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
}
func splashByte(v float32) uint8 { return uint8(min(max(v, 0), 1)*255 + .5) }

// Texture sampling matches GL_LINEAR with CLAMP_TO_EDGE and texel centers.
func splashBilerp(w, h int, u, v float32) (a, b, c, d int, fx, fy float32) {
	x, y := min(max(u*float32(w)-.5, 0), float32(w-1)), min(max(v*float32(h)-.5, 0), float32(h-1))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, w-1), min(y0+1, h-1)
	return y0*w + x0, y0*w + x1, y1*w + x0, y1*w + x1, x - float32(x0), y - float32(y0)
}
func splashSample2(field [][2]float32, w, h int, u, v float32) (out [2]float32) {
	a, b, c, d, fx, fy := splashBilerp(w, h, u, v)
	for ch := range 2 {
		ab := field[a][ch] + (field[b][ch]-field[a][ch])*fx
		cd := field[c][ch] + (field[d][ch]-field[c][ch])*fx
		out[ch] = ab + (cd-ab)*fy
	}
	return
}
func splashSample3(field [][3]float32, w, h int, u, v float32) (out [3]float32) {
	a, b, c, d, fx, fy := splashBilerp(w, h, u, v)
	for ch := range 3 {
		ab := field[a][ch] + (field[b][ch]-field[a][ch])*fx
		cd := field[c][ch] + (field[d][ch]-field[c][ch])*fx
		out[ch] = ab + (cd-ab)*fy
	}
	return
}
