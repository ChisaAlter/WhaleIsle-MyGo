package composer

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBeamGaussianMatchesIndependentTwoDimensionalConvolution(t *testing.T) {
	const width, height = 11, 9
	for _, sigma := range []float64{.7, 3, 8} {
		for _, solid := range []bool{false, true} {
			b := beamRasterData{width: width, height: height, bloom: make([]beamRGBA, width*height), scratch: make([]beamRGBA, width*height), blurSteps: make([]beamRGBA, width)}
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					if solid {
						b.bloom[y*width+x] = beamRGBA{.1, .2, .3, .5}
					} else if x == 0 && y == 0 || x == 5 && y == 4 {
						b.bloom[y*width+x] = beamRGBA{.2, .1, .3, .5}
					}
				}
			}
			original := append([]beamRGBA(nil), b.bloom...)
			b.blur(sigma)
			// Evaluate the definition as a full 2D sum, independently of the
			// production sparse/CDF passes, including source at viewport edges.
			radius := int(math.Ceil(3 * sigma))
			normalization := 0.0
			for k := -radius; k <= radius; k++ {
				normalization += math.Exp(-float64(k*k) / (2 * sigma * sigma))
			}
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					var want [4]float64
					for sy := 0; sy < height; sy++ {
						for sx := 0; sx < width; sx++ {
							dx, dy := x-sx, y-sy
							if dx < -radius || dx > radius || dy < -radius || dy > radius {
								continue
							}
							weight := math.Exp(-float64(dx*dx+dy*dy)/(2*sigma*sigma)) / (normalization * normalization)
							for c, v := range original[sy*width+sx] {
								want[c] += float64(v) * weight
							}
						}
					}
					for c, v := range b.bloom[y*width+x] {
						if math.Abs(float64(v)-want[c]) > 2e-6 {
							t.Fatalf("sigma%v solid%v pixel%d,%d channel%d: got%v want%v", sigma, solid, x, y, c, v, want[c])
						}
					}
				}
			}
			kernel := &b.gaussian.weights[0]
			b.gaussian.prepare(sigma)
			if &b.gaussian.weights[0] != kernel {
				t.Fatal("unchanged sigma rebuilt Gaussian kernel")
			}
		}
	}
}

func TestBeamInsetShadowIncludesBothEdgesOfNarrowCard(t *testing.T) {
	b := beamRasterData{w: 80, h: 20, scale: 1, width: 88, height: 28}
	b.geometry = make([]beamPixel, b.width*b.height)
	b.bloom = make([]beamRGBA, len(b.geometry))
	b.scratch = make([]beamRGBA, len(b.geometry))
	b.blurSteps = make([]beamRGBA, b.width)
	b.insetShadow(ui.RGB(255, 255, 255), 1, 3, 21)
	// A 14px-high hole occupies only about a quarter of a sigma21 Gaussian.
	// Both edges contribute at its center; a nearest-edge erfc gave ~.37.
	if alpha := b.geometry[14*b.width+44].wash[3]; alpha < .72 || alpha > 1 {
		t.Fatalf("inset shadow lost its other edges: %v", alpha)
	}
}

func TestBeamBloomBlursBeforeOuterFrameClip(t *testing.T) {
	style := readBeamStyle(nil)
	colors := []ui.Color{ui.RGB(255, 50, 100), ui.RGB(40, 140, 255)}
	var b beamRasterData
	style.GlowBlur = 0
	b.render(80, 40, 1, style, colors, 2, 0, 0, false, beamOpacities{0, 0, 1})
	if a := b.pixels.RGBAAt(1, 24).A; a != 0 {
		t.Fatal("zero-blur source escaped the card ring", a)
	}
	style.GlowBlur = 8
	b.render(80, 40, 1, style, colors, 2, 0, 0, false, beamOpacities{0, 0, 1})
	if a := b.pixels.RGBAAt(1, 24).A; a == 0 {
		t.Fatal("bloom halo was clipped to the card before blur")
	}
	if a := b.pixels.RGBAAt(0, 0).A; a != 0 {
		t.Fatal("blur escaped the radius26 outer frame clip", a)
	}
}

func TestBeamRoundedCoverageUsesPixelAreaAndEmptyInnerBoxes(t *testing.T) {
	if got := beamRoundedCoverage(6.5, 6.5, 80, 60, 22, 1); got != .625 {
		t.Fatalf("rounded-corner subpixel area changed: %v", got)
	}
	if beamRoundedCoverage(0, 0, -1, 20, 0, 1) != 0 {
		t.Fatal("collapsed inset hole retained coverage")
	}
	if a, b := beamRoundedCoverage(5.2, 7.4, 60, 60, 22, 1.5), beamRoundedCoverage(7.4, 5.2, 60, 60, 22, 1.5); a != b {
		t.Fatal("DPI coverage lost diagonal symmetry", a, b)
	}
}
