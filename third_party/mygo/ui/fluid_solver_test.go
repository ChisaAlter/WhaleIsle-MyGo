package ui

import (
	"bytes"
	"image"
	"math"
	"math/rand/v2"
	"testing"
)

func TestSplashSolverQuiescent(t *testing.T) {
	s := newSplashSolver(160, 100, 16, 24)
	for range 12 {
		s.Step(1.0/60, 100)
	}
	for _, v := range s.velocity {
		if v != [2]float32{} {
			t.Fatalf("quiescent velocity %v", v)
		}
	}
	for _, v := range s.dye {
		if v != [3]float32{} {
			t.Fatalf("quiescent dye %v", v)
		}
	}
	for _, shade := range []bool{false, true} {
		img := image.NewRGBA(image.Rect(0, 0, 48, 30))
		s.Render(img, shade)
		if !bytes.Equal(img.Pix, make([]byte, len(img.Pix))) {
			t.Fatal("quiescent render is not transparent")
		}
	}
}

func TestSplashSolverShaderPasses(t *testing.T) {
	t.Run("curl and reflected divergence", func(t *testing.T) {
		s := newSplashSolver(5, 5, 5, 5)
		for y := range 5 {
			for x := range 5 {
				s.velocity[y*5+x] = [2]float32{float32(x), float32(2*y + x)}
			}
		}
		s.computeCurl()
		s.computeDivergence()
		if s.curl[12] != 1 {
			t.Fatalf("curl center=%v, want 1", s.curl[12])
		}
		if s.divergence[12] != 3 || s.divergence[10] != 2.5 || s.divergence[14] != -1.5 {
			t.Fatalf("divergence center,left,right=%v,%v,%v", s.divergence[12], s.divergence[10], s.divergence[14])
		}
	})
	t.Run("confinement sign and clamp", func(t *testing.T) {
		s := newSplashSolver(3, 3, 3, 3)
		s.curl[4] = 2
		s.curl[7] = 3
		s.curl[5] = 4
		s.velocity[0] = [2]float32{1200, -1200}
		s.confine(.01)
		fx, fy := float32(1.5), float32(2)
		scale := float32(6) / (float32(2.5) + .0001)
		if math.Abs(float64(s.velocity[4][0]-fx*scale*.01)) > 1e-7 || math.Abs(float64(s.velocity[4][1]+fy*scale*.01)) > 1e-7 {
			t.Fatalf("confinement=%v", s.velocity[4])
		}
		if s.velocity[0] != [2]float32{1000, -1000} {
			t.Fatalf("clamp=%v", s.velocity[0])
		}
	})
	t.Run("pressure decay and twenty iterations", func(t *testing.T) {
		s := newSplashSolver(1, 1, 1, 1)
		s.pressure[0] = 10
		s.divergence[0] = 4
		s.project()
		// A one-texel clamp-to-edge Jacobi field loses divergence/4 each pass.
		if s.pressure[0] != -19 {
			t.Fatalf("pressure=%v, want .1*10 - 20", s.pressure[0])
		}
	})
	t.Run("rational dissipation and speed", func(t *testing.T) {
		for _, speed := range []float32{0, 50, 100, 200} {
			s := newSplashSolver(8, 8, 8, 8)
			for i := range s.dye {
				s.dye[i] = [3]float32{.6, .3, .15}
			}
			s.Step(.01, speed)
			want := float32(.6) / (1 + 3.5*(speed/100)*.01)
			if math.Abs(float64(s.dye[27][0]-want)) > 1e-7 {
				t.Fatalf("speed %v dye=%v want=%v", speed, s.dye[27][0], want)
			}
		}
		s := newSplashSolver(8, 8, 8, 8)
		for i := range s.velocity {
			s.velocity[i] = [2]float32{2, -4}
		}
		s.advect(.01, 100)
		if math.Abs(float64(s.velocity[27][0]-2/float32(1.02))) > 1e-6 {
			t.Fatalf("velocity dissipation=%v", s.velocity[27])
		}
	})
	t.Run("projection reduces divergence", func(t *testing.T) {
		s := newSplashSolver(64, 64, 32, 32)
		s.Splat(.5, .5, 80, 30, [3]float32{}, 100)
		s.computeDivergence()
		before := splashTestEnergy(s.divergence)
		s.project()
		s.computeDivergence()
		after := splashTestEnergy(s.divergence)
		if after >= before*.7 {
			t.Fatalf("projection energy before=%g after=%g", before, after)
		}
	})
}

func TestSplashSolverSplatResizeAndDisplay(t *testing.T) {
	s := newSplashSolver(200, 100, 5, 5)
	s.Splat(.55, .5, 12, -7, [3]float32{.6, .3, .15}, 100)
	// A texel center receives exp(0)=1, including the source aspect correction.
	if s.velocity[2*10+5] != [2]float32{12, -7} || s.dye[2*10+5] != [3]float32{.6, .3, .15} {
		t.Fatal("center splat lost force/color")
	}
	weight := float32(math.Exp(-float64(float32(.2) * float32(.2) / float32(.004))))
	if math.Abs(float64(s.dye[2*10+6][0]-.6*weight)) > 1e-6 {
		t.Fatalf("aspect Gaussian=%v", s.dye[26])
	}
	oldMemory := s.MemoryBytes()
	s.Resize(100, 200)
	if w, h := s.ImageSize(); w != 5 || h != 10 {
		t.Fatalf("portrait grid %dx%d", w, h)
	}
	if s.MemoryBytes() != oldMemory {
		t.Fatal("same area retains different buffer footprint")
	}
	for i := range s.pressure {
		if s.pressure[i] != 0 {
			t.Fatal("resize did not reset pressure")
		}
	}
	// Independent 2x2 field has a known center after resize to 3x2.
	r := newSplashSolver(2, 2, 2, 2)
	r.dye = [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0}}
	r.velocity = [][2]float32{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	r.Resize(3, 2)
	if r.dye[1] != [3]float32{.5, 0, 0} || r.velocity[1] != [2]float32{.5, 0} {
		t.Fatalf("bilinear preservation dye=%v velocity=%v", r.dye[1], r.velocity[1])
	}
	img := image.NewRGBA(image.Rect(7, 9, 10, 11))
	r.Render(img, false)
	if got := img.RGBAAt(7, 9); got.G != 255 || got.A != 255 {
		t.Fatalf("bottom-left to top-left conversion: %v", got)
	}
	r.dye[0] = [3]float32{.6, .3, .15}
	r.Render(img, false)
	if got := img.RGBAAt(7, 10); got.R != 153 || got.G != 77 || got.B != 38 || got.A != 153 {
		t.Fatalf("premultiplied color was altered: %v", got)
	}
}

