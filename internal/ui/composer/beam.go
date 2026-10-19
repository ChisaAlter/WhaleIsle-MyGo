package composer

import (
	"encoding/json"
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
)

type beamStyle struct {
	Direction  string  `json:"direction"`
	Period     float64 `json:"period"`
	Intensity  float64 `json:"intensity"`
	Bloom      float64 `json:"bloom"`
	Hue        float64 `json:"hue"`
	TrackWidth float32 `json:"trackWidth"`
	GlowBlur   float32 `json:"glowBlur"`
	Easing     string  `json:"easing"`
	Palette    struct {
		Kind   string   `json:"kind"`
		ID     string   `json:"id"`
		Colors []string `json:"colors"`
	} `json:"palette"`
	Breathing struct {
		Enabled   bool    `json:"enabled"`
		Amplitude float64 `json:"amplitude"`
		Period    float64 `json:"period"`
	} `json:"breathing"`
	HueCycle struct {
		Enabled bool    `json:"enabled"`
		Range   float64 `json:"range"`
		Period  float64 `json:"period"`
	} `json:"hueCycle"`
	Night struct {
		Enabled    bool    `json:"enabled"`
		FromMinute int     `json:"fromMinute"`
		ToMinute   int     `json:"toMinute"`
		Dim        float64 `json:"dim"`
	} `json:"night"`
}

func readBeamStyle(raw json.RawMessage) beamStyle {
	style := beamStyle{Direction: "clockwise", Period: 1.96, Intensity: 100, Bloom: 100, TrackWidth: 2, GlowBlur: 8, Easing: "linear"}
	style.Palette.Kind, style.Palette.ID = "preset", "legacy"
	style.Breathing.Amplitude, style.Breathing.Period = .3, 6
	style.HueCycle.Enabled, style.HueCycle.Range, style.HueCycle.Period = true, 30, 12
	style.Night.FromMinute, style.Night.ToMinute, style.Night.Dim = 1320, 360, 40
	_ = json.Unmarshal(raw, &style)
	return style
}

var beamPalettes = map[string][]string{
	"legacy":  {"#ff3264", "#288cff", "#32c850", "#1eb9aa", "#6446ff", "#ff7828", "#f032b4"},
	"warm":    {"#ff4d6d", "#ff8c42", "#ffd166", "#ef476f", "#ff4d6d"},
	"aurora":  {"#42e8c8", "#3f8cff", "#8a5cf6", "#e45bff", "#42e8c8"},
	"rose":    {"#ff4f81", "#ff79c6", "#c77dff", "#7b61ff", "#ff4f81"},
	"glacier": {"#d9f7ff", "#6edff6", "#328bff", "#5b6dff", "#d9f7ff"},
	"cyber":   {"#00f0ff", "#00ff8f", "#b6ff00", "#ffed00", "#00f0ff"},
	"sunset":  {"#ff5f6d", "#ffc371", "#ff9a8b", "#ff6a88", "#ff5f6d"},
	"cyan":    {"#31d7ff", "#1eb9aa", "#64a8ff", "#31d7ff"},
}

func composerRunning(m Model) bool {
	if m.Snapshot.PromptBusy || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
		return true
	}
	for _, item := range m.Snapshot.Items {
		if item.SessionID == m.Snapshot.SelectedID {
			return item.Running
		}
	}
	return false
}

// Native painter paths keep all beam layers pointer-inert and preserve the
// composer's native text, toolbar and drag-resize event ownership.
func paintComposerBeam(p *ui.Painter, r ui.Rect, style beamStyle, timeline *beamTimeline, active, reduced bool) {
	now := p.Now()
	frame := timeline.frame(now, style, active, reduced)
	if frame.animate {
		p.AnimationFrame()
	}
	if frame.opacity == (beamOpacities{}) {
		return
	}
	phase := 0.0
	hue := 0.0
	// Removing .cardBeam cancels spin/hue/breathe immediately. Its registered
	// angle falls back to 0deg while the child opacity transition fades out.
	if active {
		phase = beamPhase(frame.seconds, style)
		hue = beamHue(frame.hueSeconds, style)
	}
	hueMatrix := beamHueMatrix(hue)
	colors := beamPalettes[style.Palette.ID]
	if style.Palette.Kind == "custom" {
		colors = style.Palette.Colors
	}
	if len(colors) == 0 {
		colors = beamPalettes["legacy"]
	}
	// Normalized custom palettes contain 2..6 colors; presets at most seven.
	// Keep the common paint path on the stack and retain larger raw palettes
	// without allocating a fresh color slice on every animation frame.
	var palette [7]ui.Color
	inks := palette[:]
	if len(colors) > len(palette) {
		if cap(timeline.palette) < len(colors) {
			timeline.palette = make([]ui.Color, len(colors))
		}
		inks = timeline.palette[:len(colors)]
	}
	for i, value := range colors {
		inks[i] = chrome.Color(value)
	}
	// Source closes only open palettes. Already closed presets use n-1
	// intervals; an extra repeated first color would shift every stop.
	intervals := len(colors)
	if colors[0] == colors[len(colors)-1] {
		intervals--
	}
	const segments = 160
	for i := 0; i < segments; i++ {
		at := float64(i) / segments
		x, y := beamPoint(r, at)
		x2, y2 := beamPoint(r, float64(i+1)/segments)
		angle := math.Atan2(float64(x-r.X-r.W/2), -float64(y-r.Y-r.H/2)) / (2 * math.Pi)
		position := math.Mod(angle-phase+2, 1)
		mask := beamMask(position)
		if mask == 0 {
			continue
		}
		colorAt := position * float64(intervals)
		index := int(colorAt)
		a, b := inks[index], inks[(index+1)%len(colors)]
		mix := float32(colorAt - math.Floor(colorAt))
		color := ui.RGB(uint8(float32(a.R)*(1-mix)+float32(b.R)*mix), uint8(float32(a.G)*(1-mix)+float32(b.G)*mix), uint8(float32(a.B)*(1-mix)+float32(b.B)*mix))
		if active && style.HueCycle.Enabled {
			color = filterBeamColor(color, hueMatrix)
		}
		path := new(ui.Path).MoveTo(x, y).LineTo(x2, y2)
		alpha := float32(mask)
		// Broad translucent strokes provide the inner wash and bloom before
		// the narrow travelling edge, without a webview or shader dependency.
		// Keep their existing local coverage, multiplied by each element's
		// independent CSS opacity instead of a shared fade/dynamic factor.
		p.Clip(r, 28, func() { p.StrokePath(path, 14, color.Alpha((.08/.42)*alpha*float32(frame.opacity.inner))) })
		if frame.opacity.bloom > 0 && style.GlowBlur > 0 {
			p.StrokePath(path, style.TrackWidth+style.GlowBlur, color.Alpha((.07/.36)*alpha*float32(frame.opacity.bloom)))
			p.StrokePath(path, style.TrackWidth+style.GlowBlur/2, color.Alpha((.12/.36)*alpha*float32(frame.opacity.bloom)))
		}
		p.StrokePath(path, style.TrackWidth, color.Alpha(alpha*float32(frame.opacity.stroke)))
	}
}

