package composer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
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
	style.Period = min(60, max(.8, style.Period))
	style.Intensity = min(140, max(40, style.Intensity))
	style.Bloom = min(160, max(0, style.Bloom))
	style.Hue = min(180, max(-180, style.Hue))
	style.TrackWidth = min(4, max(.5, style.TrackWidth))
	style.GlowBlur = min(12, max(0, style.GlowBlur))
	style.Breathing.Amplitude = min(.8, max(0, style.Breathing.Amplitude))
	style.Breathing.Period = min(20, max(1, style.Breathing.Period))
	style.HueCycle.Range = min(360, max(0, style.HueCycle.Range))
	style.HueCycle.Period = min(60, max(1, style.HueCycle.Period))
	style.Night.Dim = min(100, max(0, style.Night.Dim))
	var values struct {
		Night map[string]any `json:"night"`
	}
	_ = json.Unmarshal(raw, &values)
	for key, out := range map[string]*int{"fromMinute": &style.Night.FromMinute, "toMinute": &style.Night.ToMinute} {
		if value, ok := values.Night[key].(float64); ok {
			*out = int(math.Round(min(1439, max(0, value))))
		}
	}
	if style.Direction != "clockwise" && style.Direction != "counterclockwise" && style.Direction != "pingPong" {
		style.Direction = "clockwise"
	}
	if style.Easing != "linear" && style.Easing != "ease" && style.Easing != "ease-in-out" && style.Easing != "cubic-bezier" {
		style.Easing = "linear"
	}
	valid := style.Palette.Kind == "preset" && len(beamPalettes[style.Palette.ID]) > 0
	if style.Palette.Kind == "custom" {
		colors := style.Palette.Colors[:0]
		for _, color := range style.Palette.Colors {
			if effectHexColor(color) {
				colors = append(colors, strings.ToLower(color))
			}
		}
		style.Palette.Colors = colors
		valid = len(colors) >= 2 && len(colors) <= 6
	}
	if !valid {
		style.Palette.Kind, style.Palette.ID, style.Palette.Colors = "preset", "legacy", nil
	}
	return style
}

func effectHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

var beamPalettes = map[string][]ui.Color{
	"legacy":  {design.TW.Rose500, design.TW.Blue500, design.TW.Green500, design.TW.Teal500, design.TW.Violet500, design.TW.Orange500, design.TW.Fuchsia500},
	"warm":    {design.TW.Rose500, design.TW.Orange400, design.TW.Amber300, design.TW.Rose500, design.TW.Rose500},
	"aurora":  {design.TW.Teal400, design.TW.Blue500, design.TW.Violet500, design.TW.Fuchsia400, design.TW.Teal400},
	"rose":    {design.TW.Pink500, design.TW.Pink400, design.TW.Purple400, design.TW.Indigo500, design.TW.Pink500},
	"glacier": {design.TW.Cyan100, design.TW.Sky300, design.TW.Blue500, design.TW.Indigo500, design.TW.Cyan100},
	"cyber":   {design.TW.Cyan400, design.TW.Emerald400, design.TW.Lime400, design.TW.Yellow400, design.TW.Cyan400},
	"sunset":  {design.TW.Rose400, design.TW.Amber300, design.TW.Rose300, design.TW.Pink400, design.TW.Rose400},
	"cyan":    {design.TW.Cyan400, design.TW.Teal400, design.TW.Blue400, design.TW.Cyan400},
}

func formatEffectHex(c ui.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func opaqueRGB(r, g, b uint8) ui.Color {
	var out ui.Color
	out.R, out.G, out.B, out.A = r, g, b, 255
	return out
}

// BeamPaletteColors returns the configured stops for the settings color editor.
func BeamPaletteColors(raw json.RawMessage) []string {
	style := readBeamStyle(raw)
	if style.Palette.Kind == "custom" {
		return append([]string(nil), style.Palette.Colors...)
	}
	swatches := beamPalettes[style.Palette.ID]
	out := make([]string, len(swatches))
	for i, c := range swatches {
		out[i] = formatEffectHex(c)
	}
	return out
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
	var inks []ui.Color
	if style.Palette.Kind == "custom" {
		// Normalized custom palettes contain 2..6 colors; presets at most seven.
		// Keep the common paint path on the stack and retain larger raw palettes
		// without allocating a fresh color slice on every animation frame.
		var palette [7]ui.Color
		inks = palette[:]
		if len(style.Palette.Colors) > len(palette) {
			if cap(timeline.palette) < len(style.Palette.Colors) {
				timeline.palette = make([]ui.Color, len(style.Palette.Colors))
			}
			inks = timeline.palette[:len(style.Palette.Colors)]
		}
		for i, value := range style.Palette.Colors {
			inks[i] = theme.Color(value)
		}
		inks = inks[:len(style.Palette.Colors)]
	} else {
		inks = beamPalettes[style.Palette.ID]
		if len(inks) == 0 {
			inks = beamPalettes["legacy"]
		}
	}
	// Source closes only open palettes. Already closed presets use n-1
	// intervals; an extra repeated first color would shift every stop.
	intervals := len(inks)
	if len(inks) > 0 && inks[0] == inks[len(inks)-1] {
		intervals--
	}
	timeline.raster.paint(p, r, style, inks, intervals, phase, hue, active && style.HueCycle.Enabled, frame.opacity)
}

// CSS applies hue-rotate, brightness(1.3), then saturate(1.2).
func filterBeamColor(color ui.Color, matrix [9]float64) ui.Color {
	color = applyBeamHue(color, matrix)
	r, g, b := min(255, float64(color.R)*1.3), min(255, float64(color.G)*1.3), min(255, float64(color.B)*1.3)
	luma := .213*r + .715*g + .072*b
	channel := func(value float64) uint8 { return uint8(min(255, max(0, luma+(value-luma)*1.2))) }
	return opaqueRGB(channel(r), channel(g), channel(b))
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
	return opaqueRGB(
		channel(matrix[0]*r+matrix[1]*g+matrix[2]*b),
		channel(matrix[3]*r+matrix[4]*g+matrix[5]*b),
		channel(matrix[6]*r+matrix[7]*g+matrix[8]*b),
	)
}
