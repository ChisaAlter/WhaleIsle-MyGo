package composer

import "math"

// A CSS blur length is the Gaussian standard deviation in CSS pixels. Keep a
// single normalized kernel at device scale; neither the kernel nor its CDF is
// rebuilt while an animated frame only changes angle/hue/opacity.
type beamGaussian struct {
	sigma   float64
	radius  int
	weights []float32
	prefix  []float32
}

func (g *beamGaussian) prepare(sigma float64) {
	if g.sigma == sigma && len(g.weights) != 0 {
		return
	}
	g.sigma, g.radius = sigma, int(math.Ceil(3*sigma))
	g.weights = make([]float32, 2*g.radius+1)
	g.prefix = make([]float32, len(g.weights))
	var total float64
	for i := -g.radius; i <= g.radius; i++ {
		total += math.Exp(-float64(i*i) / (2 * sigma * sigma))
	}
	var cumulative float64
	for i := -g.radius; i <= g.radius; i++ {
		weight := math.Exp(-float64(i*i)/(2*sigma*sigma)) / total
		g.weights[i+g.radius] = float32(weight)
		cumulative += weight
		g.prefix[i+g.radius] = float32(cumulative)
	}
}

// Each pass reads the complete source support before the final rounded clip.
// The four-pixel frame is an output viewport, not an intermediate blur clip:
// unlike repeated box passes, no halo must leave it and re-enter a later pass.
func (b *beamRasterData) blur(sigma float64) {
	if sigma <= 0 {
		return
	}
	b.gaussian.prepare(sigma)
	for _, vertical := range []bool{false, true} {
		beamGaussianPass(b.bloom, b.scratch, b.width, b.height, vertical, &b.gaussian, b.blurSteps)
		b.bloom, b.scratch = b.scratch, b.bloom
	}
}

// Convolve the same Gaussian either as sparse samples or as first differences
// times its CDF. The latter is exact discrete convolution as well: long solid
// runs in an inset-shadow mask cost only their two transitions, rather than a
// kernel per pixel. Both forms use transparent pixels outside the source.
func beamGaussianPass(source, target []beamRGBA, width, height int, vertical bool, g *beamGaussian, steps []beamRGBA) {
	lines, length, stride := height, width, 1
	if vertical {
		lines, length, stride = width, height, width
	}
	for line := 0; line < lines; line++ {
		base := line * width
		if vertical {
			base = line
		}
		var previous beamRGBA
		nonzero, changes := 0, 0
		for pos := 0; pos < length; pos++ {
			value := source[base+pos*stride]
			if value != (beamRGBA{}) {
				nonzero++
			}
			if value != previous {
				changes++
			}
			previous = value
			target[base+pos*stride] = beamRGBA{}
		}
		if previous != (beamRGBA{}) {
			changes++
		}
		if nonzero == 0 {
			continue
		}
		if changes >= nonzero {
			for pos := 0; pos < length; pos++ {
				value := source[base+pos*stride]
				if value == (beamRGBA{}) {
					continue
				}
				for out := max(0, pos-g.radius); out <= min(length-1, pos+g.radius); out++ {
					weight := g.weights[out-pos+g.radius]
					index := base + out*stride
					for channel := range value {
						target[index][channel] += value[channel] * weight
					}
				}
			}
			continue
		}
		clear(steps[:length])
		previous = beamRGBA{}
		for pos := 0; pos <= length; pos++ {
			var value, delta beamRGBA
			if pos < length {
				value = source[base+pos*stride]
			}
			for channel := range delta {
				delta[channel] = value[channel] - previous[channel]
			}
			previous = value
			if delta == (beamRGBA{}) {
				continue
			}
			for out := max(0, pos-g.radius); out <= min(length-1, pos+g.radius-1); out++ {
				weight := g.prefix[out-pos+g.radius]
				index := base + out*stride
				for channel := range delta {
					target[index][channel] += delta[channel] * weight
				}
			}
			if start := pos + g.radius; start < length {
				for channel := range delta {
					steps[start][channel] += delta[channel]
				}
			}
		}
		var constant beamRGBA
		for pos := 0; pos < length; pos++ {
			for channel := range constant {
				constant[channel] += steps[pos][channel]
				index := base + pos*stride
				// Cancelled CDF differences can leave negative rounding dust.
				target[index][channel] = max(0, target[index][channel]+constant[channel])
			}
		}
	}
}