// CSS applies hue-rotate, brightness(1.3), then saturate(1.2). The existing
// broad-stroke layers still approximate the source gradients and blur.
func filterBeamColor(color ui.Color, matrix [9]float64) ui.Color {
	color = applyBeamHue(color, matrix)
	r, g, b := min(255, float64(color.R)*1.3), min(255, float64(color.G)*1.3), min(255, float64(color.B)*1.3)
	luma := .213*r + .715*g + .072*b
	channel := func(value float64) uint8 { return uint8(min(255, max(0, luma+(value-luma)*1.2))) }
	return ui.RGB(channel(r), channel(g), channel(b))
}

func beamMask(t float64) float64 {
	stops := [][2]float64{{0, 0}, {.3, 0}, {.36, .1}, {.44, .35}, {.52, 1}, {.8, 1}, {.86, .35}, {.92, .1}, {.96, 0}, {1, 0}}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i][0] {
			a, b := stops[i-1], stops[i]
			return a[1] + (b[1]-a[1])*(t-a[0])/(b[0]-a[0])
		}
	}
	return 0
}

// beamPoint walks the actual rounded card perimeter clockwise.
func beamPoint(r ui.Rect, progress float64) (float32, float32) {
	radius := float64(min(float32(28), min(r.W, r.H)/2))
	w, h := float64(r.W)-2*radius, float64(r.H)-2*radius
	arc := math.Pi * radius / 2
	distance := progress * (2*w + 2*h + 4*arc)
	lengths := []float64{w, arc, h, arc, w, arc, h, arc}
	for index, length := range lengths {
		if distance <= length || index == 7 {
			x, y := float64(r.X), float64(r.Y)
			switch index {
			case 0:
				return float32(x + radius + distance), r.Y
			case 2:
				return r.X + r.W, float32(y + radius + distance)
			case 4:
				return float32(x + radius + w - distance), r.Y + r.H
			case 6:
				return r.X, float32(y + radius + h - distance)
			}
			corner := (index - 1) / 2
			cx := []float64{x + radius + w, x + radius + w, x + radius, x + radius}[corner]
			cy := []float64{y + radius, y + radius + h, y + radius + h, y + radius}[corner]
			angle := float64(corner-1)*math.Pi/2 + distance/radius
			return float32(cx + math.Cos(angle)*radius), float32(cy + math.Sin(angle)*radius)
		}
		distance -= length
	}
	return r.X + float32(radius), r.Y
}

func rotateBeamHue(color ui.Color, degrees float64) ui.Color {
	return applyBeamHue(color, beamHueMatrix(degrees))
}

func beamHueMatrix(degrees float64) [9]float64 {
	cosine, sine := math.Cos(degrees*math.Pi/180), math.Sin(degrees*math.Pi/180)
	return [9]float64{
		.213 + .787*cosine - .213*sine, .715 - .715*cosine - .715*sine, .072 - .072*cosine + .928*sine,
		.213 - .213*cosine + .143*sine, .715 + .285*cosine + .140*sine, .072 - .072*cosine - .283*sine,
		.213 - .213*cosine - .787*sine, .715 - .715*cosine + .715*sine, .072 + .928*cosine + .072*sine,
	}
}

func applyBeamHue(color ui.Color, matrix [9]float64) ui.Color {
	r, g, b := float64(color.R), float64(color.G), float64(color.B)
	channel := func(value float64) uint8 { return uint8(min(255, max(0, value))) }
	return ui.RGB(
		channel(matrix[0]*r+matrix[1]*g+matrix[2]*b),
		channel(matrix[3]*r+matrix[4]*g+matrix[5]*b),
		channel(matrix[6]*r+matrix[7]*g+matrix[8]*b),
	)
}