func TestSplashSolverSeededFiniteAndDeterministic(t *testing.T) {
	a, b := newSplashSolver(160, 100, 16, 24), newSplashSolver(160, 100, 16, 24)
	rng := rand.New(rand.NewPCG(19, 2026))
	for frame := range 120 {
		x, y := rng.Float32(), rng.Float32()
		dx, dy := (rng.Float32()-.5)*12000, (rng.Float32()-.5)*12000
		color := [3]float32{rng.Float32() * .15, rng.Float32() * .15, rng.Float32() * .15}
		for _, s := range []*splashSolver{a, b} {
			if frame%3 == 0 {
				s.Splat(x, y, dx, dy, color, 100)
			}
			s.Step(1.0/60, 100)
		}
	}
	for i, v := range a.velocity {
		for _, c := range v {
			if math.IsNaN(float64(c)) || math.IsInf(float64(c), 0) {
				t.Fatalf("nonfinite velocity %d %v", i, v)
			}
		}
		if v != b.velocity[i] {
			t.Fatal("nondeterministic velocity")
		}
	}
	for i, v := range a.dye {
		for _, c := range v {
			if math.IsNaN(float64(c)) || math.IsInf(float64(c), 0) || c < 0 {
				t.Fatalf("invalid dye %d %v", i, v)
			}
		}
		if v != b.dye[i] {
			t.Fatal("nondeterministic dye")
		}
	}
	for _, shade := range []bool{false, true} {
		img := image.NewRGBA(image.Rect(0, 0, 48, 30))
		a.Render(img, shade)
		for i := 0; i < len(img.Pix); i += 4 {
			if img.Pix[i] > img.Pix[i+3] || img.Pix[i+1] > img.Pix[i+3] || img.Pix[i+2] > img.Pix[i+3] {
				t.Fatal("render is not premultiplied")
			}
		}
	}
}

func splashTestEnergy(values []float32) (sum float64) {
	for _, v := range values {
		sum += float64(v) * float64(v)
	}
	return
}

func TestSplashSolverCenteredRenderMatchesSampler(t *testing.T) {
	rng := rand.New(rand.NewPCG(27, 2026))
	maxDifference := 0
	for _, dims := range [][3]int{{1440, 900, 64}, {901, 1441, 96}, {64, 64, 64}, {101, 63, 64}} {
		s := newSplashSolver(dims[0], dims[1], 16, dims[2])
		for i := range s.dye {
			s.dye[i] = [3]float32{rng.Float32() * 1.5, rng.Float32() * 1.5, rng.Float32() * 1.5}
		}
		w, h := s.ImageSize()
		for _, shade := range []bool{false, true} {
			actual, reference := image.NewRGBA(image.Rect(0, 0, w, h)), image.NewRGBA(image.Rect(0, 0, w, h))
			s.Render(actual, shade)
			s.renderSampled(reference, shade)
			for i, a := range actual.Pix {
				difference := int(a) - int(reference.Pix[i])
				if difference < 0 {
					difference = -difference
				}
				maxDifference = max(maxDifference, difference)
				if difference > 1 {
					t.Fatalf("%v shading %v pixel byte %d got=%d want=%d", dims, shade, i, a, reference.Pix[i])
				}
			}
		}
	}
	t.Logf("max centered/generic RGBA byte difference=%d (tolerance 1/255)", maxDifference)
}

func BenchmarkSplashSolver(b *testing.B) {
	for _, config := range []struct {
		name     string
		sim, dye int
	}{
		{"Bounded64_64", 64, 64}, {"Bounded64_96", 64, 96}, {"Bounded128_64", 128, 64}, {"Source128_1440", 128, 1440},
	} {
		b.Run(config.name, func(b *testing.B) {
			s := newSplashSolver(1440, 900, config.sim, config.dye)
			w, h := s.ImageSize()
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			s.Splat(.5, .5, 60, -30, [3]float32{.15, .05, .1}, 100)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				s.Step(1.0/60, 100)
				s.Render(img, true)
			}
			b.StopTimer()
			b.ReportMetric(float64(s.MemoryBytes()), "solver-B")
			b.ReportMetric(float64(len(img.Pix)), "image-B")
		})
	}
}

func BenchmarkSplashSolverActiveFrame64(b *testing.B) {
	s := newSplashSolver(1440, 900, 64, 64)
	w, h := s.ImageSize()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		x := .1 + float32(i%80)*.01
		s.Splat(x, .5, 60, -30, [3]float32{.15, .05, .1}, 100)
		s.Step(1.0/60, 100)
		s.Render(img, true)
	}
	b.StopTimer()
	b.ReportMetric(float64(s.MemoryBytes()), "solver-B")
	b.ReportMetric(float64(len(img.Pix)), "image-B")
}
