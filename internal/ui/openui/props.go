package openui

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/design"
)

// Prop readers. Evaluated props are validated against the library schema,
// but a renderer still reads them defensively: a wrong type is "absent".

func str(el *lang.Element, name string) string {
	switch v := el.Prop(name).(type) {
	case string:
		return v
	case float64, bool:
		return lang.ToString(v)
	}
	return ""
}

func num(el *lang.Element, name string) (float64, bool) {
	v, ok := el.Prop(name).(float64)
	if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func numOr(el *lang.Element, name string, def float64) float64 {
	if v, ok := num(el, name); ok {
		return v
	}
	return def
}

func boolean(el *lang.Element, name string) bool {
	b, _ := el.Prop(name).(bool)
	return b
}

func list(el *lang.Element, name string) []lang.Value {
	l, _ := el.Prop(name).([]lang.Value)
	return l
}

// spacing converts the language's 4 DIP units, clamped to a sane range.
func spacing(v float64) float32 {
	return design.Sp(float32(max(0, min(v, 24))))
}

// pixels reads a px number or a "123px" length; percentages and other CSS
// lengths are not fixed sizes here.
func pixels(v lang.Value) (float32, bool) {
	switch t := v.(type) {
	case float64:
		if t > 0 && t < 4096 {
			return float32(t), true
		}
	case string:
		s := strings.TrimSuffix(strings.TrimSpace(t), "px")
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 && f < 4096 {
			return float32(f), true
		}
	}
	return 0, false
}

func radius(v lang.Value, def float32) float32 {
	switch t := v.(type) {
	case float64:
		return float32(max(0, min(t, 64)))
	case string:
		switch t {
		case "sm":
			return design.RadiusSM
		case "md":
			return design.RadiusMD
		case "lg":
			return design.RadiusLG
		case "xl":
			return design.RadiusXL
		case "full":
			return design.RadiusFull
		}
	}
	return def
}

func align(s string, def ui.Align) ui.Align {
	switch s {
	case "start":
		return ui.Start
	case "center":
		return ui.Center
	case "end":
		return ui.End
	case "stretch":
		return ui.Stretch
	case "between":
		return ui.SpaceBetween
	}
	return def
}

// options normalizes Select/RadioGroup/SegmentedControl options: strings
// or {value, label, description} objects.
type option struct{ value, label, description string }

func options(v lang.Value) []option {
	l, _ := v.([]lang.Value)
	out := make([]option, 0, len(l))
	for _, it := range l {
		switch t := it.(type) {
		case string:
			out = append(out, option{t, t, ""})
		case float64, bool:
			s := lang.ToString(t)
			out = append(out, option{s, s, ""})
		case *lang.Object:
			o := option{value: lang.ToString(t.Value("value")), label: lang.ToString(t.Value("label"))}
			if t.Value("value") == nil {
				o.value = o.label
			}
			if t.Value("label") == nil {
				o.label = o.value
			}
			if d, ok := t.Value("description").(string); ok {
				o.description = d
			}
			out = append(out, o)
		}
	}
	return out
}

// tone is a theme-colored role named by the language (Badge color,
// Progress color, Box background, Text color).
func (r *Render) tone(name string) (ui.Color, bool) {
	t := r.T
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "info", "blue", "accent":
		return t.Info, true
	case "success", "green":
		return t.Success, true
	case "warning", "warn", "amber", "yellow", "orange":
		return t.Warning, true
	case "danger", "error", "red", "destructive":
		return t.Destructive, true
	case "primary", "brand":
		return t.Primary, true
	case "purple", "violet":
		return t.Chart4, true
	case "pink":
		return t.Chart5, true
	case "neutral", "gray", "grey", "muted", "secondary":
		return t.MutedForeground, true
	case "foreground", "default", "text":
		return t.Foreground, true
	}
	return ui.Color{}, false
}

// color resolves a named tone or a CSS hex/rgb color supplied by the
// program (data, not a design literal).
func (r *Render) color(name string) (ui.Color, bool) {
	if c, ok := r.tone(name); ok {
		return c, true
	}
	return parseCSSColor(name)
}

// softColor is a named background: a soft tint of the tone.
func (r *Render) softColor(name string) (ui.Color, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "muted", "neutral", "gray", "grey", "secondary":
		return r.T.Muted, true
	case "card", "surface":
		return r.T.Card, true
	}
	if c, ok := r.tone(name); ok {
		return c.Alpha(0.1), true
	}
	return parseCSSColor(name)
}

// parseCSSColor reads #rgb, #rgba, #rrggbb, #rrggbbaa and rgb()/rgba().
func parseCSSColor(s string) (ui.Color, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	var c ui.Color
	hex := func(x string) (uint8, bool) {
		v, err := strconv.ParseUint(x, 16, 8)
		return uint8(v), err == nil
	}
	if strings.HasPrefix(s, "#") {
		h := s[1:]
		if len(h) == 3 || len(h) == 4 {
			var b strings.Builder
			for _, ch := range h {
				b.WriteRune(ch)
				b.WriteRune(ch)
			}
			h = b.String()
		}
		if len(h) != 6 && len(h) != 8 {
			return c, false
		}
		var ok [4]bool
		c.R, ok[0] = hex(h[0:2])
		c.G, ok[1] = hex(h[2:4])
		c.B, ok[2] = hex(h[4:6])
		c.A, ok[3] = 255, true
		if len(h) == 8 {
			c.A, ok[3] = hex(h[6:8])
		}
		return c, ok[0] && ok[1] && ok[2] && ok[3]
	}
	// rgb(…) and rgba(…): the function name, then the channel list.
	if body, ok := strings.CutPrefix(s, "rgb"); ok {
		body = strings.TrimPrefix(body, "a")
		if strings.HasPrefix(body, "(") && strings.HasSuffix(body, ")") {
			parts := strings.FieldsFunc(body[1:len(body)-1], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
			if len(parts) < 3 {
				return c, false
			}
			ch := [4]float64{0, 0, 0, 1}
			for i := 0; i < len(parts) && i < 4; i++ {
				v, err := strconv.ParseFloat(strings.TrimSuffix(parts[i], "%"), 64)
				if err != nil {
					return c, false
				}
				if strings.HasSuffix(parts[i], "%") {
					if i < 3 {
						v = v * 255 / 100
					} else {
						v /= 100
					}
				}
				ch[i] = v
			}
			c.R, c.G, c.B = uint8(max(0, min(ch[0], 255))), uint8(max(0, min(ch[1], 255))), uint8(max(0, min(ch[2], 255)))
			c.A = uint8(max(0, min(ch[3], 1)) * 255)
			return c, true
		}
	}
	return c, false
}

// ── text helpers ──

func (r *Render) text(s string, ts design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(r.C, s).FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(weight).TextColor(col).SingleLine()
}

func (r *Render) paragraph(s string, ts design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(r.C, s).FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(weight).TextColor(col).Selectable()
}

func textStyle(size string, def design.TextStyle) design.TextStyle {
	switch size {
	case "xs":
		return design.TextXS
	case "sm":
		return design.TextSM
	case "md", "base":
		return design.TextBase
	case "lg":
		return design.TextLG
	case "xl":
		return design.TextXL
	case "2xl":
		return design.TextXL2
	case "3xl":
		return design.TextXL3
	}
	return def
}

func weight(s string, def int) int {
	switch s {
	case "normal":
		return design.WeightNormal
	case "medium":
		return design.WeightMedium
	case "semibold":
		return design.WeightSemibold
	case "bold":
		return design.WeightBold
	}
	return def
}
